package application

import (
	"context"

	"jago-bahe-backend/internal/suggestion/domain"
)

// ListMyUpvotes answers "which of these suggestions have I already upvoted?" for
// one viewer, in one round trip.
//
// A separate use case from ListSuggestions rather than a caller parameter on it,
// for the reason A.3.2 constraint 2 gives about the problem feed: the ranked list
// is derived without reference to who is asking, and an argument that changes what
// a public route returns is one refactor away from being attacker-controlled. This
// takes a viewer and returns only that viewer's own upvotes, so it can be reasoned
// about alone — it can leak nothing about anyone else because it reads nobody
// else's rows.
//
// The result decorates rows the ranking already chose; it never adds, removes or
// reorders one, and it never touches isTop.
type ListMyUpvotes struct {
	suggestions domain.Repository
}

// NewListMyUpvotes wires the use case.
func NewListMyUpvotes(s domain.Repository) *ListMyUpvotes {
	return &ListMyUpvotes{suggestions: s}
}

// Execute returns the viewer's own upvotes per suggestion id. An empty viewerID
// (an anonymous reader) yields an empty map and no query.
func (uc *ListMyUpvotes) Execute(ctx context.Context, viewerID string, suggestionIDs []string) (map[string]bool, error) {
	return uc.suggestions.UpvotesByViewer(ctx, viewerID, suggestionIDs)
}
