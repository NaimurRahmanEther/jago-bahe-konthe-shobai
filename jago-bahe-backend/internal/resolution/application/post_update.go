package application

import (
	"context"
	"strings"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// PostUpdate appends a weekly timeline entry. A "progress" update moves the case
// to InProgress (resuming work from Planned, Blocked, or Reopened) and mirrors
// that onto the problem; a "blocker" note is a plain log that requires a workable
// state but doesn't change status (a formal typed obstacle uses ReportObstacle).
type PostUpdate struct {
	repo     domain.Repository
	problems Problems
	audit    auditdomain.Repository
}

// NewPostUpdate wires the use case.
func NewPostUpdate(r domain.Repository, p Problems, audit auditdomain.Repository) *PostUpdate {
	return &PostUpdate{repo: r, problems: p, audit: audit}
}

// Execute records the update.
func (uc *PostUpdate) Execute(ctx context.Context, caseID, officialID string, kind domain.UpdateKind, text string) (*domain.Case, error) {
	if !kind.Valid() {
		return nil, domain.ErrInvalidKind
	}
	if strings.TrimSpace(text) == "" {
		return nil, domain.ErrEmptyText
	}
	c, err := loadOwnedCase(ctx, uc.repo, caseID, officialID)
	if err != nil {
		return nil, err
	}

	action := "blocker_note"
	if kind == domain.UpdateProgress {
		next, err := domain.Transition(c.Status, domain.EventProgress, domain.Guards{})
		if err != nil {
			return nil, err
		}
		c.Status = next
		action = "progress_update"
	} else if !domain.IsWorkable(c.Status) {
		return nil, domain.ErrIllegalTransition
	}

	c.Updates = append(c.Updates, domain.NewProgressUpdate(c.ID, kind, text))
	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}

	if kind == domain.UpdateProgress {
		_ = uc.problems.SetStatus(ctx, c.ProblemID, string(domain.StatusInProgress))
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, officialID, action, ""))
	return c, nil
}
