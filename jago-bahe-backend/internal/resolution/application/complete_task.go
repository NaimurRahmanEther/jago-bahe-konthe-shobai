package application

import (
	"context"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// CompleteTask marks one of a plan's weekly tasks completed. It is the official's
// public "this week's work is done" — a milestone, not a lifecycle transition, so
// it changes no case status and leaves the evidence-gated path to Done untouched.
// Legal from any workable state (Planned, InProgress, Blocked, Reopened): the plan
// exists and the case is being worked.
type CompleteTask struct {
	repo  domain.Repository
	audit auditdomain.Repository
}

// NewCompleteTask wires the use case.
func NewCompleteTask(r domain.Repository, audit auditdomain.Repository) *CompleteTask {
	return &CompleteTask{repo: r, audit: audit}
}

// Execute marks the task completed on the owning official's case.
func (uc *CompleteTask) Execute(ctx context.Context, caseID, officialID, taskID string) (*domain.Case, error) {
	c, err := loadOwnedCase(ctx, uc.repo, caseID, officialID)
	if err != nil {
		return nil, err
	}
	if !domain.IsWorkable(c.Status) {
		return nil, domain.ErrIllegalTransition
	}
	if err := c.CompleteTask(taskID); err != nil {
		return nil, err
	}
	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, officialID, "task_completed", ""))
	return c, nil
}
