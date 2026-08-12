package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
)

// ListMyProblems returns the reports the calling resident filed themselves, at
// every status.
//
// This is deliberately a separate use case from ListProblems rather than a caller
// parameter on it. The two answer different questions — the public feed derives
// PublicStatuses() and ignores whoever is asking, while this view scopes to one
// caller. Folding both into one use case would make the feed's behaviour
// conditional on an argument, and an argument that changes what a public route
// returns is one refactor away from being attacker-controlled. Two questions, two
// use cases, each provable alone (CLAUDE.md A.3.2).
//
// It exists because a report goes straight into a feed of everyone else's: without
// this, the person who filed it has no way to follow it afterwards.
type ListMyProblems struct {
	problems domain.Repository
}

// NewListMyProblems wires the use case.
func NewListMyProblems(p domain.Repository) *ListMyProblems {
	return &ListMyProblems{problems: p}
}

// Execute lists everything reporterID filed, newest first.
//
// The guard comes first, and that ordering is the invariant. Statuses is nil —
// every state, including this reporter's own Rejected — which is correct only
// because ReporterID scopes the query to their own rows. With an empty caller the
// same filter matches every problem in the seat, and this endpoint would answer
// "everything anyone ever filed" to the question "what did I file". So an absent
// caller fails closed rather than querying.
func (uc *ListMyProblems) Execute(ctx context.Context, reporterID string) ([]domain.Problem, error) {
	if reporterID == "" {
		return nil, domain.ErrNoCaller
	}

	return uc.problems.List(ctx, domain.Filter{
		ReporterID: reporterID,
		Statuses:   nil,
	})
}
