package domain

import "time"

// VoteChoice is a resident's judgement that a reported problem is real or not.
type VoteChoice string

const (
	VoteValid   VoteChoice = "valid"
	VoteInvalid VoteChoice = "invalid"
)

// Valid reports whether v is a known choice.
func (v VoteChoice) Valid() bool { return v == VoteValid || v == VoteInvalid }

// ValidationVote is one resident's validation of a problem. Distinctness (one
// vote per resident per problem) is enforced at the repository level.
type ValidationVote struct {
	ID        string
	ProblemID string
	VoterID   string
	Vote      VoteChoice
	CreatedAt time.Time
}
