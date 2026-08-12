// Package postgres implements the resolution CaseRepository via pgx. It loads and
// saves the whole Case aggregate (plan, updates, evidence, obstacles with their
// obstacle votes and unblocking plans), deriving advisory tallies and upvote
// counts on read so nothing denormalized can drift.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"jago-bahe-backend/internal/resolution/domain"
)

// CaseRepository is the pgx-backed implementation of domain.Repository.
type CaseRepository struct {
	pool *pgxpool.Pool
}

// NewCaseRepository constructs the repository.
func NewCaseRepository(pool *pgxpool.Pool) *CaseRepository {
	return &CaseRepository{pool: pool}
}

var _ domain.Repository = (*CaseRepository)(nil)

// Save upserts the case row and inserts any newly added child entities. Child
// inserts are idempotent (ON CONFLICT DO NOTHING), so re-saving the aggregate
// after appending one update/evidence/blocker persists only the new rows.
func (r *CaseRepository) Save(ctx context.Context, c *domain.Case) error {
	const caseQ = `
		INSERT INTO cases (id, problem_id, official_id, monitor_official_id, status, acknowledged_at, deadline, dispute_reason, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, NULLIF($8, ''), $9)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			acknowledged_at = EXCLUDED.acknowledged_at,
			dispute_reason = EXCLUDED.dispute_reason`
	if _, err := r.pool.Exec(ctx, caseQ,
		c.ID, c.ProblemID, c.OfficialID, c.MonitorOfficialID, string(c.Status),
		c.AcknowledgedAt, c.Deadline, c.DisputeReason, c.CreatedAt); err != nil {
		return fmt.Errorf("save case: %w", err)
	}

	if c.Plan != nil {
		const planQ = `
			INSERT INTO plans (id, case_id, strategy, timeline_weeks, obstacles, suggestion_response,
			                   answered_suggestion_id, answered_suggestion_text, created_at)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), $9)
			ON CONFLICT (case_id) DO NOTHING`
		if _, err := r.pool.Exec(ctx, planQ,
			c.Plan.ID, c.ID, c.Plan.Strategy, c.Plan.TimelineWeeks, c.Plan.Obstacles, c.Plan.SuggestionResponse,
			c.Plan.AnsweredSuggestionID, c.Plan.AnsweredSuggestionTxt, c.Plan.CreatedAt); err != nil {
			return fmt.Errorf("save plan: %w", err)
		}

		// Weekly tasks are the one child whose upsert is DO UPDATE, not DO NOTHING:
		// task/week are written once with the plan, but completed/completed_at change
		// as the official checks weeks off, and re-saving the aggregate must persist
		// that. Only those two columns are touched — the text and week are immutable.
		for _, tk := range c.Plan.Tasks {
			const taskQ = `
				INSERT INTO plan_tasks (id, plan_id, week_number, task, completed, completed_at, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (id) DO UPDATE SET
					completed = EXCLUDED.completed,
					completed_at = EXCLUDED.completed_at`
			if _, err := r.pool.Exec(ctx, taskQ,
				tk.ID, c.Plan.ID, tk.WeekNumber, tk.Task, tk.Completed, tk.CompletedAt, tk.CreatedAt); err != nil {
				return fmt.Errorf("save plan task: %w", err)
			}
		}
	}

	for _, u := range c.Updates {
		const q = `INSERT INTO progress_updates (id, case_id, kind, text, created_at)
			VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING`
		if _, err := r.pool.Exec(ctx, q, u.ID, c.ID, string(u.Kind), u.Text, u.CreatedAt); err != nil {
			return fmt.Errorf("save update: %w", err)
		}
	}
	for _, e := range c.Evidence {
		const q = `INSERT INTO evidence (id, case_id, before_image_url, after_image_url, created_at)
			VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING`
		if _, err := r.pool.Exec(ctx, q, e.ID, c.ID, e.BeforeImageURL, e.AfterImageURL, e.CreatedAt); err != nil {
			return fmt.Errorf("save evidence: %w", err)
		}
	}
	for _, b := range c.Obstacles {
		const q = `INSERT INTO obstacles (id, case_id, category, what_blocks, who_unblocks, proof_tried, created_at, resolved_at)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8) ON CONFLICT (id) DO NOTHING`
		if _, err := r.pool.Exec(ctx, q, b.ID, c.ID, string(b.Category), b.WhatBlocks, b.WhoUnblocks, b.ProofTried, b.CreatedAt, b.ResolvedAt); err != nil {
			return fmt.Errorf("save obstacle: %w", err)
		}
	}
	return nil
}

// ReplacePlan swaps a case's single plan for a new one (the revise/restart path).
// Save's plan upsert is ON CONFLICT (case_id) DO NOTHING — a plan is written once
// and only its tasks' completion changes afterwards — so it can never replace a
// plan. Replacing needs an explicit delete of the old plan and its tasks before
// the new one is inserted; the caller persists the resumed status and the "Plan
// revised" note through Save (whose plan block then no-ops on the existing row).
func (r *CaseRepository) ReplacePlan(ctx context.Context, c *domain.Case) error {
	if c.Plan == nil {
		return nil
	}
	// Delete tasks first (they reference the plan), then the plan row itself. A case
	// has exactly one plan, keyed by case_id.
	if _, err := r.pool.Exec(ctx,
		`DELETE FROM plan_tasks WHERE plan_id IN (SELECT id FROM plans WHERE case_id = $1)`, c.ID); err != nil {
		return fmt.Errorf("replace plan (delete tasks): %w", err)
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM plans WHERE case_id = $1`, c.ID); err != nil {
		return fmt.Errorf("replace plan (delete plan): %w", err)
	}
	const planQ = `
		INSERT INTO plans (id, case_id, strategy, timeline_weeks, obstacles, suggestion_response,
		                   answered_suggestion_id, answered_suggestion_text, created_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), $9)`
	if _, err := r.pool.Exec(ctx, planQ,
		c.Plan.ID, c.ID, c.Plan.Strategy, c.Plan.TimelineWeeks, c.Plan.Obstacles, c.Plan.SuggestionResponse,
		c.Plan.AnsweredSuggestionID, c.Plan.AnsweredSuggestionTxt, c.Plan.CreatedAt); err != nil {
		return fmt.Errorf("replace plan (insert plan): %w", err)
	}
	for _, tk := range c.Plan.Tasks {
		const taskQ = `
			INSERT INTO plan_tasks (id, plan_id, week_number, task, completed, completed_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`
		if _, err := r.pool.Exec(ctx, taskQ,
			tk.ID, c.Plan.ID, tk.WeekNumber, tk.Task, tk.Completed, tk.CompletedAt, tk.CreatedAt); err != nil {
			return fmt.Errorf("replace plan (insert task): %w", err)
		}
	}
	return nil
}

const caseColumns = `id, problem_id, official_id, COALESCE(monitor_official_id, ''), status,
	acknowledged_at, deadline, COALESCE(dispute_reason, ''), escalation_level, last_escalated_at, created_at`

func (r *CaseRepository) GetByID(ctx context.Context, caseID string) (*domain.Case, error) {
	return r.getOne(ctx, `WHERE id = $1`, caseID)
}

func (r *CaseRepository) GetByProblem(ctx context.Context, problemID string) (*domain.Case, error) {
	return r.getOne(ctx, `WHERE problem_id = $1`, problemID)
}

func (r *CaseRepository) getOne(ctx context.Context, where, arg string) (*domain.Case, error) {
	q := `SELECT ` + caseColumns + ` FROM cases ` + where
	c, err := scanCase(r.pool.QueryRow(ctx, q, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCaseNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadChildren(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (r *CaseRepository) ListByOfficial(ctx context.Context, officialID string) ([]domain.Case, error) {
	q := `SELECT ` + caseColumns + ` FROM cases WHERE official_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, officialID)
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
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

type rowScanner interface{ Scan(dest ...any) error }

func scanCase(row rowScanner) (*domain.Case, error) {
	var c domain.Case
	var status string
	if err := row.Scan(&c.ID, &c.ProblemID, &c.OfficialID, &c.MonitorOfficialID, &status,
		&c.AcknowledgedAt, &c.Deadline, &c.DisputeReason, &c.EscalationLevel, &c.LastEscalatedAt, &c.CreatedAt); err != nil {
		return nil, err
	}
	c.Status = domain.Status(status)
	return &c, nil
}

// loadChildren populates the aggregate's plan, updates, evidence, and obstacles
// (each blocker with its advisory tallies and ranked-on-read unblocking plans).
func (r *CaseRepository) loadChildren(ctx context.Context, c *domain.Case) error {
	if err := r.loadPlan(ctx, c); err != nil {
		return err
	}
	if err := r.loadUpdates(ctx, c); err != nil {
		return err
	}
	if err := r.loadEvidence(ctx, c); err != nil {
		return err
	}
	return r.loadObstacles(ctx, c)
}

func (r *CaseRepository) loadPlan(ctx context.Context, c *domain.Case) error {
	const q = `SELECT id, case_id, strategy, timeline_weeks, COALESCE(obstacles, ''), suggestion_response,
		COALESCE(answered_suggestion_id, ''), COALESCE(answered_suggestion_text, ''), created_at
		FROM plans WHERE case_id = $1`
	var p domain.Plan
	err := r.pool.QueryRow(ctx, q, c.ID).Scan(&p.ID, &p.CaseID, &p.Strategy, &p.TimelineWeeks, &p.Obstacles, &p.SuggestionResponse,
		&p.AnsweredSuggestionID, &p.AnsweredSuggestionTxt, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	if err := r.loadPlanTasks(ctx, &p); err != nil {
		return err
	}
	c.Plan = &p
	return nil
}

func (r *CaseRepository) loadPlanTasks(ctx context.Context, p *domain.Plan) error {
	const q = `SELECT id, plan_id, week_number, task, completed, completed_at, created_at
		FROM plan_tasks WHERE plan_id = $1 ORDER BY week_number`
	rows, err := r.pool.Query(ctx, q, p.ID)
	if err != nil {
		return fmt.Errorf("load plan tasks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tk domain.PlanTask
		if err := rows.Scan(&tk.ID, &tk.PlanID, &tk.WeekNumber, &tk.Task, &tk.Completed, &tk.CompletedAt, &tk.CreatedAt); err != nil {
			return err
		}
		p.Tasks = append(p.Tasks, tk)
	}
	return rows.Err()
}

func (r *CaseRepository) loadUpdates(ctx context.Context, c *domain.Case) error {
	const q = `SELECT id, case_id, kind, text, created_at FROM progress_updates WHERE case_id = $1 ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, c.ID)
	if err != nil {
		return fmt.Errorf("load updates: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var u domain.ProgressUpdate
		var kind string
		if err := rows.Scan(&u.ID, &u.CaseID, &kind, &u.Text, &u.CreatedAt); err != nil {
			return err
		}
		u.Kind = domain.UpdateKind(kind)
		c.Updates = append(c.Updates, u)
	}
	return rows.Err()
}

func (r *CaseRepository) loadEvidence(ctx context.Context, c *domain.Case) error {
	const q = `SELECT id, case_id, before_image_url, after_image_url, created_at FROM evidence WHERE case_id = $1 ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, c.ID)
	if err != nil {
		return fmt.Errorf("load evidence: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var e domain.Evidence
		if err := rows.Scan(&e.ID, &e.CaseID, &e.BeforeImageURL, &e.AfterImageURL, &e.CreatedAt); err != nil {
			return err
		}
		c.Evidence = append(c.Evidence, e)
	}
	return rows.Err()
}

func (r *CaseRepository) loadObstacles(ctx context.Context, c *domain.Case) error {
	const q = `
		SELECT b.id, b.case_id, b.category, b.what_blocks, b.who_unblocks, COALESCE(b.proof_tried, ''),
			b.adjudication, b.adjudicated_at, b.created_at, b.resolved_at,
			(SELECT count(*) FROM obstacle_votes ov WHERE ov.obstacle_id = b.id AND ov.choice = 'real'),
			(SELECT count(*) FROM obstacle_votes ov WHERE ov.obstacle_id = b.id AND ov.choice = 'not_convinced')
		FROM obstacles b WHERE b.case_id = $1 ORDER BY b.created_at`
	rows, err := r.pool.Query(ctx, q, c.ID)
	if err != nil {
		return fmt.Errorf("load obstacles: %w", err)
	}
	defer rows.Close()
	var obstacles []domain.Obstacle
	for rows.Next() {
		var b domain.Obstacle
		var category, adjudication string
		if err := rows.Scan(&b.ID, &b.CaseID, &category, &b.WhatBlocks, &b.WhoUnblocks, &b.ProofTried,
			&adjudication, &b.AdjudicatedAt, &b.CreatedAt, &b.ResolvedAt, &b.RealCount, &b.NotConvincedCount); err != nil {
			return err
		}
		b.Category = domain.ObstacleCategory(category)
		b.Adjudication = domain.Adjudication(adjudication)
		obstacles = append(obstacles, b)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range obstacles {
		plans, err := r.listUnblockingPlans(ctx, obstacles[i].ID)
		if err != nil {
			return err
		}
		obstacles[i].UnblockingPlans = plans
	}
	c.Obstacles = obstacles
	return nil
}

func (r *CaseRepository) listUnblockingPlans(ctx context.Context, obstacleID string) ([]domain.UnblockingPlan, error) {
	const q = `
		SELECT p.id, p.obstacle_id, p.author_id, p.text, p.created_at,
			(SELECT count(*) FROM unblocking_plan_votes v WHERE v.plan_id = p.id)
		FROM unblocking_plans p WHERE p.obstacle_id = $1 ORDER BY p.created_at`
	rows, err := r.pool.Query(ctx, q, obstacleID)
	if err != nil {
		return nil, fmt.Errorf("list unblocking plans: %w", err)
	}
	defer rows.Close()
	var out []domain.UnblockingPlan
	for rows.Next() {
		var p domain.UnblockingPlan
		if err := rows.Scan(&p.ID, &p.ObstacleID, &p.AuthorID, &p.Text, &p.CreatedAt, &p.UpvoteCount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *CaseRepository) AddObstacleVote(ctx context.Context, v domain.ObstacleVote) error {
	const q = `INSERT INTO obstacle_votes (id, obstacle_id, voter_id, choice, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, q, v.ID, v.ObstacleID, v.VoterID, string(v.Choice), v.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique(obstacle_id, voter_id)
			return domain.ErrAlreadyVotedObstacle
		}
		return fmt.Errorf("add obstacle vote: %w", err)
	}
	return nil
}

func (r *CaseRepository) AddUnblockingPlan(ctx context.Context, p *domain.UnblockingPlan) error {
	const q = `INSERT INTO unblocking_plans (id, obstacle_id, author_id, text, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	if _, err := r.pool.Exec(ctx, q, p.ID, p.ObstacleID, p.AuthorID, p.Text, p.CreatedAt); err != nil {
		return fmt.Errorf("add unblocking plan: %w", err)
	}
	return nil
}

func (r *CaseRepository) GetUnblockingPlan(ctx context.Context, planID string) (*domain.UnblockingPlan, error) {
	const q = `
		SELECT p.id, p.obstacle_id, p.author_id, p.text, p.created_at,
			(SELECT count(*) FROM unblocking_plan_votes v WHERE v.plan_id = p.id)
		FROM unblocking_plans p WHERE p.id = $1`
	var p domain.UnblockingPlan
	err := r.pool.QueryRow(ctx, q, planID).Scan(&p.ID, &p.ObstacleID, &p.AuthorID, &p.Text, &p.CreatedAt, &p.UpvoteCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get unblocking plan: %w", err)
	}
	return &p, nil
}

func (r *CaseRepository) ProblemForPlan(ctx context.Context, planID string) (string, error) {
	const q = `
		SELECT c.problem_id
		FROM unblocking_plans p
		JOIN obstacles b ON p.obstacle_id = b.id
		JOIN cases c ON b.case_id = c.id
		WHERE p.id = $1`
	var problemID string
	err := r.pool.QueryRow(ctx, q, planID).Scan(&problemID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrPlanNotFound
	}
	if err != nil {
		return "", fmt.Errorf("problem for plan: %w", err)
	}
	return problemID, nil
}

// ToggleUnblockingUpvote adds the voter's upvote, or removes it if already
// present (the toggle), then returns the plan with its refreshed count.
func (r *CaseRepository) ToggleUnblockingUpvote(ctx context.Context, planID, voterID string) (*domain.UnblockingPlan, error) {
	if _, err := r.GetUnblockingPlan(ctx, planID); err != nil {
		return nil, err // domain.ErrPlanNotFound
	}
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM unblocking_plan_votes WHERE plan_id = $1 AND voter_id = $2)`,
		planID, voterID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check upvote: %w", err)
	}
	if exists {
		if _, err := r.pool.Exec(ctx, `DELETE FROM unblocking_plan_votes WHERE plan_id = $1 AND voter_id = $2`, planID, voterID); err != nil {
			return nil, fmt.Errorf("remove upvote: %w", err)
		}
	} else {
		if _, err := r.pool.Exec(ctx,
			`INSERT INTO unblocking_plan_votes (id, plan_id, voter_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			idFor(planID, voterID), planID, voterID); err != nil {
			return nil, fmt.Errorf("add upvote: %w", err)
		}
	}
	return r.GetUnblockingPlan(ctx, planID)
}

// idFor builds a deterministic id for an upvote row (a plan/voter pair is unique).
func idFor(planID, voterID string) string { return "ubpv-" + planID + "-" + voterID }

// --- B6: confirmation, adjudication, escalation ---

func (r *CaseRepository) AddConfirmation(ctx context.Context, c domain.Confirmation) error {
	const q = `INSERT INTO confirmations (id, problem_id, resident_id, outcome, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	if _, err := r.pool.Exec(ctx, q, c.ID, c.ProblemID, c.ResidentID, string(c.Outcome), c.CreatedAt); err != nil {
		return fmt.Errorf("add confirmation: %w", err)
	}
	return nil
}

func (r *CaseRepository) GetCaseByObstacle(ctx context.Context, obstacleID string) (*domain.Case, error) {
	var caseID string
	err := r.pool.QueryRow(ctx, `SELECT case_id FROM obstacles WHERE id = $1`, obstacleID).Scan(&caseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrObstacleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("case by blocker: %w", err)
	}
	return r.GetByID(ctx, caseID)
}

func (r *CaseRepository) UpdateObstacleAdjudication(ctx context.Context, obstacleID string, verdict domain.Adjudication, adjudicatedAt time.Time, resolvedAt *time.Time) error {
	const q = `UPDATE obstacles SET adjudication = $2, adjudicated_at = $3, resolved_at = $4 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, obstacleID, string(verdict), adjudicatedAt, resolvedAt)
	if err != nil {
		return fmt.Errorf("update adjudication: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrObstacleNotFound
	}
	return nil
}

func (r *CaseRepository) ListEscalationCandidates(ctx context.Context) ([]domain.Case, error) {
	q := `SELECT ` + caseColumns + ` FROM cases
		WHERE status IN ('Assigned', 'Acknowledged', 'Planned', 'InProgress', 'Reopened', 'Blocked')`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list escalation candidates: %w", err)
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
	// Only Blocked cases need their obstacles loaded (for the review-window anchor).
	for i := range cases {
		if cases[i].Status == domain.StatusBlocked {
			if err := r.loadObstacles(ctx, &cases[i]); err != nil {
				return nil, err
			}
		}
	}
	return cases, nil
}

func (r *CaseRepository) SetEscalation(ctx context.Context, caseID string, level int, at time.Time) error {
	const q = `UPDATE cases SET escalation_level = $2, last_escalated_at = $3 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, caseID, level, at)
	if err != nil {
		return fmt.Errorf("set escalation: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrCaseNotFound
	}
	return nil
}
