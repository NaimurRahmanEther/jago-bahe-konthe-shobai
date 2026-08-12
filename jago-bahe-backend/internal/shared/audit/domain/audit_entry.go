// Package domain (audit) models the public, append-only audit trail. Every
// state-changing use case in every context appends an AuditEntry — this is the
// visible face of the transparency guarantee (Scaffold Spec §2).
package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// ActorSystem is the Actor value for a state change no human made — a threshold
// being reached, a deadline expiring, a screening window running out. Readers of
// the public trail use it to tell "the platform did this by rule" apart from "a
// named person decided this", so it must stay a single spelling.
const ActorSystem = "system"

// AuditEntry is one immutable record of a state change: who did what to which
// target, optionally why, and when.
type AuditEntry struct {
	ID         string
	TargetType string // e.g. "problem", "case"
	TargetID   string
	Actor      string // resident/official/admin id, or ActorSystem
	Action     string // e.g. "reported", "validated", "blocked"
	Reason     string // optional; empty means none
	CreatedAt  time.Time
}

// NewAuditEntry constructs an entry with a fresh id and the current time.
func NewAuditEntry(targetType, targetID, actor, action, reason string) AuditEntry {
	return AuditEntry{
		ID:         idgen.New("audit"),
		TargetType: targetType,
		TargetID:   targetID,
		Actor:      actor,
		Action:     action,
		Reason:     reason,
		CreatedAt:  time.Now().UTC(),
	}
}
