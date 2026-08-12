package application

import (
	"context"

	"jago-bahe-backend/internal/resolution/domain"
)

// GetCase returns a single case owned by the caller (with its plan, updates,
// evidence, and obstacles).
type GetCase struct {
	repo domain.Repository
}

// NewGetCase wires the use case.
func NewGetCase(r domain.Repository) *GetCase { return &GetCase{repo: r} }

// Execute loads the case and enforces ownership.
func (uc *GetCase) Execute(ctx context.Context, caseID, officialID string) (*domain.Case, error) {
	return loadOwnedCase(ctx, uc.repo, caseID, officialID)
}
