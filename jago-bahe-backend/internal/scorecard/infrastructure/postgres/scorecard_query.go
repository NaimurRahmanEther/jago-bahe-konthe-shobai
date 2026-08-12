// Package postgres implements the scorecard Query as read-optimized SQL. As a
// read model it draws directly from the tables the other contexts own (cases,
// blockers, officials, problems) and never mutates them; the fairness rule is
// applied in the domain, so this layer only projects the raw per-case facts.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	problemdomain "jago-bahe-backend/internal/problem/domain"
	"jago-bahe-backend/internal/scorecard/domain"
)

// ScorecardQuery is the pgx-backed read model.
type ScorecardQuery struct {
	pool *pgxpool.Pool
}

// NewScorecardQuery constructs the query.
func NewScorecardQuery(pool *pgxpool.Pool) *ScorecardQuery {
	return &ScorecardQuery{pool: pool}
}

var _ domain.Query = (*ScorecardQuery)(nil)

// caseFactSelect projects one fairness fact per case. The active blocker is the
// latest unresolved one (matching the resolution aggregate's ActiveBlocker),
// joined via LATERAL so a case with no open blocker still yields a row with an
// empty adjudication. ResponseDays is the first-response latency (acknowledged −
// created) in days, 0 when not yet acknowledged.
const caseFactSelect = `
	SELECT
		c.status,
		COALESCE(b.adjudication, '') AS active_blocker_adjudication,
		(c.acknowledged_at IS NOT NULL) AS acknowledged,
		COALESCE(EXTRACT(EPOCH FROM (c.acknowledged_at - c.created_at)) / 86400.0, 0) AS response_days
	FROM cases c
	LEFT JOIN LATERAL (
		SELECT adjudication
		FROM obstacles
		WHERE case_id = c.id AND resolved_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	) b ON true`

// OfficialExists reports whether the official id is in the directory.
func (q *ScorecardQuery) OfficialExists(ctx context.Context, officialID string) (bool, error) {
	var exists bool
	err := q.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM officials WHERE id = $1)`, officialID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("official exists: %w", err)
	}
	return exists, nil
}

// CaseFactsByOfficial returns the fairness facts for one official's cases.
func (q *ScorecardQuery) CaseFactsByOfficial(ctx context.Context, officialID string) ([]domain.CaseFact, error) {
	return q.queryFacts(ctx, caseFactSelect+` WHERE c.official_id = $1`, officialID)
}

// SeatCaseFacts returns the fairness facts for every case in the seat.
func (q *ScorecardQuery) SeatCaseFacts(ctx context.Context) ([]domain.CaseFact, error) {
	return q.queryFacts(ctx, caseFactSelect)
}

// RecordByOfficial projects the official's public case record: each case they
// hold, the plan they published, and the snapshotted suggestion that plan answers.
//
// Two joins carry the design's guarantees. The problems join is filtered to
// publicly visible statuses, so a report still awaiting screening never surfaces
// through an official's page — a back door the feed itself closes. And the
// answered suggestion is read from the plan's own snapshot columns rather than
// re-ranked from suggestion_votes: the ranking rule has exactly one authority
// (suggestion/domain.Service), and re-deriving "top" here would let this page name
// a different question than the problem page the community actually voted on.
func (q *ScorecardQuery) RecordByOfficial(ctx context.Context, officialID string) ([]domain.CaseRecord, error) {
	const sql = `
		SELECT
			c.id, c.problem_id, p.title, c.status, c.created_at, c.deadline,
			(c.acknowledged_at IS NOT NULL), c.acknowledged_at,
			(pl.id IS NOT NULL),
			COALESCE(pl.strategy, ''), COALESCE(pl.timeline_weeks, 0), COALESCE(pl.obstacles, ''),
			COALESCE(pl.suggestion_response, ''), COALESCE(pl.answered_suggestion_text, ''), pl.created_at,
			COALESCE(b.adjudication, '') AS active_blocker_adjudication
		FROM cases c
		JOIN problems p ON p.id = c.problem_id AND p.status = ANY($2::text[])
		LEFT JOIN plans pl ON pl.case_id = c.id
		LEFT JOIN LATERAL (
			SELECT adjudication
			FROM obstacles
			WHERE case_id = c.id AND resolved_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		) b ON true
		WHERE c.official_id = $1
		ORDER BY c.created_at DESC`

	rows, err := q.pool.Query(ctx, sql, officialID, publicProblemStatuses())
	if err != nil {
		return nil, fmt.Errorf("record by official: %w", err)
	}
	defer rows.Close()

	var out []domain.CaseRecord
	for rows.Next() {
		var r domain.CaseRecord
		var adjudication string
		if err := rows.Scan(
			&r.CaseID, &r.ProblemID, &r.Title, &r.Status, &r.CreatedAt, &r.Deadline,
			&r.Acknowledged, &r.AcknowledgedAt,
			&r.HasPlan,
			&r.Strategy, &r.TimelineWeeks, &r.Obstacles,
			&r.SuggestionResponse, &r.AnsweredSuggestion, &r.PlanCreatedAt,
			&adjudication,
		); err != nil {
			return nil, fmt.Errorf("scan case record: %w", err)
		}
		r.BlockedOnHigherAuthority = domain.FairnessFlag(r.Status, domain.Adjudication(adjudication))
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate case records: %w", err)
	}
	return out, nil
}

// queryFacts runs a caseFactSelect variant and scans the rows into CaseFacts.
func (q *ScorecardQuery) queryFacts(ctx context.Context, sql string, args ...any) ([]domain.CaseFact, error) {
	rows, err := q.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query case facts: %w", err)
	}
	defer rows.Close()

	var facts []domain.CaseFact
	for rows.Next() {
		var f domain.CaseFact
		var adjudication string
		if err := rows.Scan(&f.Status, &adjudication, &f.Acknowledged, &f.ResponseDays); err != nil {
			return nil, fmt.Errorf("scan case fact: %w", err)
		}
		f.ActiveBlocker = domain.Adjudication(adjudication)
		facts = append(facts, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate case facts: %w", err)
	}
	return facts, nil
}

// publicProblemStatuses returns the statuses a problem must hold to be counted or
// shown publicly, as SQL text-array arguments.
//
// This imports the problem domain's PublicStatuses() rather than repeating the
// list locally, deliberately breaking this package's usual "local literals, couple
// to no other domain" convention (see scorecard/domain/fairness.go). The reason is
// the direction each mistake fails in: a stale copy of a *case status* string
// mis-buckets a number, but a stale copy of *this* list silently publishes hidden
// reports. A leak must not be one forgotten literal away, so there is exactly one
// authority for what is public and every reader defers to it.
func publicProblemStatuses() []string {
	statuses := problemdomain.PublicStatuses()
	out := make([]string, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, string(s))
	}
	return out
}

// SeatProblemCount returns the number of publicly visible problems in the seat.
//
// The status filter is load-bearing, not decorative. Before B9 every problem was
// public and an unfiltered COUNT(*) was correct; now a problem awaiting screening
// is hidden from everyone but its reporter and its union's admin, so counting it
// here would publish exactly the fact the screening gate withholds — how many
// unscreened reports exist — to any anonymous caller.
func (q *ScorecardQuery) SeatProblemCount(ctx context.Context) (int, error) {
	var n int
	const sql = `SELECT COUNT(*) FROM problems WHERE status = ANY($1::text[])`
	if err := q.pool.QueryRow(ctx, sql, publicProblemStatuses()).Scan(&n); err != nil {
		return 0, fmt.Errorf("seat problem count: %w", err)
	}
	return n, nil
}
