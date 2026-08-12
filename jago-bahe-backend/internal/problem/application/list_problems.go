package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
)

// ListProblems returns the public problem feed, optionally filtered.
type ListProblems struct {
	problems domain.Repository
}

// NewListProblems wires the use case.
func NewListProblems(p domain.Repository) *ListProblems {
	return &ListProblems{problems: p}
}

// Execute lists publicly visible problems matching the filter (area / status /
// official).
//
// It derives its own status set rather than trusting the caller's: an absent
// filter becomes PublicStatuses(), and a requested one must be a known status or
// be refused. Nothing is hidden from this feed today, so the check no longer
// withholds anything — it rejects an unrecognised ?status= outright instead of
// quietly returning nothing, and it is the one place a future hidden state would
// be excluded. Keeping the derivation here is also what lets this use case stay
// provable on its own: it takes no caller, so it cannot accidentally grow one.
func (uc *ListProblems) Execute(ctx context.Context, areaID, status, officialID string) ([]domain.Problem, error) {
	statuses := domain.PublicStatuses()
	if status != "" {
		s := domain.Status(status)
		if !s.PubliclyVisible() {
			return nil, domain.ErrStatusNotPublic
		}
		statuses = []domain.Status{s}
	}

	return uc.problems.List(ctx, domain.Filter{
		AreaID:     areaID,
		OfficialID: officialID,
		Statuses:   statuses,
	})
}
