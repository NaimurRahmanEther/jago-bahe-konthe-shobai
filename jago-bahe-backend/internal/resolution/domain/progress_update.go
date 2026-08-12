package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// UpdateKind distinguishes a routine progress note from a plain obstacle note in
// the public timeline. (A formal typed Obstacle — which changes status and
// notifies a higher authority — is created via report_obstacle, not here.)
type UpdateKind string

const (
	UpdateProgress UpdateKind = "progress"
	UpdateObstacle UpdateKind = "obstacle"
)

// Valid reports whether k is a known kind.
func (k UpdateKind) Valid() bool { return k == UpdateProgress || k == UpdateObstacle }

// ProgressUpdate is one entry in a case's public weekly timeline.
type ProgressUpdate struct {
	ID        string
	CaseID    string
	Kind      UpdateKind
	Text      string
	CreatedAt time.Time
}

// NewProgressUpdate constructs a timeline entry.
func NewProgressUpdate(caseID string, kind UpdateKind, text string) ProgressUpdate {
	return ProgressUpdate{
		ID:        idgen.New("upd"),
		CaseID:    caseID,
		Kind:      kind,
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}
}
