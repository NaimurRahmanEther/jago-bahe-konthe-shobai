package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"jago-bahe-backend/internal/resolution/domain"
)

// This file is the persistence half of the observation ladder (B19): the read side
// of escalation, where a case its official has gone silent on surfaces to the
// official above them. See internal/resolution/domain/observation.go for the rules.

const observationColumns = `id, case_id, observer_official_id, level, opened_at, resolved_at`

// ListByMonitor returns the unfinished cases the official directly monitors.
//
// Resolved and Disputed cases are excluded because they are finished — nobody is
// waiting on the official — but nothing else is filtered: monitoring is continuous
// from assignment, not something that begins when a deadline is missed, precisely
// so a higher authority can never claim they did not know (Concept §7).
func (r *CaseRepository) ListByMonitor(ctx context.Context, officialID string) ([]domain.Case, error) {
	q := `SELECT ` + caseColumns + ` FROM cases
		WHERE monitor_official_id = $1 AND status NOT IN ('Resolved', 'Disputed')
		ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, officialID)
	if err != nil {
		return nil, fmt.Errorf("list cases by monitor: %w", err)
	}
	defer rows.Close()

	var cases []domain.Case
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		cases = append(cases, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range cases {
		if err := r.loadChildren(ctx, &cases[i]); err != nil {
			return nil, err
		}
	}
	return cases, nil
}

// CasesByIDs loads whole cases by id, for the rows a higher rung sees only because
// an observation was opened against them (they are not its direct monitor).
func (r *CaseRepository) CasesByIDs(ctx context.Context, caseIDs []string) ([]domain.Case, error) {
	if len(caseIDs) == 0 {
		return nil, nil
	}
	q := `SELECT ` + caseColumns + ` FROM cases WHERE id = ANY($1) ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, caseIDs)
	if err != nil {
		return nil, fmt.Errorf("cases by ids: %w", err)
	}
	defer rows.Close()

	var cases []domain.Case
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		cases = append(cases, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range cases {
		if err := r.loadChildren(ctx, &cases[i]); err != nil {
			return nil, err
		}
	}
	return cases, nil
}

// ListObservationsByObserver returns every observation opened against the official,
// open and resolved alike — the resolved ones are the history of silences that were
// eventually answered, which is exactly what makes a pattern visible.
func (r *CaseRepository) ListObservationsByObserver(ctx context.Context, officialID string) ([]domain.Observation, error) {
	q := `SELECT ` + observationColumns + ` FROM case_observations
		WHERE observer_official_id = $1 ORDER BY opened_at DESC`
	return r.queryObservations(ctx, q, officialID)
}

// ListObservationsByCases loads many cases' observations in ONE query — the
// page-decoration shape (A.3.4). Never call it per row.
func (r *CaseRepository) ListObservationsByCases(ctx context.Context, caseIDs []string) ([]domain.Observation, error) {
	if len(caseIDs) == 0 {
		return nil, nil
	}
	q := `SELECT ` + observationColumns + ` FROM case_observations
		WHERE case_id = ANY($1) ORDER BY opened_at DESC`
	return r.queryObservations(ctx, q, caseIDs)
}

func (r *CaseRepository) queryObservations(ctx context.Context, q string, arg any) ([]domain.Observation, error) {
	rows, err := r.pool.Query(ctx, q, arg)
	if err != nil {
		return nil, fmt.Errorf("list observations: %w", err)
	}
	defer rows.Close()

	var out []domain.Observation
	for rows.Next() {
		var o domain.Observation
		if err := rows.Scan(&o.ID, &o.CaseID, &o.ObserverOfficialID, &o.Level, &o.OpenedAt, &o.ResolvedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.loadObservationNotes(ctx, out)
}

// loadObservationNotes attaches every note to its observation in one query, for the
// same reason the caller batches the observations themselves.
func (r *CaseRepository) loadObservationNotes(ctx context.Context, obs []domain.Observation) error {
	if len(obs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(obs))
	for i := range obs {
		ids = append(ids, obs[i].ID)
	}
	const q = `SELECT id, observation_id, text, created_at FROM observation_notes
		WHERE observation_id = ANY($1) ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		return fmt.Errorf("load observation notes: %w", err)
	}
	defer rows.Close()

	byObservation := make(map[string][]domain.ObservationNote)
	for rows.Next() {
		var n domain.ObservationNote
		if err := rows.Scan(&n.ID, &n.ObservationID, &n.Text, &n.CreatedAt); err != nil {
			return err
		}
		byObservation[n.ObservationID] = append(byObservation[n.ObservationID], n)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range obs {
		obs[i].Notes = byObservation[obs[i].ID]
	}
	return nil
}

// OpenObservation inserts a rung and reports whether it actually inserted.
//
// The ON CONFLICT target names the partial unique index's predicate, so a rung
// already OPEN on this case is a no-op while a resolved one does not block a fresh
// insert. That is what makes the worker safe to run every minute — and what makes a
// case that went quiet, was answered, and went quiet again open rung 1 a second
// time as a new row. The bool is what keeps the audit entry firing exactly once.
func (r *CaseRepository) OpenObservation(ctx context.Context, o *domain.Observation) (bool, error) {
	const q = `
		INSERT INTO case_observations (id, case_id, observer_official_id, level, opened_at, resolved_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NULL, $6)
		ON CONFLICT (case_id, level) WHERE resolved_at IS NULL DO NOTHING`
	ct, err := r.pool.Exec(ctx, q, o.ID, o.CaseID, o.ObserverOfficialID, o.Level, o.OpenedAt, time.Now().UTC())
	if err != nil {
		return false, fmt.Errorf("open observation: %w", err)
	}
	return ct.RowsAffected() > 0, nil
}

// ResolveOpenObservations closes every open observation on a case, returning how
// many closed. The rows are kept: that an official was silent for eleven days is a
// fact about the public record, and deleting it on response would erase it.
func (r *CaseRepository) ResolveOpenObservations(ctx context.Context, caseID string, at time.Time) (int, error) {
	const q = `UPDATE case_observations SET resolved_at = $2 WHERE case_id = $1 AND resolved_at IS NULL`
	ct, err := r.pool.Exec(ctx, q, caseID, at)
	if err != nil {
		return 0, fmt.Errorf("resolve observations: %w", err)
	}
	return int(ct.RowsAffected()), nil
}

// GetObservation loads one observation with its notes.
func (r *CaseRepository) GetObservation(ctx context.Context, observationID string) (*domain.Observation, error) {
	q := `SELECT ` + observationColumns + ` FROM case_observations WHERE id = $1`
	var o domain.Observation
	err := r.pool.QueryRow(ctx, q, observationID).
		Scan(&o.ID, &o.CaseID, &o.ObserverOfficialID, &o.Level, &o.OpenedAt, &o.ResolvedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrObservationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get observation: %w", err)
	}
	one := []domain.Observation{o}
	if err := r.loadObservationNotes(ctx, one); err != nil {
		return nil, err
	}
	return &one[0], nil
}

// AddObservationNote appends a monitor's note. There is no update and no delete:
// notes are append-only, like every other public reason on this platform.
func (r *CaseRepository) AddObservationNote(ctx context.Context, n domain.ObservationNote) error {
	const q = `INSERT INTO observation_notes (id, observation_id, text, created_at) VALUES ($1, $2, $3, $4)`
	if _, err := r.pool.Exec(ctx, q, n.ID, n.ObservationID, n.Text, n.CreatedAt); err != nil {
		return fmt.Errorf("add observation note: %w", err)
	}
	return nil
}

// LastActivityByCases returns each case's last ASSIGNEE activity, in one query.
//
// It states the same rule as domain.Case.LastActivityAt in SQL, so the worker's
// per-minute scan need not load whole aggregates for every open case in the seat.
// The duplication is deliberate and is the same trade the codebase already makes
// for Case.BlockedOnHigherAuthority vs scorecard.FairnessFlag; as there, a
// cross-implementation test pins the two to the same answer.
//
// The omissions ARE the rule and must be kept in step with the domain method:
// obstacles.adjudicated_at and obstacles.resolved_at are the ADMIN's acts,
// obstacle_votes and unblocking_plans are the PUBLIC judging, and observation_notes
// are the MONITOR writing. None of them is the official answering, so none of them
// may restart the clock that measures the official's silence.
func (r *CaseRepository) LastActivityByCases(ctx context.Context, caseIDs []string) (map[string]time.Time, error) {
	if len(caseIDs) == 0 {
		return map[string]time.Time{}, nil
	}
	const q = `
		SELECT case_id, MAX(acted_at) FROM (
			SELECT id AS case_id, acknowledged_at AS acted_at FROM cases
				WHERE id = ANY($1) AND acknowledged_at IS NOT NULL
			UNION ALL
			SELECT case_id, created_at FROM plans WHERE case_id = ANY($1)
			UNION ALL
			SELECT p.case_id, t.completed_at FROM plan_tasks t
				JOIN plans p ON p.id = t.plan_id
				WHERE p.case_id = ANY($1) AND t.completed_at IS NOT NULL
			UNION ALL
			SELECT case_id, created_at FROM progress_updates WHERE case_id = ANY($1)
			UNION ALL
			SELECT case_id, created_at FROM evidence WHERE case_id = ANY($1)
			UNION ALL
			SELECT case_id, created_at FROM obstacles WHERE case_id = ANY($1)
		) acts
		GROUP BY case_id`
	rows, err := r.pool.Query(ctx, q, caseIDs)
	if err != nil {
		return nil, fmt.Errorf("last activity by cases: %w", err)
	}
	defer rows.Close()

	out := make(map[string]time.Time, len(caseIDs))
	for rows.Next() {
		var id string
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, err
		}
		out[id] = at
	}
	return out, rows.Err()
}
