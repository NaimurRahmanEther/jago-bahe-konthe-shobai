package postgres

import (
	"context"
	"fmt"

	"jago-bahe-backend/internal/assignment/domain"
)

// This file persists the union admins' advice on above-union forwarding (B20).
// There is no outcome, no quorum and no window to store: advice settles nothing,
// and the super admin may forward at any count. See CLAUDE.md A.3.8.

const suggestionColumns = `id, problem_id, admin_account_id, suggested_official_id, reason, created_at`

// UpsertSuggestion records one admin's advice, replacing their own previous
// advice on the same problem.
//
// The ON CONFLICT target is the one-per-admin unique index, so a change is an
// update rather than a second row. That is the deliberate difference from the
// ballot this replaced: a ballot settled something, so a second cast was refused,
// while advice that settles nothing should be revisable when the other advice
// teaches you something. Each change is audited separately, so the record shows
// that the admin moved.
func (r *AssignmentRepository) UpsertSuggestion(ctx context.Context, s *domain.Suggestion) error {
	const q = `
		INSERT INTO forwarding_suggestions (id, problem_id, admin_account_id, suggested_official_id, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (problem_id, admin_account_id) DO UPDATE
		SET suggested_official_id = EXCLUDED.suggested_official_id,
		    reason                = EXCLUDED.reason,
		    created_at            = EXCLUDED.created_at`
	if _, err := r.pool.Exec(ctx, q,
		s.ID, s.ProblemID, s.AdminAccountID, s.SuggestedOfficialID, s.Reason, s.CreatedAt); err != nil {
		return fmt.Errorf("upsert forwarding suggestion: %w", err)
	}
	return nil
}

// SuggestionsByProblem returns every admin's advice on one problem, oldest first.
func (r *AssignmentRepository) SuggestionsByProblem(ctx context.Context, problemID string) ([]domain.Suggestion, error) {
	q := `SELECT ` + suggestionColumns + ` FROM forwarding_suggestions
		WHERE problem_id = $1 ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, problemID)
	if err != nil {
		return nil, fmt.Errorf("list forwarding suggestions: %w", err)
	}
	defer rows.Close()

	var out []domain.Suggestion
	for rows.Next() {
		s, err := scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SuggestionsByProblems returns the advice on many problems in ONE query, keyed by
// problem id — the page-decoration shape (A.3.4). A queue of twenty reports costs
// one query, never twenty.
func (r *AssignmentRepository) SuggestionsByProblems(ctx context.Context, problemIDs []string) (map[string][]domain.Suggestion, error) {
	out := make(map[string][]domain.Suggestion, len(problemIDs))
	if len(problemIDs) == 0 {
		return out, nil
	}
	q := `SELECT ` + suggestionColumns + ` FROM forwarding_suggestions
		WHERE problem_id = ANY($1) ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, problemIDs)
	if err != nil {
		return nil, fmt.Errorf("list forwarding suggestions by problems: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		s, err := scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		out[s.ProblemID] = append(out[s.ProblemID], s)
	}
	return out, rows.Err()
}

func scanSuggestion(row rowScanner) (domain.Suggestion, error) {
	var s domain.Suggestion
	err := row.Scan(&s.ID, &s.ProblemID, &s.AdminAccountID, &s.SuggestedOfficialID, &s.Reason, &s.CreatedAt)
	return s, err
}

type rowScanner interface{ Scan(dest ...any) error }
