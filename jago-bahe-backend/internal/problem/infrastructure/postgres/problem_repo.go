// Package postgres implements the problem Repository (problems + their
// validation votes) via pgx.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	problemdomain "jago-bahe-backend/internal/problem/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// ProblemRepository is the pgx-backed implementation of problemdomain.Repository.
type ProblemRepository struct {
	pool *pgxpool.Pool
}

// NewProblemRepository constructs the repository.
func NewProblemRepository(pool *pgxpool.Pool) *ProblemRepository {
	return &ProblemRepository{pool: pool}
}

var _ problemdomain.Repository = (*ProblemRepository)(nil)

const problemColumns = `id, title, description, area_id, COALESCE(address, ''), lat, lng,
	reporter_id, pointed_official_id, COALESCE(proposed_solution, ''), COALESCE(image_url, ''),
	status, valid_count, created_at,
	COALESCE(rejection_reason, ''), COALESCE(reviewed_by, ''), reviewed_at`

func (r *ProblemRepository) Create(ctx context.Context, p *problemdomain.Problem) error {
	const q = `
		INSERT INTO problems
			(id, title, description, area_id, address, lat, lng, reporter_id,
			 pointed_official_id, proposed_solution, image_url, status, valid_count, created_at)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9,NULLIF($10,''),NULLIF($11,''),$12,$13,$14)`
	_, err := r.pool.Exec(ctx, q,
		p.ID, p.Title, p.Description, p.Location.AreaID.String(), p.Location.Address, p.Location.Lat, p.Location.Lng,
		p.ReporterID, p.PointedOfficialID, p.ProposedSolution, p.ImageURL, string(p.Status), p.ValidCount, p.CreatedAt)
	if err != nil {
		return fmt.Errorf("problem create: %w", err)
	}
	return nil
}

func (r *ProblemRepository) GetByID(ctx context.Context, id string) (*problemdomain.Problem, error) {
	q := `SELECT ` + problemColumns + ` FROM problems WHERE id = $1`
	p, err := scanProblem(r.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, problemdomain.ErrProblemNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProblemRepository) List(ctx context.Context, f problemdomain.Filter) ([]problemdomain.Problem, error) {
	// Optional filters via a NULL/empty sentinel so a single prepared query serves
	// all. Statuses is a set rather than one value because callers ask for state
	// groups — the feed for PublicStatuses(), the screening queue for
	// PendingApproval — which equality cannot express. A nil Statuses means no
	// status restriction.
	var statuses []string
	for _, s := range f.Statuses {
		statuses = append(statuses, string(s))
	}

	q := `SELECT ` + problemColumns + ` FROM problems
		WHERE ($1 = '' OR area_id = $1)
		  AND ($2::text[] IS NULL OR status = ANY($2::text[]))
		  AND ($3 = '' OR pointed_official_id = $3)
		  AND ($4 = '' OR reporter_id = $4)
		ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, f.AreaID, statuses, f.OfficialID, f.ReporterID)
	if err != nil {
		return nil, fmt.Errorf("problem list: %w", err)
	}
	defer rows.Close()

	var out []problemdomain.Problem
	for rows.Next() {
		p, err := scanProblem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *ProblemRepository) AddVote(ctx context.Context, v problemdomain.ValidationVote) error {
	const q = `
		INSERT INTO validation_votes (id, problem_id, voter_id, vote)
		VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, q, v.ID, v.ProblemID, v.VoterID, string(v.Vote))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique(problem_id, voter_id)
			return problemdomain.ErrAlreadyVoted
		}
		return fmt.Errorf("add vote: %w", err)
	}
	return nil
}

func (r *ProblemRepository) ListVotes(ctx context.Context, problemID string) ([]problemdomain.ValidationVote, error) {
	const q = `SELECT id, problem_id, voter_id, vote, created_at FROM validation_votes WHERE problem_id = $1`
	rows, err := r.pool.Query(ctx, q, problemID)
	if err != nil {
		return nil, fmt.Errorf("list votes: %w", err)
	}
	defer rows.Close()

	var out []problemdomain.ValidationVote
	for rows.Next() {
		var v problemdomain.ValidationVote
		var choice string
		if err := rows.Scan(&v.ID, &v.ProblemID, &v.VoterID, &choice, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan vote: %w", err)
		}
		v.Vote = problemdomain.VoteChoice(choice)
		out = append(out, v)
	}
	return out, rows.Err()
}

// VotesByViewer returns the viewer's own vote on each of the given problems.
//
// One query for a whole page: the alternative — a correlated subquery appended to
// problemColumns — would push the viewer id into GetByID and List, which are
// caller-free by design, and land a per-viewer value on the shared scanProblem
// path where every other loader would silently leave it zero.
//
// An anonymous viewer costs no query at all. The UNIQUE (problem_id, voter_id)
// index from 000006 backs the problem_id half of the predicate; there is no index
// on voter_id alone and none is needed at one seat's scale.
func (r *ProblemRepository) VotesByViewer(ctx context.Context, viewerID string, problemIDs []string) (map[string]problemdomain.VoteChoice, error) {
	out := make(map[string]problemdomain.VoteChoice)
	if viewerID == "" || len(problemIDs) == 0 {
		return out, nil
	}

	const q = `SELECT problem_id, vote FROM validation_votes WHERE voter_id = $1 AND problem_id = ANY($2::text[])`
	rows, err := r.pool.Query(ctx, q, viewerID, problemIDs)
	if err != nil {
		return nil, fmt.Errorf("votes by viewer: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var problemID, choice string
		if err := rows.Scan(&problemID, &choice); err != nil {
			return nil, fmt.Errorf("scan viewer vote: %w", err)
		}
		out[problemID] = problemdomain.VoteChoice(choice)
	}
	return out, rows.Err()
}

// TitlesByIDs resolves problem ids to titles for the public activity feed.
//
// The status predicate is the point, not housekeeping. Filtering to publicly
// visible statuses in the QUERY means a PendingApproval report's title cannot
// reach the seat-wide feed even if an action were somehow logged against one —
// the guarantee holds without every caller remembering to check, which is the
// difference between a safety property and a convention. An unresolved id is
// absent from the map and the row renders without a title.
func (r *ProblemRepository) TitlesByIDs(ctx context.Context, problemIDs []string) (map[string]string, error) {
	out := make(map[string]string)
	if len(problemIDs) == 0 {
		return out, nil
	}

	public := problemdomain.PublicStatuses()
	statuses := make([]string, 0, len(public))
	for _, s := range public {
		statuses = append(statuses, string(s))
	}

	const q = `SELECT id, title FROM problems WHERE id = ANY($1::text[]) AND status = ANY($2::text[])`
	rows, err := r.pool.Query(ctx, q, problemIDs, statuses)
	if err != nil {
		return nil, fmt.Errorf("problem titles by ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, fmt.Errorf("scan problem title: %w", err)
		}
		out[id] = title
	}
	return out, rows.Err()
}

func (r *ProblemRepository) UpdateValidation(ctx context.Context, problemID string, validCount int, status problemdomain.Status) error {
	const q = `UPDATE problems SET valid_count = $2, status = $3 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, problemID, validCount, string(status))
	if err != nil {
		return fmt.Errorf("update validation: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return problemdomain.ErrProblemNotFound
	}
	return nil
}

func (r *ProblemRepository) SetStatus(ctx context.Context, problemID string, status problemdomain.Status) error {
	const q = `UPDATE problems SET status = $2 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, problemID, string(status))
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return problemdomain.ErrProblemNotFound
	}
	return nil
}

// Update persists a reporter's edit. The editable window lives in the WHERE
// clause (status = 'Reported' AND valid_count = 0) so it is enforced atomically:
// a validation vote landing between the caller's load and this write flips the
// status or bumps the count, the update matches no rows, and the reporter is told
// ErrNotEditable rather than overwriting a report others have already endorsed.
// area_id and pointed_official_id are deliberately not updatable — they fix the
// jurisdiction and the accountable official at filing time.
func (r *ProblemRepository) Update(ctx context.Context, p *problemdomain.Problem) error {
	const q = `
		UPDATE problems
		SET title = $2, description = $3, address = NULLIF($4, ''), lat = $5, lng = $6,
		    proposed_solution = NULLIF($7, '')
		WHERE id = $1 AND status = 'Reported' AND valid_count = 0`
	ct, err := r.pool.Exec(ctx, q,
		p.ID, p.Title, p.Description, p.Location.Address, p.Location.Lat, p.Location.Lng, p.ProposedSolution)
	if err != nil {
		return fmt.Errorf("problem update: %w", err)
	}
	if ct.RowsAffected() == 0 {
		// The caller loaded the problem first, so it exists: no rows means it left
		// the editable window (a vote landed, or its status moved) under us.
		return problemdomain.ErrNotEditable
	}
	return nil
}

// SetWithdrawn records a reporter's retraction. The `status = $2` guard is a
// compare-and-set on the source status, mirroring SetScreening: a problem assigned
// between the load and this write keeps its new status and the caller gets
// ErrNotWithdrawable rather than withdrawing an in-flight case out from under an
// official.
func (r *ProblemRepository) SetWithdrawn(ctx context.Context, problemID string, from problemdomain.Status) error {
	const q = `UPDATE problems SET status = $3 WHERE id = $1 AND status = $2`
	ct, err := r.pool.Exec(ctx, q, problemID, string(from), string(problemdomain.StatusWithdrawn))
	if err != nil {
		return fmt.Errorf("set withdrawn: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return problemdomain.ErrNotWithdrawable
	}
	return nil
}

// SetScreening records a moderation decision. The `status = $2` guard makes the
// transition conditional in the same statement that performs it, so two admins
// acting on the same problem cannot both succeed — the loser gets ErrNotRejectable
// rather than silently overwriting the winner's decision. It also closes the race
// with assignment: a problem assigned between the load and this write keeps its
// new status instead of being taken down out from under the official.
func (r *ProblemRepository) SetScreening(ctx context.Context, problemID string, from, to problemdomain.Status, reason problemdomain.RejectionReason, reviewedBy string, reviewedAt time.Time) error {
	const q = `
		UPDATE problems
		SET status = $3, rejection_reason = NULLIF($4, ''), reviewed_by = NULLIF($5, ''), reviewed_at = $6
		WHERE id = $1 AND status = $2`
	ct, err := r.pool.Exec(ctx, q, problemID, string(from), string(to), string(reason), reviewedBy, reviewedAt)
	if err != nil {
		return fmt.Errorf("set screening: %w", err)
	}
	if ct.RowsAffected() == 0 {
		// The caller loaded the problem first, so it exists: no rows means its
		// status moved under us.
		return problemdomain.ErrNotRejectable
	}
	return nil
}

// Delete permanently removes the problem row, and with it — by ON DELETE CASCADE
// declared in the migrations — its validation votes, suggestions, assignment,
// admin votes, case (and the case's plan, updates, blockers and evidence) and
// confirmations. The cascade is the database's, not this statement's: nothing here
// enumerates the dependents, so a table added later with a CASCADE FK joins the
// blast radius silently. Check the FK when you add one.
//
// Unlike SetWithdrawn and SetScreening there is no `status = $2` compare-and-set,
// because there is no window to guard: every status is deletable (CLAUDE.md A.3.3).
// No rows affected therefore means the problem is genuinely gone, not that it moved.
func (r *ProblemRepository) Delete(ctx context.Context, problemID string) error {
	const q = `DELETE FROM problems WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, problemID)
	if err != nil {
		return fmt.Errorf("delete problem: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return problemdomain.ErrProblemNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProblem(row rowScanner) (*problemdomain.Problem, error) {
	var p problemdomain.Problem
	var areaID, address, status, rejectionReason string
	var lat, lng *float64
	if err := row.Scan(
		&p.ID, &p.Title, &p.Description, &areaID, &address, &lat, &lng,
		&p.ReporterID, &p.PointedOfficialID, &p.ProposedSolution, &p.ImageURL,
		&status, &p.ValidCount, &p.CreatedAt,
		&rejectionReason, &p.ReviewedBy, &p.ReviewedAt,
	); err != nil {
		return nil, err
	}
	p.Location = valueobject.Location{
		AreaID:  valueobject.AreaID(areaID),
		Address: address,
		Lat:     lat,
		Lng:     lng,
	}
	p.Status = problemdomain.Status(status)
	p.RejectionReason = problemdomain.RejectionReason(rejectionReason)
	return &p, nil
}
