package domain

import (
	"context"
	"errors"
)

// Sentinel errors mapped to HTTP status by the http layer.
var (
	ErrSuggestionNotFound = errors.New("suggestion not found")
	ErrProblemNotFound    = errors.New("problem not found")
	ErrEmptyText          = errors.New("suggestion text is required")
	ErrNotVerified        = errors.New("resident is not verified")
	ErrNotAreaResident    = errors.New("resident is not in this problem's area")
	ErrAreaNotFound       = errors.New("area not found")
)

// Repository is the port for suggestion persistence. It also owns the
// suggestions' upvotes (part of the aggregate). UpvoteCount is computed live by
// the implementation on read, so callers always see a current tally.
type Repository interface {
	Create(ctx context.Context, s *Suggestion) error
	GetByID(ctx context.Context, id string) (*Suggestion, error)
	ListByProblem(ctx context.Context, problemID string) ([]Suggestion, error)

	// HasUpvoted reports whether this voter already upvoted this suggestion.
	HasUpvoted(ctx context.Context, suggestionID, voterID string) (bool, error)
	// UpvotesByViewer returns the ids, among those given, that this viewer has
	// upvoted. One query for a whole list, so the UI can render an upvote already
	// cast instead of re-offering the button on a reload — the same problem
	// problemDTO.myVote exists to solve, and the same shape
	// ProblemRepository.VotesByViewer solves it with.
	//
	// It returns only the viewer's OWN rows. The counts are public; who upvoted
	// what is not.
	UpvotesByViewer(ctx context.Context, viewerID string, suggestionIDs []string) (map[string]bool, error)
	// AddUpvote records an upvote; it is idempotent under the distinctness
	// invariant (one upvote per resident per suggestion).
	AddUpvote(ctx context.Context, v SuggestionVote) error
	// RemoveUpvote withdraws this voter's upvote (the toggle-off path).
	RemoveUpvote(ctx context.Context, suggestionID, voterID string) error
}
