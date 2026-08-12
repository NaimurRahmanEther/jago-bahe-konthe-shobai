package application

import (
	"context"
	"strings"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// NoteObservation records what a monitor did about a silence: the one action the
// observation ladder gives them.
//
// Concept §7 asks for it by name — each monitor gets "their own response window and
// a recorded action to take, so the monitoring role stays workable" — and the point
// of recording it is that it makes supervision itself accountable. A monitor who
// watches a case go silent for a month and writes nothing has that visible too.
//
// It is the ONLY thing an observer may do. There is no endpoint here that reassigns
// the case, takes it over, or closes it: a declared obstacle moves responsibility
// up the ladder, silence moves only visibility, and the work stays with the
// official who was given it (Concept §7, CLAUDE.md A.3.7).
type NoteObservation struct {
	repo  domain.Repository
	audit auditdomain.Repository
}

// NewNoteObservation wires the use case.
func NewNoteObservation(r domain.Repository, audit auditdomain.Repository) *NoteObservation {
	return &NoteObservation{repo: r, audit: audit}
}

// Execute appends a note to the observation, if the caller is its observer.
func (uc *NoteObservation) Execute(ctx context.Context, observationID, callerOfficialID, text string) (*domain.ObservationNote, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.ErrEmptyText
	}
	if callerOfficialID == "" {
		return nil, domain.ErrNotObserver
	}

	obs, err := uc.repo.GetObservation(ctx, observationID)
	if err != nil {
		return nil, err
	}
	if obs.ObserverOfficialID != callerOfficialID {
		return nil, domain.ErrNotObserver
	}
	// A resolved observation is history. Allowing a note on it would let the record
	// of a silence be edited after the silence ended.
	if !obs.IsOpen() {
		return nil, domain.ErrObservationResolved
	}

	note := domain.NewObservationNote(obs.ID, text, time.Now().UTC())
	if err := uc.repo.AddObservationNote(ctx, note); err != nil {
		return nil, err
	}

	// The note is public through the audit trail — the same route a rejection
	// ground, an assignment override reason and a re-plan reason take. Concept §7
	// asks for the monitor's action to be "publicly noted", and the problem's own
	// trail is where this platform publishes reasons.
	kase, err := uc.repo.GetByID(ctx, obs.CaseID)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.Append(ctx, auditdomain.NewAuditEntry(
		"problem", kase.ProblemID, callerOfficialID, "observation_noted", text)); err != nil {
		return nil, err
	}
	return &note, nil
}
