package domain

import "time"

// SuggestionVote is one resident's upvote of a suggestion. An upvote is a toggle:
// its presence means the resident supports the suggestion, its absence means they
// do not. Distinctness (one upvote per resident per suggestion) is enforced at the
// repository level.
type SuggestionVote struct {
	ID           string
	SuggestionID string
	VoterID      string
	CreatedAt    time.Time
}
