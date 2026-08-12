package application

import (
	"context"
	"strings"

	"jago-bahe-backend/internal/resolution/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// ProposeUnblockingPlan records a community-proposed way to overcome a blocker —
// the suggestion engine re-pointed at the blocker. Only verified area residents
// may propose.
type ProposeUnblockingPlan struct {
	repo     domain.Repository
	areas    areadomain.Repository
	problems Problems
	identity Identity
	audit    auditdomain.Repository
}

// NewProposeUnblockingPlan wires the use case.
func NewProposeUnblockingPlan(r domain.Repository, areas areadomain.Repository, p Problems, id Identity, audit auditdomain.Repository) *ProposeUnblockingPlan {
	return &ProposeUnblockingPlan{repo: r, areas: areas, problems: p, identity: id, audit: audit}
}

// Execute stores the unblocking plan.
func (uc *ProposeUnblockingPlan) Execute(ctx context.Context, problemID, authorID, text string) (*domain.UnblockingPlan, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.ErrEmptyText
	}
	if err := ensureAreaResident(ctx, uc.identity, uc.areas, uc.problems, problemID, authorID); err != nil {
		return nil, err
	}
	_, blocker, err := activeObstacle(ctx, uc.repo, problemID)
	if err != nil {
		return nil, err
	}

	plan := domain.NewUnblockingPlan(blocker.ID, authorID, text)
	if err := uc.repo.AddUnblockingPlan(ctx, plan); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", problemID, authorID, "unblocking_plan_proposed", ""))
	return plan, nil
}
