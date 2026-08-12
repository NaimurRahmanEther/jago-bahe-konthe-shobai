// Package postgres implements the identity repositories via pgx.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	identitydomain "jago-bahe-backend/internal/identity/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// AccountRepository is the pgx-backed implementation of the account port.
type AccountRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepository constructs the repository.
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

var _ identitydomain.AccountRepository = (*AccountRepository)(nil)

func (r *AccountRepository) Create(ctx context.Context, a *identitydomain.Account) error {
	const q = `
		INSERT INTO accounts (id, name, phone, password_hash, role, nid, union_id, verified, official_id, created_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, NULLIF($9, ''), $10)`
	_, err := r.pool.Exec(ctx, q,
		a.ID, a.Name, a.Phone.String(), a.PasswordHash, string(a.Role),
		a.NID, a.UnionID.String(), a.Verified, a.OfficialID, a.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation (phone)
				return identitydomain.ErrPhoneTaken
			case "23503": // foreign_key_violation (union_id)
				return identitydomain.ErrInvalidUnion
			}
		}
		return fmt.Errorf("account create: %w", err)
	}
	return nil
}

func (r *AccountRepository) GetByPhone(ctx context.Context, phone valueobject.PhoneNumber) (*identitydomain.Account, error) {
	const q = `
		SELECT id, name, phone, password_hash, role,
		       COALESCE(nid, ''), COALESCE(union_id, ''), verified, COALESCE(official_id, ''), created_at
		FROM accounts WHERE phone = $1`
	return r.scanOne(ctx, q, phone.String())
}

func (r *AccountRepository) GetByID(ctx context.Context, id string) (*identitydomain.Account, error) {
	const q = `
		SELECT id, name, phone, password_hash, role,
		       COALESCE(nid, ''), COALESCE(union_id, ''), verified, COALESCE(official_id, ''), created_at
		FROM accounts WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *AccountRepository) SetVerified(ctx context.Context, id string, verified bool) error {
	ct, err := r.pool.Exec(ctx, `UPDATE accounts SET verified = $2 WHERE id = $1`, id, verified)
	if err != nil {
		return fmt.Errorf("account set verified: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return identitydomain.ErrAccountNotFound
	}
	return nil
}

// SetOfficialID binds an account to a directory office. Called only by an
// approved claim — the sole runtime path to this link, which every official-scoped
// route and all admin scoping depend on.
func (r *AccountRepository) SetOfficialID(ctx context.Context, accountID, officialID string) error {
	ct, err := r.pool.Exec(ctx, `UPDATE accounts SET official_id = $2 WHERE id = $1`, accountID, officialID)
	if err != nil {
		return fmt.Errorf("account set official id: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return identitydomain.ErrAccountNotFound
	}
	return nil
}

// ListUnverifiedResidents returns a union's unverified residents — the queue that
// union's admin works through. Scoped to residents because an official's standing
// comes from a claim, not from this flag.
func (r *AccountRepository) ListUnverifiedResidents(ctx context.Context, unionID valueobject.AreaID) ([]identitydomain.Account, error) {
	const q = `
		SELECT id, name, phone, password_hash, role,
		       COALESCE(nid, ''), COALESCE(union_id, ''), verified, COALESCE(official_id, ''), created_at
		FROM accounts
		WHERE role = 'resident' AND verified = false AND union_id = $1
		ORDER BY created_at`
	return r.queryAccounts(ctx, q, unionID.String())
}

func (r *AccountRepository) ListByRole(ctx context.Context, role identitydomain.Role) ([]identitydomain.Account, error) {
	const q = `
		SELECT id, name, phone, password_hash, role,
		       COALESCE(nid, ''), COALESCE(union_id, ''), verified, COALESCE(official_id, ''), created_at
		FROM accounts WHERE role = $1 ORDER BY id`
	return r.queryAccounts(ctx, q, string(role))
}

// NamesByIDs resolves account ids to names in one query. Mirrors the problem
// repository's VotesByViewer: empty input costs no query at all, and unknown ids
// are absent from the map rather than an error — the activity feed's actors
// include directory offices and "system", which are not accounts.
func (r *AccountRepository) NamesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string)
	if len(ids) == 0 {
		return out, nil
	}

	const q = `SELECT id, name FROM accounts WHERE id = ANY($1::text[])`
	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("account names by ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan account name: %w", err)
		}
		out[id] = name
	}
	return out, rows.Err()
}

// queryAccounts runs an account SELECT with the standard column list.
func (r *AccountRepository) queryAccounts(ctx context.Context, q string, args ...any) ([]identitydomain.Account, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("account query: %w", err)
	}
	defer rows.Close()

	var out []identitydomain.Account
	for rows.Next() {
		var a identitydomain.Account
		var phone, roleStr, union string
		if err := rows.Scan(
			&a.ID, &a.Name, &phone, &a.PasswordHash, &roleStr,
			&a.NID, &union, &a.Verified, &a.OfficialID, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("account scan: %w", err)
		}
		a.Phone = valueobject.PhoneNumber(phone)
		a.Role = identitydomain.Role(roleStr)
		a.UnionID = valueobject.AreaID(union)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AccountRepository) scanOne(ctx context.Context, q string, arg string) (*identitydomain.Account, error) {
	var a identitydomain.Account
	var phone, role, union string
	err := r.pool.QueryRow(ctx, q, arg).Scan(
		&a.ID, &a.Name, &phone, &a.PasswordHash, &role,
		&a.NID, &union, &a.Verified, &a.OfficialID, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identitydomain.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account scan: %w", err)
	}
	a.Phone = valueobject.PhoneNumber(phone)
	a.Role = identitydomain.Role(role)
	a.UnionID = valueobject.AreaID(union)
	return &a, nil
}
