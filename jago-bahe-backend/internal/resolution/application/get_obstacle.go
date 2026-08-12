package application

import (
	"context"
	"errors"

	"jago-bahe-backend/internal/resolution/domain"
)

// GetObstacle returns the active blocker a problem is being publicly judged on
// (with its unblocking plans ranked), or (nil, false) when the problem is not
// blocked. Public read.
type GetObstacle struct {
	repo domain.Repository
	svc  *domain.Service
}

// NewGetObstacle wires the use case.
func NewGetObstacle(r domain.Repository, svc *domain.Service) *GetObstacle {
	return &GetObstacle{repo: r, svc: svc}
}

// Execute returns the active blocker for a problem, if any.
func (uc *GetObstacle) Execute(ctx context.Context, problemID string) (*domain.Obstacle, bool, error) {
	c, err := uc.repo.GetByProblem(ctx, problemID)
	if errors.Is(err, domain.ErrCaseNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	b, ok := c.ActiveObstacle()
	if !ok {
		return nil, false, nil
	}
	b.UnblockingPlans = uc.svc.RankUnblocking(b.UnblockingPlans)
	return b, true, nil
}

// activeObstacle resolves a problem's active blocker or a typed error (used by the
// write paths). ErrObstacleNotFound covers both "no case" and "no open blocker".
func activeObstacle(ctx context.Context, repo domain.Repository, problemID string) (*domain.Case, *domain.Obstacle, error) {
	c, err := repo.GetByProblem(ctx, problemID)
	if errors.Is(err, domain.ErrCaseNotFound) {
		return nil, nil, domain.ErrObstacleNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	b, ok := c.ActiveObstacle()
	if !ok {
		return nil, nil, domain.ErrObstacleNotFound
	}
	return c, b, nil
}
