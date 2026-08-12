// Package domain (suggestion) owns community-proposed fixes and their upvotes.
// Pure logic — no database, HTTP, or other contexts. It copies the problem
// reference slice: the Service is the sole authority for ranking and "top"
// selection; the http/UI layers only display the outcome.
package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// Suggestion is the aggregate root: one resident's proposed fix for a problem.
// UpvoteCount and IsTop are derived (count from the votes, IsTop from the ranking
// service) and populated on read — they are not authoritative stored state.
type Suggestion struct {
	ID          string
	ProblemID   string
	AuthorID    string
	Text        string
	UpvoteCount int
	IsTop       bool
	CreatedAt   time.Time
}

// NewSuggestion constructs a freshly proposed suggestion (no upvotes yet).
func NewSuggestion(problemID, authorID, text string) *Suggestion {
	return &Suggestion{
		ID:        idgen.New("sug"),
		ProblemID: problemID,
		AuthorID:  authorID,
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}
}
