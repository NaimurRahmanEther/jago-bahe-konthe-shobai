package application

import (
	"context"

	"jago-bahe-backend/internal/resolution/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
)

// UpvoteUnblockingPlan toggles a resident's upvote on an unblocking plan (one per
// resident per plan; the presence of the upvote is support). Only verified area
// residents may upvote, mirroring the suggestion rule.
type UpvoteUnblockingPlan struct {
	repo     domain.Repository
	areas    areadomain.Repository
	problems Problems
	identity Identity
}

// NewUpvoteUnblockingPlan wires the use case.
func NewUpvoteUnblockingPlan(r domain.Repository, areas areadomain.Repository, p Problems, id Identity) *UpvoteUnblockingPlan {
	return &UpvoteUnblockingPlan{repo: r, areas: areas, problems: p, identity: id}
}

// Execute toggles the upvote and returns the plan with its refreshed count.
func (uc *UpvoteUnblockingPlan) Execute(ctx context.Context, planID, voterID string) (*domain.UnblockingPlan, error) {
	problemID, err := uc.repo.ProblemForPlan(ctx, planID)
	if err != nil {
		return nil, err // domain.ErrPlanNotFound
	}
	if err := ensureAreaResident(ctx, uc.identity, uc.areas, uc.problems, problemID, voterID); err != nil {
		return nil, err
	}
	return uc.repo.ToggleUnblockingUpvote(ctx, planID, voterID)
}
