package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// Observation is one rung of the monitor ladder watching a case its official has
// gone silent on: the copy that "surfaces on the monitor's dashboard, flagged and
// publicly noted" (Concept §7). Level 1 is the direct monitor, 2 the tier above,
// up to MaxEscalationLevel.
//
// It records VISIBILITY ONLY. The case is never moved off the official and the
// observer is given no power over it — a declared obstacle moves responsibility
// up, silence moves only visibility (Concept §7, CLAUDE.md A.3.7).
//
// An observation is opened by the worker and resolved when the official finally
// acts. The resolved row is KEPT: that an official was silent for eleven days is a
// fact about the public record, and deleting it on response would erase it.
type Observation struct {
	ID                 string
	CaseID             string
	ObserverOfficialID string
	Level              int
	OpenedAt           time.Time
	ResolvedAt         *time.Time
	Notes              []ObservationNote
}

// NewObservation opens a rung of the ladder on a case at the given instant.
func NewObservation(caseID, observerOfficialID string, level int, at time.Time) *Observation {
	return &Observation{
		ID:                 idgen.New("obs"),
		CaseID:             caseID,
		ObserverOfficialID: observerOfficialID,
		Level:              level,
		OpenedAt:           at,
	}
}

// IsOpen reports whether the official is still silent on this rung.
func (o *Observation) IsOpen() bool { return o.ResolvedAt == nil }

// Resolve closes the observation at the moment the official answered. It is
// idempotent: an already-resolved observation keeps its first resolution, because
// when the silence ENDED is the fact worth keeping.
func (o *Observation) Resolve(at time.Time) {
	if o.ResolvedAt != nil {
		return
	}
	o.ResolvedAt = &at
}

// ObservationNote is the one recorded action a monitor has: a public note on the
// case they are watching ("followed up with the chairman on this date"). Concept §7
// asks for it explicitly, so that the monitoring role is workable and so that a
// supervisor ignoring an unresponsive subordinate is itself visible.
//
// Notes are APPEND-ONLY — never one overwritable column — and each one also lands
// on the problem's public audit trail, which is how "publicly noted" is satisfied
// (the same route a rejection ground and an assignment override reason take).
type ObservationNote struct {
	ID            string
	ObservationID string
	Text          string
	CreatedAt     time.Time
}

// NewObservationNote constructs a monitor's note on an observation.
func NewObservationNote(observationID, text string, at time.Time) ObservationNote {
	return ObservationNote{
		ID:            idgen.New("obsn"),
		ObservationID: observationID,
		Text:          text,
		CreatedAt:     at,
	}
}

// MissingRungs returns the ladder rungs that should be open at the given target
// level but are not, lowest first. Walking in order is what stops a case becoming
// visible to the MP without being visible to the tier below it first; the caller
// opens them in the order returned and stops when the ladder runs out of officials.
//
// A resolved observation does not count as open, which is what lets a case that
// went quiet, was answered, and went quiet again re-open rung 1 as a NEW row rather
// than silently never escalating a second time.
func MissingRungs(open []Observation, target int) []int {
	have := make(map[int]bool, len(open))
	for i := range open {
		if open[i].IsOpen() {
			have[open[i].Level] = true
		}
	}
	var out []int
	for level := 1; level <= target; level++ {
		if !have[level] {
			out = append(out, level)
		}
	}
	return out
}
