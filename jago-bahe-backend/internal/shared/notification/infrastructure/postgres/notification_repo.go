// Package postgres implements the notification Repository via pgx.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// NotificationRepository is the pgx-backed implementation of
// notificationdomain.Repository.
type NotificationRepository struct {
	pool *pgxpool.Pool
}

// NewNotificationRepository constructs a NotificationRepository over the given pool.
func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

var _ notificationdomain.Repository = (*NotificationRepository)(nil)

// recipientPredicate matches the caller's OWN two identities and nothing else.
//
// The `$n <> ”` halves are load-bearing rather than defensive: every resident,
// admin and super admin calls with an empty OfficialID, so without them a single
// blank-recipient row would be delivered to all of them at once. The domain
// constructors and Append's guard stop such a row being written; this stops it
// being READ if one ever exists. Belt and braces, on a private message store.
//
// $1 = accountID, $2 = officialID. Covered by idx_notifications_recipient.
const recipientPredicate = `(
	(recipient_kind = 'account'  AND $1 <> '' AND recipient_id = $1)
	OR (recipient_kind = 'official' AND $2 <> '' AND recipient_id = $2)
)`

const notificationColumns = `id, recipient_kind, recipient_id, type, problem_id, problem_title, detail, read_at, created_at`

// Append stores one message. The enum and recipient guards live here, in front of
// the DB's CHECK constraints, so a bad write fails with a domain sentinel rather
// than a Postgres error the HTTP layer would have to translate (A.4.8).
func (r *NotificationRepository) Append(ctx context.Context, n notificationdomain.Notification) error {
	if n.RecipientID == "" {
		// The Case.MonitorOfficialID-of-an-MP case. Write sites append with `_ =`,
		// so this is silently correct: nobody is above the assignee, so there is
		// nobody to tell, and that is a fact about the ladder rather than a failure.
		return notificationdomain.ErrNoRecipient
	}
	if !n.RecipientKind.Valid() {
		return notificationdomain.ErrInvalidKind
	}
	if !n.Type.Valid() {
		return notificationdomain.ErrInvalidType
	}

	const q = `
		INSERT INTO notifications (id, recipient_kind, recipient_id, type, problem_id, problem_title, detail, read_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, q,
		n.ID, string(n.RecipientKind), n.RecipientID, string(n.Type),
		n.ProblemID, n.ProblemTitle, n.Detail, n.ReadAt, n.CreatedAt)
	if err != nil {
		return fmt.Errorf("notification append: %w", err)
	}
	return nil
}

// ListForRecipient returns the caller's own messages, newest first.
//
// The tie-break on id is not cosmetic: several notifications can share a created_at
// to the microsecond (the assigner writes two in one call), and an unstable order
// would make the list flicker between refetches.
func (r *NotificationRepository) ListForRecipient(ctx context.Context, rec notificationdomain.Recipients, limit int) ([]notificationdomain.Notification, error) {
	const q = `SELECT ` + notificationColumns + `
		FROM notifications
		WHERE ` + recipientPredicate + `
		ORDER BY created_at DESC, id DESC
		LIMIT $3`
	return r.query(ctx, q, rec.AccountID, rec.OfficialID, limit)
}

// CountUnread is the bell's badge. Covered by the partial idx_notifications_unread.
func (r *NotificationRepository) CountUnread(ctx context.Context, rec notificationdomain.Recipients) (int, error) {
	const q = `SELECT COUNT(*) FROM notifications
		WHERE ` + recipientPredicate + ` AND read_at IS NULL`
	var n int
	if err := r.pool.QueryRow(ctx, q, rec.AccountID, rec.OfficialID).Scan(&n); err != nil {
		return 0, fmt.Errorf("notification count unread: %w", err)
	}
	return n, nil
}

// MarkRead marks one message read.
//
// The scope is in the UPDATE itself rather than in a load-then-check, so zero rows
// affected is the ONLY possible answer for someone else's id — there is no branch
// in which the row is fetched, found to belong to another person, and refused
// differently. The handler maps ErrNotificationNotFound to 404 and never 403, so an
// attacker cannot tell an id that exists from one that does not.
//
// There is deliberately no `AND read_at IS NULL`: marking an already-read message
// read again is a no-op that succeeds, so a double-tap is not a 404.
func (r *NotificationRepository) MarkRead(ctx context.Context, id string, rec notificationdomain.Recipients) error {
	const q = `UPDATE notifications SET read_at = COALESCE(read_at, now())
		WHERE id = $3 AND (
			(recipient_kind = 'account'  AND $1 <> '' AND recipient_id = $1)
			OR (recipient_kind = 'official' AND $2 <> '' AND recipient_id = $2)
		)`
	tag, err := r.pool.Exec(ctx, q, rec.AccountID, rec.OfficialID, id)
	if err != nil {
		return fmt.Errorf("notification mark read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notificationdomain.ErrNotificationNotFound
	}
	return nil
}

// MarkAllRead marks every unread message read and reports how many.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, rec notificationdomain.Recipients) (int, error) {
	const q = `UPDATE notifications SET read_at = now()
		WHERE ` + recipientPredicate + ` AND read_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, rec.AccountID, rec.OfficialID)
	if err != nil {
		return 0, fmt.Errorf("notification mark all read: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *NotificationRepository) query(ctx context.Context, q string, args ...any) ([]notificationdomain.Notification, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("notification list: %w", err)
	}
	defer rows.Close()

	out := make([]notificationdomain.Notification, 0)
	for rows.Next() {
		var n notificationdomain.Notification
		var kind, typ string
		if err := rows.Scan(&n.ID, &kind, &n.RecipientID, &typ,
			&n.ProblemID, &n.ProblemTitle, &n.Detail, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("notification scan: %w", err)
		}
		n.RecipientKind = notificationdomain.Kind(kind)
		n.Type = notificationdomain.Type(typ)
		out = append(out, n)
	}
	return out, rows.Err()
}
