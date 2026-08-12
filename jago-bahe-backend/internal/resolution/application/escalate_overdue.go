package application

import (
	"context"
	"fmt"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// EscalateOverdue is the deterministic escalation scan run by cmd/worker. It does
// two things per non-terminal case, on two deliberately separate clocks:
//
//  1. LATENESS. It computes the visibility level the case should have climbed to —
//     one rung per elapsed deadline window past the anchor (the first-response
//     deadline for a silent case, or the blocker review-window cap for a stalled
//     blocker) — and raises cases.escalation_level if higher, auditing each hop.
//     This clock only ever rises, which is what makes the level an honest record of
//     how late a case ever got.
//
//  2. SILENCE (B19). It opens and closes the OBSERVATIONS that put a copy of the
//     case in front of the officials above the assignee, and it reads the silence
//     clock instead: the deadline, re-anchored on the official's last act (see
//     domain.SilenceLevel). An official who is posting weekly updates on a late case
//     is not silent, and must not accumulate observers.
//
// The work is never moved off the official. Only its visibility climbs (Concept §7).
// Being idempotent, re-running the scan never double-escalates and never opens a
// rung twice.
type EscalateOverdue struct {
	repo                domain.Repository
	audit               auditdomain.Repository
	ladder              Ladder
	deadlineWindow      time.Duration // D
	blockerReviewWindow time.Duration // R
}

// NewEscalateOverdue wires the use case.
func NewEscalateOverdue(r domain.Repository, audit auditdomain.Repository, ladder Ladder, deadlineWindow, blockerReviewWindow time.Duration) *EscalateOverdue {
	return &EscalateOverdue{repo: r, audit: audit, ladder: ladder, deadlineWindow: deadlineWindow, blockerReviewWindow: blockerReviewWindow}
}

// ScanResult reports what one pass changed.
type ScanResult struct {
	Escalated          int // cases whose visibility level rose
	ObservationsOpened int // ladder rungs newly watching a silent case
	ObservationsClosed int // rungs released because the official answered
}

// Run scans candidates and escalates those overdue as of now.
func (uc *EscalateOverdue) Run(ctx context.Context, now time.Time) (ScanResult, error) {
	var result ScanResult

	cases, err := uc.repo.ListEscalationCandidates(ctx)
	if err != nil {
		return result, err
	}
	if len(cases) == 0 {
		return result, nil
	}

	// One batch query for the whole scan rather than loading every open case's
	// whole aggregate every tick (A.3.4: the scan is a page).
	ids := make([]string, 0, len(cases))
	for i := range cases {
		ids = append(ids, cases[i].ID)
	}
	lastActivity, err := uc.repo.LastActivityByCases(ctx, ids)
	if err != nil {
		return result, err
	}
	observations, err := uc.repo.ListObservationsByCases(ctx, ids)
	if err != nil {
		return result, err
	}
	openByCase := make(map[string][]domain.Observation, len(cases))
	for _, o := range observations {
		if o.IsOpen() {
			openByCase[o.CaseID] = append(openByCase[o.CaseID], o)
		}
	}

	for i := range cases {
		c := &cases[i]

		raised, err := uc.raiseLevel(ctx, c, now)
		if err != nil {
			return result, err
		}
		if raised {
			result.Escalated++
		}

		// The repository's candidate query already filters to non-terminal work, but
		// the rule is stated here too: a finished case must never acquire an
		// observer, and a read side must not depend on a WHERE clause for that.
		if !domain.IsEscalatable(c.Status) {
			continue
		}
		opened, closed, err := uc.syncObservations(ctx, c, openByCase[c.ID], lastActivity[c.ID], now)
		if err != nil {
			return result, err
		}
		result.ObservationsOpened += opened
		result.ObservationsClosed += closed
	}
	return result, nil
}

// raiseLevel is clock 1: unchanged B6 behaviour. It reports whether the level rose.
func (uc *EscalateOverdue) raiseLevel(ctx context.Context, c *domain.Case, now time.Time) (bool, error) {
	anchor, blocked, ok := escalationAnchor(c, uc.blockerReviewWindow)
	if !ok {
		return false, nil
	}
	target := domain.EscalationTarget(anchor, now, uc.deadlineWindow, domain.MaxEscalationLevel)
	if target <= c.EscalationLevel {
		return false, nil
	}
	if err := uc.repo.SetEscalation(ctx, c.ID, target, now); err != nil {
		return false, err
	}
	reason := fmt.Sprintf("no response past deadline; visibility raised to tier %d", target)
	if blocked {
		reason = fmt.Sprintf("blocker unresolved past review window; visibility raised to tier %d", target)
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, auditdomain.ActorSystem, "escalated", reason))
	c.EscalationLevel = target
	return true, nil
}

// syncObservations is clock 2: it puts the case in front of the officials above the
// assignee while they are silent, and releases them when the official answers.
//
// Resolve runs BEFORE open so that a case answered this tick is not simultaneously
// escalated on stale state. A case that is answered and then goes quiet again opens
// rung 1 as a NEW row — the resolved one no longer blocks the partial unique index
// — so a second silence is escalated as fully as the first, which is exactly what
// the level's high-water rule cannot express on its own.
func (uc *EscalateOverdue) syncObservations(ctx context.Context, c *domain.Case, open []domain.Observation, lastActivity, now time.Time) (opened, closed int, err error) {
	if !lastActivity.IsZero() {
		var stillOpen []domain.Observation
		for _, o := range open {
			if o.OpenedAt.Before(lastActivity) {
				continue // answered after this rung opened
			}
			stillOpen = append(stillOpen, o)
		}
		if len(stillOpen) != len(open) {
			n, err := uc.repo.ResolveOpenObservations(ctx, c.ID, now)
			if err != nil {
				return opened, closed, err
			}
			closed += n
			open = nil
			days := int(now.Sub(c.Deadline).Hours() / 24)
			_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, auditdomain.ActorSystem,
				"observation_resolved", fmt.Sprintf("the official responded; %d day(s) past the response deadline", days)))
		}
	}

	target := domain.SilenceLevel(c.Deadline, lastActivity, now, uc.deadlineWindow, domain.MaxEscalationLevel)
	missing := domain.MissingRungs(open, target)
	if len(missing) == 0 {
		return opened, closed, nil
	}

	// The ladder is resolved per case rather than per rung, and a failure here ends
	// this case's escalation without ending the scan: one official with an
	// unresolvable area must not stop every other case in the seat from escalating.
	chain, err := uc.ladder.Chain(ctx, c.OfficialID, domain.MaxEscalationLevel)
	if err != nil || len(chain) == 0 {
		return opened, closed, nil
	}

	for _, level := range missing {
		if level > len(chain) {
			break // the ladder ran out of officials before the rungs ran out
		}
		obs := domain.NewObservation(c.ID, chain[level-1], level, now)
		inserted, err := uc.repo.OpenObservation(ctx, obs)
		if err != nil {
			return opened, closed, err
		}
		if !inserted {
			continue // another pass already opened this rung
		}
		opened++
		silentDays := int(now.Sub(c.Deadline).Hours() / 24)
		_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, auditdomain.ActorSystem,
			"observation_opened", fmt.Sprintf("no response for %d day(s); now visible to the monitor at tier %d", silentDays, level)))
	}
	return opened, closed, nil
}

// escalationAnchor returns the instant a case's escalation clock starts, whether
// it is a blocked-case anchor, and whether escalation applies at all. A silent
// case anchors on its first-response deadline; a Blocked case anchors on the
// blocker's review-window cap (the clock is paused until then), so a blocker can
// never be a permanent parking spot.
func escalationAnchor(c *domain.Case, reviewWindow time.Duration) (time.Time, bool, bool) {
	if c.Status == domain.StatusBlocked {
		b, ok := c.ActiveObstacle()
		if !ok {
			return time.Time{}, false, false
		}
		return b.CreatedAt.Add(reviewWindow), true, true
	}
	if domain.IsEscalatable(c.Status) {
		return c.Deadline, false, true
	}
	return time.Time{}, false, false
}
