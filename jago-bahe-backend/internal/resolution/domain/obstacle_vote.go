package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// ObstacleVoteChoice is the public's advisory read on whether a blocker is a
// genuine obstacle. It is PRESSURE ONLY — it never changes status or the
// scorecard; the named authority adjudicates (Concept §8, B6).
type ObstacleVoteChoice string

const (
	ObstacleReal         ObstacleVoteChoice = "real"
	ObstacleNotConvinced ObstacleVoteChoice = "not_convinced"
)

// Valid reports whether c is a known choice.
func (c ObstacleVoteChoice) Valid() bool {
	return c == ObstacleReal || c == ObstacleNotConvinced
}

// ObstacleVote is one resident's advisory vote on a blocker (one per resident per
// blocker — the distinctness invariant).
type ObstacleVote struct {
	ID         string
	ObstacleID string
	VoterID    string
	Choice     ObstacleVoteChoice
	CreatedAt  time.Time
}

// NewObstacleVote constructs an advisory vote.
func NewObstacleVote(obstacleID, voterID string, choice ObstacleVoteChoice) ObstacleVote {
	return ObstacleVote{
		ID:         idgen.New("obv"),
		ObstacleID: obstacleID,
		VoterID:    voterID,
		Choice:     choice,
		CreatedAt:  time.Now().UTC(),
	}
}
