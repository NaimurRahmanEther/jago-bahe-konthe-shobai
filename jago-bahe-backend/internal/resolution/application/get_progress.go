package application

import (
	"context"
	"errors"

	"jago-bahe-backend/internal/resolution/domain"
)

// GetProgress returns the public progress of the case behind a problem: the
// plan the official published, the updates they posted, the evidence, and the
// fairness flag. It is a public read — the same plan is already published per
// official by the record (B10), so serving it per problem discloses nothing new,
// and reporter-only would be incoherent rather than protective.
//
// It never reads the problem row, and that omission is the security property.
// Everything it cannot answer collapses to one indistinguishable "no case":
//
//   - the problem does not exist
//   - the problem is public but not yet assigned (the common case: everything in
//     the feed that has not yet reached V and been routed to an official)
//   - the problem is assigned but no official has looked at it yet, so the case
//     has not been materialized (cases are created lazily, on first read)
//
// Gating this on the reporter would mean loading the problem to check it, and then
// "not found" and "found but withheld" become separately observable — an oracle
// over ids for no benefit, since the plan behind an assigned case is public per
// official anyway. GetObstacle already behaves this way; this follows it
// (CLAUDE.md A.3.2).
type GetProgress struct {
	repo domain.Repository
}

// NewGetProgress wires the use case.
func NewGetProgress(r domain.Repository) *GetProgress {
	return &GetProgress{repo: r}
}

// Execute returns the case behind a problem, or (nil, false) when there is none
// to report. A missing case is never an error.
func (uc *GetProgress) Execute(ctx context.Context, problemID string) (*domain.Case, bool, error) {
	c, err := uc.repo.GetByProblem(ctx, problemID)
	if errors.Is(err, domain.ErrCaseNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return c, true, nil
}
