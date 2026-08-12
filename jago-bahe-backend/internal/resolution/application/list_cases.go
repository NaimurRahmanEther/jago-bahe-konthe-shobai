package application

import (
	"context"

	"jago-bahe-backend/internal/resolution/domain"
)

// ListCases returns the cases assigned to an official — their public work inbox.
// It materializes a case (status Assigned) for any assignment that has not yet
// produced one, so a freshly assigned problem shows up ready to acknowledge.
type ListCases struct {
	repo        domain.Repository
	assignments Assignments
}

// NewListCases wires the use case.
func NewListCases(r domain.Repository, a Assignments) *ListCases {
	return &ListCases{repo: r, assignments: a}
}

// Execute lists the official's cases.
func (uc *ListCases) Execute(ctx context.Context, officialID string) ([]domain.Case, error) {
	assignments, err := uc.assignments.ListByOfficial(ctx, officialID)
	if err != nil {
		return nil, err
	}
	for _, a := range assignments {
		if _, err := ensureCase(ctx, uc.repo, uc.assignments, a.ProblemID); err != nil {
			return nil, err
		}
	}
	return uc.repo.ListByOfficial(ctx, officialID)
}
