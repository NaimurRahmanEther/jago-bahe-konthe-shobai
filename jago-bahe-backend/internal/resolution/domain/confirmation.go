package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// ConfirmationOutcome is the reporting resident's verdict on a Done case:
// solved → Resolved, not_solved → Reopened. The confirmation use case is wired in
// B6; the type lives here as part of the resolution vocabulary.
type ConfirmationOutcome string

const (
	ConfirmationSolved    ConfirmationOutcome = "solved"
	ConfirmationNotSolved ConfirmationOutcome = "not_solved"
)

// Valid reports whether o is a known outcome.
func (o ConfirmationOutcome) Valid() bool {
	return o == ConfirmationSolved || o == ConfirmationNotSolved
}

// Confirmation records a reporting resident's verdict on a Done case.
type Confirmation struct {
	ID         string
	ProblemID  string
	ResidentID string
	Outcome    ConfirmationOutcome
	CreatedAt  time.Time
}

// NewConfirmation constructs a confirmation record.
func NewConfirmation(problemID, residentID string, outcome ConfirmationOutcome) Confirmation {
	return Confirmation{
		ID:         idgen.New("cnf"),
		ProblemID:  problemID,
		ResidentID: residentID,
		Outcome:    outcome,
		CreatedAt:  time.Now().UTC(),
	}
}
