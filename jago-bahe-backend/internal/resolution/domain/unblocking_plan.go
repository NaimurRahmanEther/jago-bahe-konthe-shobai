package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// UnblockingPlan is a community-proposed way to overcome a blocker — the
// suggestion engine re-pointed at the blocker. UpvoteCount is derived on read;
// ranking reuses the suggestion "most upvotes wins" rule (service.RankUnblocking).
type UnblockingPlan struct {
	ID          string
	ObstacleID  string
	AuthorID    string
	Text        string
	UpvoteCount int
	CreatedAt   time.Time
}

// NewUnblockingPlan constructs an unblocking plan.
func NewUnblockingPlan(obstacleID, authorID, text string) *UnblockingPlan {
	return &UnblockingPlan{
		ID:         idgen.New("ubp"),
		ObstacleID: obstacleID,
		AuthorID:   authorID,
		Text:       text,
		CreatedAt:  time.Now().UTC(),
	}
}
