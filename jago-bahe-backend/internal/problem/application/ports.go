// Package application holds the problem use cases. It orchestrates the problem
// domain, the shared area geography, the audit log, and a small Identity port
// that exposes just the facts the problem context needs from the identity
// context (voter eligibility, official existence) — keeping the contexts decoupled.
package application

import "context"

// Voter is the subset of an account the problem context needs to decide
// validation eligibility.
type Voter struct {
	IsResident bool
	Verified   bool
	UnionID    string
}

// Identity is the port into the identity context. The composition root supplies
// an adapter over the identity repositories.
type Identity interface {
	Voter(ctx context.Context, accountID string) (Voter, error)
	OfficialExists(ctx context.Context, officialID string) (bool, error)
}

// AssignmentView is the public half of an assignment: who is working on the
// problem, and — when the admin overrode the public's nominee — the reason they
// gave for it. The rest of the assignment (monitor, priority, deadline) is
// admin-facing internals and deliberately not carried here.
//
// OverrideReason is empty for the ordinary case where the admin confirmed the
// public's choice, because there was nothing to justify.
type AssignmentView struct {
	OfficialID     string
	OverrideReason string
}

// Assignments is the port into the assignment context, used to publish which
// official a problem was actually handed to.
//
// The problem context must not import assignment (dependencies point inward, and
// these are sibling contexts), so the composition root supplies an adapter over
// the assignment repository — the same shape as the resolution context's
// Suggestions port.
type Assignments interface {
	// For returns the assignment of one problem, or a zero AssignmentView when it
	// has none. Unassigned is the normal state of most problems and is never an
	// error — the adapter swallows the not-found sentinel.
	For(ctx context.Context, problemID string) (AssignmentView, error)
	// ForMany resolves a whole page in one query, keyed by problem id and omitting
	// unassigned problems. The feed publishes the assigned official for every row,
	// so this must not degrade into a per-row For — that is the N+1 the field was
	// withheld from the feed to avoid.
	ForMany(ctx context.Context, problemIDs []string) (map[string]AssignmentView, error)
}

// Admins is the port into the admin side, used by the B9 screening gate.
// RequireAdmin is role-only with no area awareness, so it admits any admin to any
// problem — the union check has to happen here, in the use case, exactly as the
// assignment context does it.
type Admins interface {
	// UnionOf returns the union an admin belongs to (via their linked official),
	// or "" when they have none on record.
	UnionOf(ctx context.Context, adminAccountID string) (string, error)
}
