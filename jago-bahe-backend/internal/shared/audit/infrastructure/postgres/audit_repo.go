// Package postgres implements the append-only audit Repository via pgx.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// AuditRepository is the pgx-backed implementation of auditdomain.Repository.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository constructs an AuditRepository over the given pool.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

var _ auditdomain.Repository = (*AuditRepository)(nil)

// Append inserts a new entry. There is no update path, and the only delete path is
// DeleteByTarget below — the table is append-only apart from that one exception.
func (r *AuditRepository) Append(ctx context.Context, e auditdomain.AuditEntry) error {
	const q = `
		INSERT INTO audit_entries (id, target_type, target_id, actor, action, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)`
	_, err := r.pool.Exec(ctx, q, e.ID, e.TargetType, e.TargetID, e.Actor, e.Action, e.Reason, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("audit append: %w", err)
	}
	return nil
}

// DeleteByTarget erases every entry for one target. See the port's doc comment for
// why this exists and why nothing but the reporter's hard delete may call it.
//
// Deleting no rows is not an error: a problem with no audit entries is unusual but
// not impossible, and the caller has already removed the row this trail described —
// failing here would report a problem that no longer exists.
//
// idx_audit_target (target_type, target_id, created_at) covers this predicate.
func (r *AuditRepository) DeleteByTarget(ctx context.Context, targetType, targetID string) error {
	const q = `DELETE FROM audit_entries WHERE target_type = $1 AND target_id = $2`
	if _, err := r.pool.Exec(ctx, q, targetType, targetID); err != nil {
		return fmt.Errorf("audit delete by target: %w", err)
	}
	return nil
}

const auditColumns = `id, target_type, target_id, actor, action, COALESCE(reason, ''), created_at`

// ListByTarget returns a target's entries, oldest first.
func (r *AuditRepository) ListByTarget(ctx context.Context, targetType, targetID string) ([]auditdomain.AuditEntry, error) {
	const q = `SELECT ` + auditColumns + `
		FROM audit_entries
		WHERE target_type = $1 AND target_id = $2
		ORDER BY created_at ASC`
	return r.query(ctx, q, targetType, targetID)
}

// ListByActions returns recent entries for any of the given actions, newest
// first — the super admin's oversight feed. Newest-first here, unlike
// ListByTarget's oldest-first: a trail is read as a story from the start, but
// oversight is read to see what just happened.
func (r *AuditRepository) ListByActions(ctx context.Context, actions []string, limit int) ([]auditdomain.AuditEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `SELECT ` + auditColumns + `
		FROM audit_entries
		WHERE action = ANY($1::text[])
		ORDER BY created_at DESC
		LIMIT $2`
	return r.query(ctx, q, actions, limit)
}

func (r *AuditRepository) query(ctx context.Context, q string, args ...any) ([]auditdomain.AuditEntry, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("audit list: %w", err)
	}
	defer rows.Close()

	var out []auditdomain.AuditEntry
	for rows.Next() {
		var e auditdomain.AuditEntry
		if err := rows.Scan(&e.ID, &e.TargetType, &e.TargetID, &e.Actor, &e.Action, &e.Reason, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("audit scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
