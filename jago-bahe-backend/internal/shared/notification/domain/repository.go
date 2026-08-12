package domain

import (
	"context"
	"errors"
)

// Sentinel errors for the notification log.
var (
	// ErrNoRecipient is returned by Append for a message with an empty recipient —
	// the Case.MonitorOfficialID-of-an-MP case, where MonitorFor returns "" because
	// there is nobody above the assignee. Write sites append fire-and-forget with
	// `_ =`, so this is never surfaced to a caller; its job is to make the guard
	// central and unforgettable rather than repeated at eight call sites.
	ErrNoRecipient = errors.New("notification: empty recipient")

	// ErrInvalidType and ErrInvalidKind guard the two enums on the way in, so a
	// typo'd literal fails here rather than at the DB's CHECK constraint.
	ErrInvalidType = errors.New("notification: unknown type")
	ErrInvalidKind = errors.New("notification: unknown recipient kind")

	// ErrNotificationNotFound is what MarkRead returns when the update matched no
	// row — whether the id does not exist, or exists and belongs to someone else.
	// The two are DELIBERATELY the same answer, and the HTTP layer maps this to
	// 404 and never 403, so mark-read is no oracle over notification ids. Same
	// rule GetProblem's mayViewPending follows (A.3.1 constraint 2).
	ErrNotificationNotFound = errors.New("notification: not found")

	// ErrNoCaller mirrors problemdomain.ErrNoCaller: an unauthenticated read is a
	// 401, never a silently empty list that reads as "you have no messages".
	ErrNoCaller = errors.New("notification: no caller")
)

// Repository is the port for the notification log.
//
// It is append-plus-mark-read: there is no update path other than setting read_at,
// and NO DELETE PATH AT ALL. Erasure comes from notifications.problem_id's
// ON DELETE CASCADE, which is strictly better than audit's manual DeleteByTarget —
// that exists only because audit_entries has no FK to problems, and this table has
// one. A hard-deleted report therefore takes its notifications with it for free
// (A.3.3).
//
// EVERY method takes Recipients, never a bare id. See that type's doc comment: it
// is what makes "read someone else's notifications" inexpressible rather than
// merely forbidden.
//
// Marking read is the ONE state change in this platform that appends no audit
// entry. The audit log is public, so an entry per mark-read would publish who is
// reading what — surveillance wearing the costume of transparency (A.3.2 rule 4).
// Reads are unaudited for the same reason; this is that rule followed one step
// further, into the one write that is ABOUT reading.
type Repository interface {
	// Append stores one message. It refuses an empty recipient (ErrNoRecipient), an
	// unknown Type (ErrInvalidType) and an unknown Kind (ErrInvalidKind).
	Append(ctx context.Context, n Notification) error

	// ListForRecipient returns the caller's own messages, newest first.
	ListForRecipient(ctx context.Context, r Recipients, limit int) ([]Notification, error)

	// CountUnread is the bell's badge — a count over the caller's own rows only, so
	// its denominator is exactly the set the caller may read. Contrast the B10
	// SeatProblemCount regression, whose denominator included rows the reader could
	// not see and so leaked how many unscreened reports existed.
	CountUnread(ctx context.Context, r Recipients) (int, error)

	// MarkRead marks one message read. Scoping lives in the UPDATE itself, so zero
	// rows affected is the only possible answer for someone else's id, and it maps
	// to ErrNotificationNotFound. Idempotent: marking an already-read message read
	// again succeeds rather than 404ing.
	MarkRead(ctx context.Context, id string, r Recipients) error

	// MarkAllRead marks every unread message read and reports how many.
	MarkAllRead(ctx context.Context, r Recipients) (int, error)
}
