package application

import (
	"context"
	"strings"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// ReportObstacle declares a typed obstacle: it keeps the case open, moves it to
// Blocked, records the blocker (category routes the forward to the named
// authority), and logs a blocker entry on the timeline. Legal only from Planned
// or InProgress. The advisory public judgment happens separately on the problem.
type ReportObstacle struct {
	repo     domain.Repository
	problems Problems
	audit    auditdomain.Repository
	notify   notificationdomain.Repository
}

// NewReportObstacle wires the use case.
func NewReportObstacle(r domain.Repository, p Problems, audit auditdomain.Repository, n notificationdomain.Repository) *ReportObstacle {
	return &ReportObstacle{repo: r, problems: p, audit: audit, notify: n}
}

// Execute blocks the case.
func (uc *ReportObstacle) Execute(ctx context.Context, caseID, officialID string, category domain.ObstacleCategory, whatBlocks, whoUnblocks, proofTried string) (*domain.Case, error) {
	if !category.Valid() {
		return nil, domain.ErrInvalidCategory
	}
	if strings.TrimSpace(whatBlocks) == "" || strings.TrimSpace(whoUnblocks) == "" {
		return nil, domain.ErrEmptyText
	}
	c, err := loadOwnedCase(ctx, uc.repo, caseID, officialID)
	if err != nil {
		return nil, err
	}

	next, err := domain.Transition(c.Status, domain.EventBlock, domain.Guards{})
	if err != nil {
		return nil, err
	}
	c.Status = next
	c.Obstacles = append(c.Obstacles, domain.NewObstacle(c.ID, category, whatBlocks, whoUnblocks, proofTried))
	c.Updates = append(c.Updates, domain.NewProgressUpdate(c.ID, domain.UpdateObstacle, whatBlocks))

	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = uc.problems.SetStatus(ctx, c.ProblemID, string(domain.StatusBlocked))
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, officialID, "blocked", whatBlocks))

	// B21: THE MONITOR, NOT THE NAMED AUTHORITY — and that is a documented deviation
	// from the one notification the design documents promise outright.
	//
	// Concept §8, Scaffold §2 and Backend Plan B5 all say the platform "notifies the
	// named higher authority". It cannot, as written: Obstacle.WhoUnblocks is FREE
	// TEXT (the guard above only checks it is non-empty), so there is no id to write
	// a row to. The addressable party is Case.MonitorOfficialID — "the tier notified
	// on obstacles/escalation", domain/case.go — so that is who is told, and
	// whoUnblocks travels as Detail so the monitor sees who the official says can
	// actually unblock it. Making the promise literal means a typed authority
	// reference on the obstacle, which is its own phase, not a quiet widening of
	// this one. See CLAUDE.md A.3.9 constraint 5.
	//
	// MonitorOfficialID is "" when the assignee is at the top of the ladder — the MP
	// has nobody above them, so MonitorFor returns "". Append refuses an empty
	// recipient and nothing is written, which is honest. DO NOT "fix" that by
	// dropping the guard: a blank recipient row matches every caller who passes an
	// empty officialId, which is every resident and admin in the seat.
	//
	// Entitlement: the monitor already sees this case continuously from assignment
	// via ListByMonitor (A.3.7 rule 4), so this discloses nothing new.
	_ = uc.notify.Append(ctx, notificationdomain.ForOfficial(
		c.MonitorOfficialID, notificationdomain.TypeObstacleDeclared, c.ProblemID,
		notificationTitle(ctx, uc.problems, c.ProblemID), whoUnblocks))
	return c, nil
}
