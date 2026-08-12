// Package application (area) exposes the seat's geography as a use case. It is a
// directory read and nothing more: the areas table is reference data, written by
// migrations and never by the API.
package application

import (
	"context"

	"jago-bahe-backend/internal/shared/area/domain"
)

// ListAreas returns the seat's whole geography.
//
// A thin wrapper over the repository, mirroring identity's ListOfficials: there
// is no rule to enforce here, and inventing a service layer for a directory read
// would be ceremony. The ordering the client depends on is the repository's
// (level, then id), so a select renders the same way on every load.
type ListAreas struct {
	areas domain.Repository
}

// NewListAreas wires the use case.
func NewListAreas(a domain.Repository) *ListAreas {
	return &ListAreas{areas: a}
}

// Execute lists every area in the seat.
//
// Unfiltered by design. One seat is bounded — nine rows today, roughly fifty at
// full ward granularity — so the client filters the one payload client-side
// rather than the API growing ?level= / ?parent= query surface it does not need.
func (uc *ListAreas) Execute(ctx context.Context) ([]domain.Area, error) {
	return uc.areas.List(ctx)
}
