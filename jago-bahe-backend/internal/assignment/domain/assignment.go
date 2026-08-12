package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// Assignment records that a validated problem was handed to an official. It sets
// the monitor (the tier up the ladder, notified on blockers/escalation in B5/B6),
// a priority, and the first-response deadline D that drives escalation.
// OverrideReason is required and public when an admin overrides the public's
// pointed official; it is empty when the public's choice was confirmed.
type Assignment struct {
	ID                string
	ProblemID         string
	OfficialID        string
	MonitorOfficialID string
	Priority          string
	Deadline          time.Time
	OverrideReason    string
	CreatedAt         time.Time
}

// NewAssignment constructs an assignment record.
func NewAssignment(problemID, officialID, monitorOfficialID, priority string, deadline time.Time, overrideReason string) *Assignment {
	return &Assignment{
		ID:                idgen.New("asgn"),
		ProblemID:         problemID,
		OfficialID:        officialID,
		MonitorOfficialID: monitorOfficialID,
		Priority:          priority,
		Deadline:          deadline,
		OverrideReason:    overrideReason,
		CreatedAt:         time.Now().UTC(),
	}
}
