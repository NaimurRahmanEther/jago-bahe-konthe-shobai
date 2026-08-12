package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
)

// ListMyVotes answers "which of these problems have I already validated?" for one
// viewer, in one round trip.
//
// It is a separate use case from ListProblems and GetProblem rather than a caller
// parameter on either, for the reason A.3.2 constraint 2 gives: the public feed
// derives what it returns without reference to who is asking, and an argument that
// changes what a public route returns is one refactor away from being
// attacker-controlled. This use case takes a viewer and returns only that viewer's
// own votes, so it can be reasoned about on its own — it can leak nothing about
// anyone else because it never reads anyone else's rows.
//
// The result decorates rows the feed already selected; it never adds, removes or
// reorders one.
type ListMyVotes struct {
	problems domain.Repository
}

// NewListMyVotes wires the use case.
func NewListMyVotes(p domain.Repository) *ListMyVotes {
	return &ListMyVotes{problems: p}
}

// Execute returns the viewer's own vote per problem id. An empty viewerID (an
// anonymous reader) yields an empty map and no query.
func (uc *ListMyVotes) Execute(ctx context.Context, viewerID string, problemIDs []string) (map[string]domain.VoteChoice, error) {
	return uc.problems.VotesByViewer(ctx, viewerID, problemIDs)
}
