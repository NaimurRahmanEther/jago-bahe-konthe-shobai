package application

import (
	"context"

	"jago-bahe-backend/internal/identity/domain"
)

// GetOfficial returns a single directory entry by id.
type GetOfficial struct {
	officials domain.OfficialRepository
}

// NewGetOfficial wires the use case.
func NewGetOfficial(o domain.OfficialRepository) *GetOfficial {
	return &GetOfficial{officials: o}
}

// Execute returns the official, or domain.ErrOfficialNotFound.
func (uc *GetOfficial) Execute(ctx context.Context, id string) (*domain.Official, error) {
	return uc.officials.GetByID(ctx, id)
}
