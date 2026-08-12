// Package domain (problem) owns problems, their post-publication moderation, and
// their community validation. Pure logic — no database, HTTP, or other contexts.
// This is the reference slice every later context copies.
package domain

// Status is a problem's lifecycle state. B2 uses Reported and Validated; the rest
// are set by later contexts (assignment, resolution) but are defined here so the
// vocabulary is complete and matches the frontend StatusBadge.
type Status string

const (
	// StatusPendingApproval is where every new problem starts. It is NOT public: a
	// report is held invisible to everyone but its reporter and its own union's
	// admin until that admin approves it. The gate exists so spam and defamation
	// are not publicly readable before a human has seen them; the cost, accepted
	// deliberately, is that an admin who never acts leaves a genuine report hidden
	// (there is no auto-publish — see the reversal note on Screening). Approval
	// moves it to Reported; rejection moves it to Rejected.
	StatusPendingApproval Status = "PendingApproval"

	// StatusReported is where an approved problem lands: public and votable, but
	// still unvalidated — clearing the spam gate says nothing about merit, which
	// stays the community's call via the threshold V.
	StatusReported   Status = "Reported"
	StatusValidated  Status = "Validated"
	StatusAssigned   Status = "Assigned"
	StatusInProgress Status = "InProgress"
	StatusBlocked    Status = "Blocked"
	StatusDone       Status = "Done"
	StatusResolved   Status = "Resolved"
	StatusReopened   Status = "Reopened"

	// StatusRejected is terminal and, deliberately, public: a rejected problem
	// keeps its reason on the record so a wrongly-buried one has a witness.
	StatusRejected Status = "Rejected"

	// StatusWithdrawn is terminal and public: the reporter took their own problem
	// down before it was assigned. Like Rejected, it stays on the record rather
	// than vanishing — the reporter can retract a report, but not erase that it was
	// filed. The retraction's reason travels on the public audit entry. See
	// Status.Withdrawable for the window.
	StatusWithdrawn Status = "Withdrawn"
)

// Valid reports whether s is one of the eleven known states. The Postgres column
// has no CHECK constraint, so this is the only guard against an unknown status
// reaching the domain.
func (s Status) Valid() bool {
	switch s {
	case StatusPendingApproval, StatusReported, StatusValidated, StatusAssigned,
		StatusInProgress, StatusBlocked, StatusDone, StatusResolved, StatusReopened,
		StatusRejected, StatusWithdrawn:
		return true
	}
	return false
}

// PublicStatuses is the set a problem must hold to appear in the public feed:
// every state except PendingApproval. A pending report is deliberately withheld —
// it is the one hidden state, invisible until its union admin approves it. This is
// also the frontend's mirror (models.js ProblemStatus, ProblemFeed's filter
// options), so PendingApproval must be excluded here exactly as it is there.
func PublicStatuses() []Status {
	return []Status{
		StatusReported, StatusValidated, StatusAssigned, StatusInProgress,
		StatusBlocked, StatusDone, StatusResolved, StatusReopened, StatusRejected,
		StatusWithdrawn,
	}
}

// PubliclyVisible reports whether a problem in this state may be shown to anyone.
// Every known state is public except PendingApproval, which is readable only by
// its reporter and its own union's admin (see application/get_problem.go). It is
// what list_problems validates a requested ?status= against; an unknown status is
// withheld too.
func (s Status) PubliclyVisible() bool {
	return s != StatusPendingApproval && s.Valid()
}

// Rejectable reports whether an admin may still take this problem down as spam,
// abuse, a duplicate, or wrong-area. The window is deliberately narrow: only
// before assignment. Once a problem is assigned, a case exists, an official is
// working in public, and a takedown would erase that work — so from there the only
// authority over the problem is the lifecycle itself.
//
// Rejecting is never a judgement on merit; that is the community's call via V.
// Widening this window widens the takedown veto, so treat additions as a change to
// the platform's power balance, not a convenience.
//
// PendingApproval is included because it is where the admin's screening decision
// is made: a report the admin finds to be spam is rejected here, before it is ever
// public. Reported and Validated remain rejectable so a report that slipped
// through approval (a mistaken approval, or a duplicate that only later becomes
// apparent) can still be taken down — but only before assignment.
func (s Status) Rejectable() bool {
	return s == StatusPendingApproval || s == StatusReported || s == StatusValidated
}

// Withdrawable reports whether the reporter may still take their own problem
// down. The window mirrors Rejectable — only before assignment — for the same
// reason: once a problem is assigned a case exists and an official is working in
// public, and letting the reporter withdraw it there would erase that work.
//
// It is a separate method from Rejectable, not an alias, because the two are
// different powers held by different people: an admin's takedown veto and the
// reporter's own retraction. Keeping them separate lets either window move without
// silently moving the other.
//
// PendingApproval is included: a report is its reporter's own until an admin acts,
// so they may retract it while it is still awaiting review.
func (s Status) Withdrawable() bool {
	return s == StatusPendingApproval || s == StatusReported || s == StatusValidated
}

// EligibleForAssignment reports whether an admin may still forward this problem
// to an official. It means "the report is public and no case exists yet" — it is
// NOT a vote threshold.
//
// This used to be `s == StatusValidated`: a report could not reach an official
// until V distinct verified residents had validated it. B17 removed that gate. The
// community's votes are now evidence the admin weighs, not a condition they wait
// on — an admin may forward a Reported problem with zero validations if they judge
// it trustworthy, and the public record shows the count they acted on. V still
// flips Reported to Validated, but that flip is a public endorsement rather than a
// key. See CLAUDE.md A.3.1 for the rule and why it changed.
//
// Both bounds of the window are load-bearing:
//
//   - PendingApproval is excluded because screening comes first. An unscreened
//     report is not public, and forwarding one would hand an official work that no
//     human has yet confirmed is not spam or defamation.
//   - Every post-assignment state (Assigned, InProgress, Blocked, Done, Resolved,
//     Reopened) is excluded because a case already exists and resolution owns the
//     lifecycle from there; assigning again would fork an official's public work.
//   - Rejected and Withdrawn are terminal.
//
// Widening this set hands the admin more of the public's authority, so treat
// additions as a change to the platform's power balance, not a convenience — the
// same standard Rejectable sets.
func (s Status) EligibleForAssignment() bool {
	return s == StatusReported || s == StatusValidated
}
