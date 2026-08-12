// Package postgres implements the suggestion Repository (suggestions + their
// upvotes) via pgx. UpvoteCount is computed live with a COUNT subquery so reads
// always reflect the current tally; only the votes are stored.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	suggestiondomain "jago-bahe-backend/internal/suggestion/domain"
)

// SuggestionRepository is the pgx-backed implementation of the domain Repository.
type SuggestionRepository struct {
	pool *pgxpool.Pool
}

// NewSuggestionRepository constructs the repository.
func NewSuggestionRepository(pool *pgxpool.Pool) *SuggestionRepository {
	return &SuggestionRepository{pool: pool}
}

var _ suggestiondomain.Repository = (*SuggestionRepository)(nil)

// selectColumns projects a suggestion row with its live upvote count.
const selectColumns = `
	s.id, s.problem_id, s.author_id, s.text,
	(SELECT COUNT(*) FROM suggestion_votes v WHERE v.suggestion_id = s.id) AS upvote_count,
	s.created_at`

func (r *SuggestionRepository) Create(ctx context.Context, s *suggestiondomain.Suggestion) error {
	const q = `INSERT INTO suggestions (id, problem_id, author_id, text, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, q, s.ID, s.ProblemID, s.AuthorID, s.Text, s.CreatedAt)
	if err != nil {
		return fmt.Errorf("suggestion create: %w", err)
	}
	return nil
}

func (r *SuggestionRepository) GetByID(ctx context.Context, id string) (*suggestiondomain.Suggestion, error) {
	q := `SELECT ` + selectColumns + ` FROM suggestions s WHERE s.id = $1`
	s, err := scanSuggestion(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, suggestiondomain.ErrSuggestionNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *SuggestionRepository) ListByProblem(ctx context.Context, problemID string) ([]suggestiondomain.Suggestion, error) {
	q := `SELECT ` + selectColumns + ` FROM suggestions s
		WHERE s.problem_id = $1
		ORDER BY upvote_count DESC, s.created_at ASC`
	rows, err := r.pool.Query(ctx, q, problemID)
	if err != nil {
		return nil, fmt.Errorf("suggestion list: %w", err)
	}
	defer rows.Close()

	var out []suggestiondomain.Suggestion
	for rows.Next() {
		s, err := scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *SuggestionRepository) HasUpvoted(ctx context.Context, suggestionID, voterID string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM suggestion_votes WHERE suggestion_id = $1 AND voter_id = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, q, suggestionID, voterID).Scan(&exists); err != nil {
		return false, fmt.Errorf("has upvoted: %w", err)
	}
	return exists, nil
}

// UpvotesByViewer returns the viewer's own upvotes across the given suggestions.
//
// One query for a whole list, mirroring ProblemRepository.VotesByViewer. An
// anonymous viewer costs no query at all. The UNIQUE (suggestion_id, voter_id)
// index backs the suggestion_id half of the predicate.
//
// Only rows belonging to this viewer are ever selected — the upvote COUNTS are
// public, but who cast them is not, and a query that could return another
// account's row is one refactor away from publishing it.
func (r *SuggestionRepository) UpvotesByViewer(ctx context.Context, viewerID string, suggestionIDs []string) (map[string]bool, error) {
	out := make(map[string]bool)
	if viewerID == "" || len(suggestionIDs) == 0 {
		return out, nil
	}

	const q = `SELECT suggestion_id FROM suggestion_votes WHERE voter_id = $1 AND suggestion_id = ANY($2::text[])`
	rows, err := r.pool.Query(ctx, q, viewerID, suggestionIDs)
	if err != nil {
		return nil, fmt.Errorf("upvotes by viewer: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var suggestionID string
		if err := rows.Scan(&suggestionID); err != nil {
			return nil, fmt.Errorf("scan viewer upvote: %w", err)
		}
		out[suggestionID] = true
	}
	return out, rows.Err()
}

func (r *SuggestionRepository) AddUpvote(ctx context.Context, v suggestiondomain.SuggestionVote) error {
	const q = `INSERT INTO suggestion_votes (id, suggestion_id, voter_id) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, q, v.ID, v.SuggestionID, v.VoterID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique(suggestion_id, voter_id)
			return nil // idempotent: already upvoted
		}
		return fmt.Errorf("add upvote: %w", err)
	}
	return nil
}

func (r *SuggestionRepository) RemoveUpvote(ctx context.Context, suggestionID, voterID string) error {
	const q = `DELETE FROM suggestion_votes WHERE suggestion_id = $1 AND voter_id = $2`
	if _, err := r.pool.Exec(ctx, q, suggestionID, voterID); err != nil {
		return fmt.Errorf("remove upvote: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSuggestion(row rowScanner) (*suggestiondomain.Suggestion, error) {
	var s suggestiondomain.Suggestion
	if err := row.Scan(&s.ID, &s.ProblemID, &s.AuthorID, &s.Text, &s.UpvoteCount, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}
