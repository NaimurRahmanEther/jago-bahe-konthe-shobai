// Package postgres implements the assignment Repository (assignment records and
// the union admins' advisory forwarding suggestions) via pgx.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"jago-bahe-backend/internal/assignment/domain"
)

// AssignmentRepository is the pgx-backed implementation of domain.Repository.
type AssignmentRepository struct {
	pool *pgxpool.Pool
}

// NewAssignmentRepository constructs the repository.
func NewAssignmentRepository(pool *pgxpool.Pool) *AssignmentRepository {
	return &AssignmentRepository{pool: pool}
}

var _ domain.Repository = (*AssignmentRepository)(nil)

func (r *AssignmentRepository) CreateAssignment(ctx context.Context, a *domain.Assignment) error {
	const q = `
		INSERT INTO assignments
			(id, problem_id, official_id, monitor_official_id, priority, deadline, override_reason, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, NULLIF($7, ''), $8)`
	_, err := r.pool.Exec(ctx, q,
		a.ID, a.ProblemID, a.OfficialID, a.MonitorOfficialID, a.Priority, a.Deadline, a.OverrideReason, a.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique(problem_id)
			return domain.ErrAlreadyAssigned
		}
		return fmt.Errorf("create assignment: %w", err)
	}
	return nil
}

func (r *AssignmentRepository) GetAssignmentByProblem(ctx context.Context, problemID string) (*domain.Assignment, error) {
	const q = `
		SELECT id, problem_id, official_id, COALESCE(monitor_official_id, ''), priority,
		       deadline, COALESCE(override_reason, ''), created_at
		FROM assignments WHERE problem_id = $1`
	var a domain.Assignment
	err := r.pool.QueryRow(ctx, q, problemID).Scan(
		&a.ID, &a.ProblemID, &a.OfficialID, &a.MonitorOfficialID, &a.Priority,
		&a.Deadline, &a.OverrideReason, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAssignmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	return &a, nil
}

// AssignmentsByProblems resolves a whole page of problems' assignments in one
// query. Problems with no assignment are simply absent from the map — unassigned
// is the normal state of most reports, never an error.
//
// An empty input costs no query at all, mirroring VotesByViewer's guard. The
// UNIQUE (problem_id) index from the assignments migration backs the predicate.
func (r *AssignmentRepository) AssignmentsByProblems(ctx context.Context, problemIDs []string) (map[string]domain.Assignment, error) {
	out := make(map[string]domain.Assignment)
	if len(problemIDs) == 0 {
		return out, nil
	}

	const q = `
		SELECT id, problem_id, official_id, COALESCE(monitor_official_id, ''), priority,
		       deadline, COALESCE(override_reason, ''), created_at
		FROM assignments WHERE problem_id = ANY($1::text[])`
	rows, err := r.pool.Query(ctx, q, problemIDs)
	if err != nil {
		return nil, fmt.Errorf("assignments by problems: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var a domain.Assignment
		if err := rows.Scan(&a.ID, &a.ProblemID, &a.OfficialID, &a.MonitorOfficialID,
			&a.Priority, &a.Deadline, &a.OverrideReason, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		out[a.ProblemID] = a
	}
	return out, rows.Err()
}

func (r *AssignmentRepository) ListByOfficial(ctx context.Context, officialID string) ([]domain.Assignment, error) {
	const q = `
		SELECT id, problem_id, official_id, COALESCE(monitor_official_id, ''), priority,
		       deadline, COALESCE(override_reason, ''), created_at
		FROM assignments WHERE official_id = $1 ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, officialID)
	if err != nil {
		return nil, fmt.Errorf("list assignments by official: %w", err)
	}
	defer rows.Close()

	var out []domain.Assignment
	for rows.Next() {
		var a domain.Assignment
		if err := rows.Scan(
			&a.ID, &a.ProblemID, &a.OfficialID, &a.MonitorOfficialID, &a.Priority,
			&a.Deadline, &a.OverrideReason, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
