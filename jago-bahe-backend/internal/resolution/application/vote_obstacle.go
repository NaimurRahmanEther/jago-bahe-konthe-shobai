package application

import (
	"context"

	"jago-bahe-backend/internal/resolution/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// VoteObstacle records a resident's advisory "is this obstacle real?" vote on a
// blocker. It is PRESSURE ONLY: it moves the tally but never changes the case
// status or the scorecard verdict (the named authority adjudicates in B6). Only
// verified area residents may vote, one vote per resident per blocker.
type VoteObstacle struct {
	repo     domain.Repository
	areas    areadomain.Repository
	problems Problems
	identity Identity
	audit    auditdomain.Repository
}

// NewVoteObstacle wires the use case.
func NewVoteObstacle(r domain.Repository, areas areadomain.Repository, p Problems, id Identity, audit auditdomain.Repository) *VoteObstacle {
	return &VoteObstacle{repo: r, areas: areas, problems: p, identity: id, audit: audit}
}

// Execute records the vote and returns the blocker with refreshed advisory tallies.
func (uc *VoteObstacle) Execute(ctx context.Context, problemID, voterID string, choice domain.ObstacleVoteChoice) (*domain.Obstacle, error) {
	if !choice.Valid() {
		return nil, domain.ErrInvalidChoice
	}
	if err := ensureAreaResident(ctx, uc.identity, uc.areas, uc.problems, problemID, voterID); err != nil {
		return nil, err
	}
	_, blocker, err := activeObstacle(ctx, uc.repo, problemID)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.AddObstacleVote(ctx, domain.NewObstacleVote(blocker.ID, voterID, choice)); err != nil {
		return nil, err // domain.ErrAlreadyVotedObstacle
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", problemID, voterID, "obstacle_vote", string(choice)))

	// Reload so the returned blocker carries the refreshed advisory counts.
	_, refreshed, err := activeObstacle(ctx, uc.repo, problemID)
	if err != nil {
		return nil, err
	}
	return refreshed, nil
}
