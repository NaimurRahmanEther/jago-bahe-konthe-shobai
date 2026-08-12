package domain

import "errors"

// Sentinel lifecycle errors mapped to HTTP status by the http layer.
var (
	// ErrIllegalTransition is returned when an event is not legal from the current
	// status — the state machine's central guarantee.
	ErrIllegalTransition = errors.New("illegal lifecycle transition")
	// ErrEvidenceRequired is returned when Done is attempted without evidence.
	ErrEvidenceRequired = errors.New("evidence is required before a case can be marked done")
)

// Event is a lifecycle action requested on a case.
type Event string

const (
	EventAcknowledge     Event = "acknowledge"
	EventDispute         Event = "dispute"
	EventPlan            Event = "plan"
	EventReplan          Event = "replan"
	EventProgress        Event = "progress"
	EventBlock           Event = "block"
	EventMarkDone        Event = "mark_done"
	EventConfirmResolved Event = "confirm_resolved"
	EventConfirmReopened Event = "confirm_reopened"
)

// Guards carries the facts a transition depends on beyond the current status
// (currently just whether evidence is attached, which gates Done).
type Guards struct {
	HasEvidence bool
}

// Transition is the state machine: it returns the status that results from
// applying ev to from, or an error if the move is illegal. This is the single
// place legal transitions are defined; every use case routes through it.
//
//	Assigned      → Acknowledged (acknowledge) | Disputed (dispute)
//	Acknowledged  → Planned (plan)
//	Planned       → InProgress (progress) | Blocked (block)
//	InProgress    → InProgress (progress | replan) | Blocked (block) | Done (mark_done, needs evidence)
//	Blocked       → InProgress (progress — work resumes)
//	Reopened      → InProgress (progress | replan)
//	Done          → Resolved (confirm_resolved) | Reopened (confirm_reopened)
//
// The confirm_* events are driven by the resident confirmation use case (B6); an
// official never reaches Resolved themselves (they can only reach Done).
//
// replan lets the official post a fresh plan to restart work after the first plan
// stalled — legal only from InProgress (e.g. after an obstacle was resolved) and
// Reopened (a resident sent it back). It resolves to InProgress: a new plan is the
// resumption of active work, not a regression to Planned (which would leave the
// public problem row, mirrored to InProgress, disagreeing with the case). It is
// deliberately NOT legal from Blocked — a blocked case must first be unblocked
// (obstacle adjudicated, or a progress note) before it can be re-planned.
func Transition(from Status, ev Event, g Guards) (Status, error) {
	switch from {
	case StatusAssigned:
		switch ev {
		case EventAcknowledge:
			return StatusAcknowledged, nil
		case EventDispute:
			return StatusDisputed, nil
		}
	case StatusAcknowledged:
		if ev == EventPlan {
			return StatusPlanned, nil
		}
	case StatusPlanned:
		switch ev {
		case EventProgress:
			return StatusInProgress, nil
		case EventBlock:
			return StatusBlocked, nil
		}
	case StatusInProgress:
		switch ev {
		case EventProgress, EventReplan:
			return StatusInProgress, nil
		case EventBlock:
			return StatusBlocked, nil
		case EventMarkDone:
			if !g.HasEvidence {
				return from, ErrEvidenceRequired
			}
			return StatusDone, nil
		}
	case StatusBlocked:
		if ev == EventProgress {
			return StatusInProgress, nil
		}
	case StatusReopened:
		switch ev {
		case EventProgress, EventReplan:
			return StatusInProgress, nil
		}
	case StatusDone:
		switch ev {
		case EventConfirmResolved:
			return StatusResolved, nil
		case EventConfirmReopened:
			return StatusReopened, nil
		}
	}
	return from, ErrIllegalTransition
}

// IsWorkable reports whether a case is in a state where the official may log
// progress/blocker updates and attach evidence (the working side-states). These
// operations don't themselves change status, so they guard on this rather than
// going through Transition.
func IsWorkable(s Status) bool {
	switch s {
	case StatusPlanned, StatusInProgress, StatusBlocked, StatusReopened:
		return true
	}
	return false
}
