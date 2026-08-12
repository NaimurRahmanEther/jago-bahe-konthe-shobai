package domain

import "time"

// MaxEscalationLevel caps how far a case's visibility can climb: the pilot ladder
// runs from a ward-level official up through the union/upazila monitors to the MP
// (~3 hops). Level 0 = with the official only; each higher level = one more rung
// of the monitor ladder notified. Silence never removes the work from the
// official — only its visibility climbs (Concept §7).
const MaxEscalationLevel = 3

// EscalationTarget returns the visibility level a case should have climbed to,
// given how far past its anchor (the first-response deadline for a silent case,
// or the blocker review-window cap for a stalled blocker) it now is: one rung per
// elapsed window, capped at maxLevel. It is deterministic and idempotent — the
// same inputs always yield the same level — so re-running the worker never
// double-escalates. Before the anchor it is 0.
func EscalationTarget(anchor, now time.Time, window time.Duration, maxLevel int) int {
	if window <= 0 || !now.After(anchor) {
		return 0
	}
	level := int(now.Sub(anchor)/window) + 1
	if level > maxLevel {
		return maxLevel
	}
	return level
}

// SilenceLevel returns how many rungs of the monitor ladder a case's SILENCE has
// climbed: the same arithmetic as EscalationTarget, re-anchored on the official's
// last act.
//
// The two anchors mean different things, which is why they are not simply the
// later of the two. The deadline is the moment an answer was DUE — a case one hour
// past it with nothing on the record is already silent, so rung 1 opens. A response
// is the moment an answer was GIVEN, and it buys a fresh full window: the official
// has to be quiet for another whole D before a monitor is called back in. Take the
// later of (deadline) and (lastActivity + window) accordingly.
//
// The two clocks are deliberately separate and must stay so. EscalationTarget
// measures LATENESS against a fixed deadline and can only ever rise, which is what
// makes cases.escalation_level an honest record of how late a case ever got. This
// measures SILENCE, and it restarts every time the official acts — because an
// official posting weekly updates on a late case is not silent, and stacking more
// observers onto them would say they were. Merging the two would also hand an
// official a way to suppress escalation forever by posting an empty note every
// window (CLAUDE.md A.3.7).
func SilenceLevel(deadline, lastActivity, now time.Time, window time.Duration, maxLevel int) int {
	anchor := deadline
	if !lastActivity.IsZero() {
		if fresh := lastActivity.Add(window); fresh.After(anchor) {
			anchor = fresh
		}
	}
	return EscalationTarget(anchor, now, window, maxLevel)
}

// IsEscalatable reports whether a status is one the escalation scan considers
// (non-terminal work): a silent case still owned by the official, or a Blocked
// case whose review window may have lapsed. Done/Resolved/Disputed are finished.
func IsEscalatable(s Status) bool {
	switch s {
	case StatusAssigned, StatusAcknowledged, StatusPlanned, StatusInProgress, StatusReopened, StatusBlocked:
		return true
	}
	return false
}
