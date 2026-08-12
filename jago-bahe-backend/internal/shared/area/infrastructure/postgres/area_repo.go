// Package postgres implements the area Repository via pgx. It is the only place
// area domain types touch SQL; the domain layer never imports this package.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	areadomain "jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// AreaRepository is the pgx-backed implementation of areadomain.Repository.
type AreaRepository struct {
	pool *pgxpool.Pool
}

// NewAreaRepository constructs an AreaRepository over the given pool.
func NewAreaRepository(pool *pgxpool.Pool) *AreaRepository {
	return &AreaRepository{pool: pool}
}

var _ areadomain.Repository = (*AreaRepository)(nil)

func (r *AreaRepository) GetByID(ctx context.Context, id valueobject.AreaID) (*areadomain.Area, error) {
	const q = `SELECT id, name, level, COALESCE(parent_id, '') FROM areas WHERE id = $1`
	var a areadomain.Area
	var lvl, parent string
	err := r.pool.QueryRow(ctx, q, id.String()).Scan(&a.ID, &a.Name, &lvl, &parent)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, areadomain.ErrAreaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("area get: %w", err)
	}
	a.Level = areadomain.Level(lvl)
	a.ParentID = valueobject.AreaID(parent)
	return &a, nil
}

func (r *AreaRepository) List(ctx context.Context) ([]areadomain.Area, error) {
	const q = `SELECT id, name, level, COALESCE(parent_id, '') FROM areas ORDER BY level, id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("area list: %w", err)
	}
	defer rows.Close()
	return scanAreas(rows)
}

func (r *AreaRepository) Children(ctx context.Context, parentID valueobject.AreaID) ([]areadomain.Area, error) {
	const q = `SELECT id, name, level, COALESCE(parent_id, '') FROM areas WHERE parent_id = $1 ORDER BY id`
	rows, err := r.pool.Query(ctx, q, parentID.String())
	if err != nil {
		return nil, fmt.Errorf("area children: %w", err)
	}
	defer rows.Close()
	return scanAreas(rows)
}

func scanAreas(rows pgx.Rows) ([]areadomain.Area, error) {
	var out []areadomain.Area
	for rows.Next() {
		var a areadomain.Area
		var lvl, parent string
		if err := rows.Scan(&a.ID, &a.Name, &lvl, &parent); err != nil {
			return nil, fmt.Errorf("area scan: %w", err)
		}
		a.Level = areadomain.Level(lvl)
		a.ParentID = valueobject.AreaID(parent)
		out = append(out, a)
	}
	return out, rows.Err()
}
