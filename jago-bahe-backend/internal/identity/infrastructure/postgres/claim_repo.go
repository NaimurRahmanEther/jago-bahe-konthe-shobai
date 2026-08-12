package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	identitydomain "jago-bahe-backend/internal/identity/domain"
)

// ClaimRepository is the pgx-backed implementation of the claim port.
type ClaimRepository struct {
	pool *pgxpool.Pool
}

// NewClaimRepository constructs the repository.
func NewClaimRepository(pool *pgxpool.Pool) *ClaimRepository {
	return &ClaimRepository{pool: pool}
}

var _ identitydomain.ClaimRepository = (*ClaimRepository)(nil)

const claimColumns = `id, account_id, official_id, status, COALESCE(reviewed_by, ''), reviewed_at,
	COALESCE(reason, ''), created_at`

func (r *ClaimRepository) Create(ctx context.Context, c *identitydomain.OfficialClaim) error {
	const q = `
		INSERT INTO official_claims (id, account_id, official_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, q, c.ID, c.AccountID, c.OfficialID, string(c.Status), c.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique(account_id)
			return identitydomain.ErrAlreadyClaimed
		}
		return fmt.Errorf("claim create: %w", err)
	}
	return nil
}

func (r *ClaimRepository) GetByID(ctx context.Context, id string) (*identitydomain.OfficialClaim, error) {
	q := `SELECT ` + claimColumns + ` FROM official_claims WHERE id = $1`
	c, err := scanClaim(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identitydomain.ErrClaimNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *ClaimRepository) ListPending(ctx context.Context) ([]identitydomain.OfficialClaim, error) {
	q := `SELECT ` + claimColumns + ` FROM official_claims WHERE status = 'Pending' ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("claim list pending: %w", err)
	}
	defer rows.Close()

	var out []identitydomain.OfficialClaim
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// Decide records an approval or rejection.
//
// The `status = 'Pending'` guard lives in the same statement that performs the
// update, so two reviewers acting on one claim cannot both succeed — the loser is
// told it was already decided rather than silently overwriting the winner.
//
// A unique-violation here means the partial index on approved claims refused a
// second holder for the office: someone else has already been confirmed as this
// person. That is a real answer, not a database accident, so it surfaces as
// ErrOfficeTaken.
func (r *ClaimRepository) Decide(ctx context.Context, claimID string, to identitydomain.ClaimStatus, reviewedBy, reason string, at time.Time) error {
	const q = `
		UPDATE official_claims
		SET status = $2, reviewed_by = $3, reason = NULLIF($4, ''), reviewed_at = $5
		WHERE id = $1 AND status = 'Pending'`
	ct, err := r.pool.Exec(ctx, q, claimID, string(to), reviewedBy, reason, at)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // uq_official_claims_approved
			return identitydomain.ErrOfficeTaken
		}
		return fmt.Errorf("claim decide: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return identitydomain.ErrClaimNotPending
	}
	return nil
}

func scanClaim(row rowScanner) (*identitydomain.OfficialClaim, error) {
	var c identitydomain.OfficialClaim
	var status string
	if err := row.Scan(&c.ID, &c.AccountID, &c.OfficialID, &status, &c.ReviewedBy, &c.ReviewedAt,
		&c.Reason, &c.CreatedAt); err != nil {
		return nil, err
	}
	c.Status = identitydomain.ClaimStatus(status)
	return &c, nil
}
