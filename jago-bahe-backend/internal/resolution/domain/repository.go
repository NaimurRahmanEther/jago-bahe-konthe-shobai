package domain

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors mapped to HTTP status codes in one place by the http layer.
var (
	ErrCaseNotFound       = errors.New("case not found")
	ErrNotCaseOwner       = errors.New("this case is assigned to another official")
	ErrTaskNotFound       = errors.New("no such weekly task on this plan")
	ErrObstacleNotFound   = errors.New("no active obstacle to judge")
	ErrPlanNotFound       = errors.New("unblocking plan not found")
	ErrProblemNotFound    = errors.New("problem not found")
	ErrAssignmentMissing  = errors.New("problem has no assignment to open a case from")
	ErrEmptyText          = errors.New("text is required")
	ErrInvalidKind        = errors.New("invalid update kind")
	ErrInvalidCategory    = errors.New("invalid obstacle category")
	ErrInvalidChoice      = errors.New("invalid vote choice")
	ErrNotVerified        = errors.New("resident is not verified")
	ErrNotAreaResident    = errors.New("resident is not in this problem's area")
	ErrAreaNotFound       = errors.New("area not found")
	ErrNotReporter        = errors.New("only the reporting resident can confirm")
	ErrInvalidOutcome     = errors.New("invalid confirmation outcome")
	ErrInvalidVerdict     = errors.New("invalid adjudication verdict")
	ErrInvalidDecision    = errors.New("invalid acknowledgement decision")
	ErrAlreadyAdjudicated = errors.New("this obstacle has already been adjudicated")

	ErrObservationNotFound = errors.New("observation not found")
	ErrNotObserver         = errors.New("this observation belongs to another official")
	ErrObservationResolved = errors.New("this observation is already resolved")
)

// Repository is the port for case persistence. It loads and saves the whole Case
// aggregate — plan, updates, evidence, and obstacles (with their obstacle votes
// and unblocking plans) — as one unit (golden rule A.4.3).
type Repository interface {
	// Save upserts the whole aggregate (the case row plus any newly added child
	// entities). Implementations derive advisory tallies and upvote counts on read.
	//
	// Save's plan upsert is write-once (a plan is created and then only its tasks'
	// completion changes), so it cannot REPLACE a plan — see ReplacePlan.
	Save(ctx context.Context, c *Case) error
	// ReplacePlan swaps a case's single plan for a new one — the revise/restart
	// path (a stalled case gets a fresh plan). It removes the old plan and its
	// weekly tasks and inserts c.Plan and its tasks; the case status and the
	// accompanying timeline note are persisted by the caller via Save. It is
	// separate because Save deliberately never overwrites an existing plan.
	ReplacePlan(ctx context.Context, c *Case) error
	GetByID(ctx context.Context, caseID string) (*Case, error)
	// GetByProblem returns the case for a problem, or ErrCaseNotFound if none has
	// been materialized yet.
	GetByProblem(ctx context.Context, problemID string) (*Case, error)
	// ListByOfficial returns every case currently assigned to the official.
	ListByOfficial(ctx context.Context, officialID string) ([]Case, error)

	// AddObstacleVote records one resident's advisory vote on a blocker; returns
	// ErrAlreadyVotedObstacle if the resident already voted on it.
	AddObstacleVote(ctx context.Context, v ObstacleVote) error
	// AddUnblockingPlan persists a community unblocking plan for a blocker.
	AddUnblockingPlan(ctx context.Context, p *UnblockingPlan) error
	// GetUnblockingPlan loads a single unblocking plan (for upvote/eligibility).
	GetUnblockingPlan(ctx context.Context, planID string) (*UnblockingPlan, error)
	// ProblemForPlan resolves the problem an unblocking plan belongs to (plan →
	// blocker → case → problem), so the area-residency guard can be applied.
	ProblemForPlan(ctx context.Context, planID string) (string, error)
	// ToggleUnblockingUpvote adds or removes a resident's upvote on a plan and
	// returns the plan with its refreshed count.
	ToggleUnblockingUpvote(ctx context.Context, planID, voterID string) (*UnblockingPlan, error)

	// --- B6: confirmation, adjudication, escalation ---

	// AddConfirmation records a reporting resident's verdict on a Done case.
	AddConfirmation(ctx context.Context, c Confirmation) error
	// GetCaseByObstacle loads the whole case that owns a blocker (for adjudication),
	// or ErrObstacleNotFound.
	GetCaseByObstacle(ctx context.Context, obstacleID string) (*Case, error)
	// UpdateObstacleAdjudication records the authority's verdict on a blocker and,
	// on a deny, its resolved-at (the blocker closes and the case resumes).
	UpdateObstacleAdjudication(ctx context.Context, obstacleID string, verdict Adjudication, adjudicatedAt time.Time, resolvedAt *time.Time) error
	// ListEscalationCandidates returns every non-terminal case (loaded with its
	// obstacles) for the deterministic overdue scan.
	ListEscalationCandidates(ctx context.Context) ([]Case, error)
	// SetEscalation records a case's new visibility level (the work is not moved).
	SetEscalation(ctx context.Context, caseID string, level int, at time.Time) error

	// --- B19: the observation ladder (the read side of escalation) ---

	// ListByMonitor returns the unfinished cases the official is the DIRECT monitor
	// of. Monitoring is continuous from assignment, not only once something goes
	// wrong, so this is deliberately not filtered by escalation (Concept §7).
	ListByMonitor(ctx context.Context, officialID string) ([]Case, error)
	// CasesByIDs loads whole cases by id, for the rows a HIGHER rung sees only
	// because an observation was opened against them — it is not their direct
	// monitor, so ListByMonitor would not return them.
	CasesByIDs(ctx context.Context, caseIDs []string) ([]Case, error)
	// ListObservationsByObserver returns every observation — open and resolved,
	// with its notes — opened against the official. It is how a HIGHER rung sees a
	// case at all: only the direct monitor gets continuous visibility.
	ListObservationsByObserver(ctx context.Context, officialID string) ([]Observation, error)
	// ListObservationsByCases loads the observations of many cases in ONE query,
	// the page-decoration shape (A.3.4) — never one query per row.
	ListObservationsByCases(ctx context.Context, caseIDs []string) ([]Observation, error)
	// OpenObservation inserts a rung, reporting whether it actually inserted. The
	// partial unique index on (case_id, level) WHERE resolved_at IS NULL makes a
	// duplicate a no-op, and the bool is what keeps the audit entry firing exactly
	// once no matter how often the scan runs.
	OpenObservation(ctx context.Context, o *Observation) (bool, error)
	// ResolveOpenObservations closes every open observation on a case at the given
	// instant, returning how many closed. Resolved rows are kept as history.
	ResolveOpenObservations(ctx context.Context, caseID string, at time.Time) (int, error)
	// GetObservation loads one observation with its notes, or ErrObservationNotFound.
	GetObservation(ctx context.Context, observationID string) (*Observation, error)
	// AddObservationNote appends a monitor's note. Notes are never overwritten.
	AddObservationNote(ctx context.Context, n ObservationNote) error
	// LastActivityByCases returns each case's last ASSIGNEE activity in one query,
	// for the worker's scan. It states the same rule as Case.LastActivityAt in SQL
	// so the scan need not load whole aggregates every tick; a cross-implementation
	// test pins the two to the same answer.
	LastActivityByCases(ctx context.Context, caseIDs []string) (map[string]time.Time, error)
}

// ErrAlreadyVotedObstacle is returned when a resident votes twice on one blocker.
var ErrAlreadyVotedObstacle = errors.New("already voted on this obstacle")
