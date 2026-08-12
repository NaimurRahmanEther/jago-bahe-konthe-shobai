package application

import (
	"context"
	"strings"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// RevisePlan lets the official post a fresh plan to restart work on a case that
// stalled — after an obstacle was resolved (InProgress) or a resident sent it back
// (Reopened). It exists because SubmitPlan is legal only once (Acknowledged →
// Planned) and overwrites the single Plan, so without this a reopened or unblocked
// case is stuck forever with a plan it has already outgrown.
//
// The restart is put on the public record three ways: the new Plan itself, a
// "Plan revised" entry appended to the weekly timeline (carrying the official's
// reason), and a plan_revised audit entry. The prior plan's text is not archived
// — the pivot and its reason are what the record needs, and the obstacle that
// caused it persists on the case (surfaced by the public progress view).
//
// It requires the same reason a resumption does: a fresh plan is active work, so
// the case resolves to InProgress and the problem status is mirrored to match
// (never regressed to Planned, which would leave the two disagreeing).
type RevisePlan struct {
	repo        domain.Repository
	suggestions Suggestions
	problems    Problems
	audit       auditdomain.Repository
}

// NewRevisePlan wires the use case.
func NewRevisePlan(r domain.Repository, s Suggestions, p Problems, audit auditdomain.Repository) *RevisePlan {
	return &RevisePlan{repo: r, suggestions: s, problems: p, audit: audit}
}

// Execute validates and stores the replacement plan. reason is the official's
// public explanation of what changed; it is required, like the plan itself.
func (uc *RevisePlan) Execute(ctx context.Context, caseID, officialID, strategy string, tasks []string, obstacles, suggestionResponse, reason string) (*domain.Case, error) {
	c, err := loadOwnedCase(ctx, uc.repo, caseID, officialID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(strategy) == "" || strings.TrimSpace(suggestionResponse) == "" || strings.TrimSpace(reason) == "" {
		return nil, domain.ErrEmptyText // strategy, the suggestion answer, and the restart reason are all required
	}
	cleaned := make([]string, 0, len(tasks))
	for _, tk := range tasks {
		if strings.TrimSpace(tk) == "" {
			return nil, domain.ErrEmptyText // every listed week must carry a task
		}
		cleaned = append(cleaned, strings.TrimSpace(tk))
	}
	if len(cleaned) == 0 {
		return nil, domain.ErrEmptyText // a plan needs at least one week
	}

	// Gate on the lifecycle: replan is legal only from InProgress and Reopened.
	next, err := domain.Transition(c.Status, domain.EventReplan, domain.Guards{})
	if err != nil {
		return nil, err
	}

	// Re-snapshot the top suggestion at revise time: the community's priorities may
	// have moved since the first plan, and the new plan must answer what leads now
	// (same rule as SubmitPlan — never re-ranked later).
	top, err := uc.suggestions.Top(ctx, c.ProblemID)
	if err != nil {
		return nil, err
	}

	c.Status = next
	c.Plan = domain.NewPlan(c.ID, strategy, cleaned, obstacles, suggestionResponse, top.ID, top.Text)
	c.Updates = append(c.Updates, domain.NewProgressUpdate(c.ID, domain.UpdateProgress, "Plan revised: "+strings.TrimSpace(reason)))

	// Replace the plan first (Save's plan upsert is write-once and would drop the
	// new plan while inserting its tasks against it — an FK violation). Save then
	// persists the resumed status and the "Plan revised" note.
	if err := uc.repo.ReplacePlan(ctx, c); err != nil {
		return nil, err
	}
	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}

	_ = uc.problems.SetStatus(ctx, c.ProblemID, string(domain.StatusInProgress))
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, officialID, "plan_revised", strings.TrimSpace(reason)))
	return c, nil
}
