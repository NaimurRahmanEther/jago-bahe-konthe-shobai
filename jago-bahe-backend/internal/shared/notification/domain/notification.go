// Package domain (notification) models the platform's in-app messages: the small,
// private set of events that close a loop for a party who has no other surface to
// learn from. It sits in the shared kernel beside audit because it is the same kind
// of thing — an append-mostly LOG, not a bounded context. It holds no policy about
// who is told what; that decision lives inline in each writing use case, beside the
// audit append it sits next to (CLAUDE.md A.3.9).
//
// The message is SNAPSHOTTED, never a live projection. ProblemTitle and Detail are
// copied at write time for the reason plans.answered_suggestion_text is: a
// notification records what was true when it was sent, and a reporter may edit
// their own title afterwards. It also means this package needs no ports at all —
// unlike audit, which declares Actors and Problems, it never has to look anything
// up. If you find yourself adding an application/ports.go here, something has been
// un-snapshotted; check that first.
package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// Kind names WHICH ID SPACE RecipientID belongs to.
//
// The platform has two, and the JWT carries both: an account id for residents,
// admins and the super admin, and a DIRECTORY OFFICE id for officials (an
// official's actions are recorded against `off-mayor`, never their account).
// audit_entries.actor holds the same two spaces in one column and tells them apart
// by prefix convention alone — auditActorAdapter in cmd/api/adapters.go says so,
// and accepts that a hypothetical id present in both would resolve to the account.
//
// That is fine for a DISPLAY NAME, where a collision costs a wrong label. It is not
// fine here, where a collision costs a private message reaching the wrong person,
// and the authorship graph is a safety matter (A.3.2 rule 1). So the row records
// which space its id is in, and the read query matches on the PAIR.
type Kind string

// The two id spaces.
const (
	KindAccount  Kind = "account"
	KindOfficial Kind = "official"
)

// Valid reports whether k is a known id space.
func (k Kind) Valid() bool { return k == KindAccount || k == KindOfficial }

// Type is the closed set of eight events — the "loop-closing set".
//
// It is deliberately NOT the audit action vocabulary. One audit action fans out to
// two types (`assigned` -> problem_assigned for the reporter AND case_assigned for
// the official) and two fold into one (`blocker_confirmed` / `blocker_denied` ->
// obstacle_adjudicated, discriminated by Detail). Reusing the audit literals would
// look tidy and would couple two vocabularies that version apart — the audit trail
// is a public record of power exercised, this is a private message queue.
type Type string

// The eight. Detail's meaning is per-type and is noted on each.
const (
	// To the REPORTER — recipient kind account.
	TypeProblemApproved       Type = "problem_approved"
	TypeProblemRejected       Type = "problem_rejected"       // Detail = the rejection ground enum
	TypeProblemAssigned       Type = "problem_assigned"       // Detail = the assigned official's NAME
	TypeConfirmationRequested Type = "confirmation_requested" // Detail = ""

	// To the ASSIGNED OFFICIAL — recipient kind official.
	TypeCaseAssigned        Type = "case_assigned"        // Detail = the deadline D, RFC3339
	TypeCaseReopened        Type = "case_reopened"        // Detail = ""
	TypeObstacleAdjudicated Type = "obstacle_adjudicated" // Detail = "confirmed" | "denied"

	// To the MONITOR OFFICIAL — recipient kind official. The one promise the design
	// documents make outright (Concept §8), delivered to the addressable party; see
	// report_obstacle.go and A.3.9 constraint 5 for why that is the monitor and not
	// the free-text authority the official named.
	TypeObstacleDeclared Type = "obstacle_declared" // Detail = the free-text whoUnblocks
)

// AllTypes is every legal Type. It exists so the domain test can iterate rather
// than restate the list: a ninth constant added without a Valid() case fails there,
// which is the cheap half of guarding against a type nothing recognises.
var AllTypes = []Type{
	TypeProblemApproved,
	TypeProblemRejected,
	TypeProblemAssigned,
	TypeConfirmationRequested,
	TypeCaseAssigned,
	TypeCaseReopened,
	TypeObstacleAdjudicated,
	TypeObstacleDeclared,
}

// Valid whitelists the eight.
//
// The `adjudicated` failure (A.3.5 rule 2) is this set's failure mode exactly: an
// action written by nothing renders as a row nobody ever sees, which is
// indistinguishable from a seat where the event never happens. Three guards,
// because Valid() alone catches only the wrong-spelling half — notifications.type
// carries a CHECK constraint (000023), each writing use case's test asserts the
// exact literal rather than the constant, and bn.json needs a key per type, where
// a missing one renders as the raw key because there is no fallbackLng (A.5.3).
func (t Type) Valid() bool {
	switch t {
	case TypeProblemApproved, TypeProblemRejected, TypeProblemAssigned,
		TypeConfirmationRequested, TypeCaseAssigned, TypeCaseReopened,
		TypeObstacleAdjudicated, TypeObstacleDeclared:
		return true
	}
	return false
}

// Notification is one in-app message.
//
// There is deliberately no Actor field. Only problem_assigned names a person, and
// it names them as snapshotted Detail text; an actor id would create a second
// polymorphic id space to resolve and would tempt a read-time NamesByIDs batch,
// reintroducing the ports this package does without.
type Notification struct {
	ID            string
	RecipientKind Kind
	RecipientID   string
	Type          Type
	ProblemID     string
	ProblemTitle  string // snapshotted; "" when the writer had no title to hand
	Detail        string // type-specific; read ONLY inside a switch on Type
	ReadAt        *time.Time
	CreatedAt     time.Time
}

// ForResident addresses an ACCOUNT — a resident, an admin, or the super admin.
//
// ForResident and ForOfficial are the only constructors, and the id space is in the
// FUNCTION NAME so it cannot be transposed at a call site. That matters concretely:
// Case.MonitorOfficialID is "" when the assignee is at the top of the ladder, and
// an empty recipient in a shared column would match every caller who passes an
// empty officialId — which is every non-official in the seat.
func ForResident(accountID string, t Type, problemID, problemTitle, detail string) Notification {
	return newNotification(KindAccount, accountID, t, problemID, problemTitle, detail)
}

// ForOfficial addresses a DIRECTORY OFFICE — never an account. See ForResident.
func ForOfficial(officialID string, t Type, problemID, problemTitle, detail string) Notification {
	return newNotification(KindOfficial, officialID, t, problemID, problemTitle, detail)
}

func newNotification(k Kind, recipientID string, t Type, problemID, problemTitle, detail string) Notification {
	return Notification{
		ID:            idgen.New("notif"),
		RecipientKind: k,
		RecipientID:   recipientID,
		Type:          t,
		ProblemID:     problemID,
		ProblemTitle:  problemTitle,
		Detail:        detail,
		CreatedAt:     time.Now().UTC(),
	}
}

// Unread reports whether the recipient has not yet marked this read.
func (n Notification) Unread() bool { return n.ReadAt == nil }

// Recipients is the CALLER'S OWN two identities, taken from their JWT and nowhere
// else. Every read and every write on the Repository takes one of these; there is
// deliberately NO method anywhere that accepts a bare recipient id, so a route that
// reads someone else's notifications is not expressible.
//
// That is A.3.2 rule 1 ("the view is self-derived") moved out of convention and
// into the type system, because a private-message store is where forgetting that
// convention costs the most.
type Recipients struct {
	AccountID  string // auth.Claims.UserID
	OfficialID string // auth.Claims.OfficialID — "" for residents, admins, super admin
}

// Empty reports whether the caller has no identity at all — an unauthenticated
// request. Callers answer it with ErrNoCaller rather than an empty list: a silently
// empty list reads as "you have no messages", which is a different claim.
func (r Recipients) Empty() bool { return r.AccountID == "" && r.OfficialID == "" }
