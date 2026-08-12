package application

import (
	"context"

	"jago-bahe-backend/internal/identity/domain"
)

// ListOfficials returns the public officials directory.
type ListOfficials struct {
	officials domain.OfficialRepository
}

// NewListOfficials wires the use case.
func NewListOfficials(o domain.OfficialRepository) *ListOfficials {
	return &ListOfficials{officials: o}
}

// Execute lists all officials.
func (uc *ListOfficials) Execute(ctx context.Context) ([]domain.Official, error) {
	return uc.officials.List(ctx)
}
