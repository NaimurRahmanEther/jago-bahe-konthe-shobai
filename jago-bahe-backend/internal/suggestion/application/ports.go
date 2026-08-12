// Package application holds the suggestion use cases. It orchestrates the
// suggestion domain, the shared area geography, the audit log, and two small
// ports — Identity (voter eligibility) and Problems (the problem's area) — so the
// suggestion context stays decoupled from the identity and problem contexts.
package application

import (
	"context"

	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// Voter is the subset of an account the suggestion context needs to decide who
// may propose or upvote.
type Voter struct {
	IsResident bool
	Verified   bool
	UnionID    string
}

// Identity is the port into the identity context. The composition root supplies
// an adapter over the identity repositories.
type Identity interface {
	Voter(ctx context.Context, accountID string) (Voter, error)
}

// Problems is the port into the problem context: it resolves the area a problem
// belongs to (for the area-residency check) and, via its error, the problem's
// existence. The composition root adapts the problem repository.
type Problems interface {
	// AreaID returns the problem's area, or domain.ErrProblemNotFound if absent.
	AreaID(ctx context.Context, problemID string) (valueobject.AreaID, error)
}
