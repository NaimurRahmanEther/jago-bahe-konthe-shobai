package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	identitydomain "jago-bahe-backend/internal/identity/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// OfficialRepository is the pgx-backed read model for the officials directory.
type OfficialRepository struct {
	pool *pgxpool.Pool
}

// NewOfficialRepository constructs the repository.
func NewOfficialRepository(pool *pgxpool.Pool) *OfficialRepository {
	return &OfficialRepository{pool: pool}
}

var _ identitydomain.OfficialRepository = (*OfficialRepository)(nil)

func (r *OfficialRepository) List(ctx context.Context) ([]identitydomain.Official, error) {
	const q = `SELECT id, name, phone, tier, area_id FROM officials ORDER BY id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("official list: %w", err)
	}
	defer rows.Close()

	var out []identitydomain.Official
	for rows.Next() {
		o, err := scanOfficial(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (r *OfficialRepository) GetByID(ctx context.Context, id string) (*identitydomain.Official, error) {
	const q = `SELECT id, name, phone, tier, area_id FROM officials WHERE id = $1`
	o, err := scanOfficial(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identitydomain.ErrOfficialNotFound
	}
	if err != nil {
		return nil, err
	}
	return o, nil
}

// NamesByIDs resolves directory office ids to office-holder names in one query.
// The companion to AccountRepository.NamesByIDs: an audit entry's actor is an
// account id for an admin's action and an OFFICE id for an official's, so the
// activity feed resolves against both.
func (r *OfficialRepository) NamesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string)
	if len(ids) == 0 {
		return out, nil
	}

	const q = `SELECT id, name FROM officials WHERE id = ANY($1::text[])`
	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("official names by ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan official name: %w", err)
		}
		out[id] = name
	}
	return out, rows.Err()
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanOfficial(row rowScanner) (*identitydomain.Official, error) {
	var o identitydomain.Official
	var phone, tier, area string
	if err := row.Scan(&o.ID, &o.Name, &phone, &tier, &area); err != nil {
		return nil, err
	}
	o.Phone = valueobject.PhoneNumber(phone)
	o.Tier = valueobject.Tier(tier)
	o.AreaID = valueobject.AreaID(area)
	return &o, nil
}
