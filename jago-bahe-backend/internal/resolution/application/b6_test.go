package application_test

import (
	"context"
	"testing"
	"time"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// --- in-memory fakes (B6) ---

type fakeRepo struct {
	cases         map[string]*domain.Case // by case id
	confirmations []domain.Confirmation
	escalations   map[string]int // case id -> level last written

	// B19 — the observation ladder. Implemented in observation_test.go.
	observations []*domain.Observation
	notes        []domain.ObservationNote
	// lastActivity overrides the value derived from the aggregate, so a test can
	// stand in for the SQL projection the worker actually uses.
	lastActivity map[string]time.Time
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{cases: map[string]*domain.Case{}, escalations: map[string]int{}, lastActivity: map[string]time.Time{}}
}

func cloneCase(c *domain.Case) *domain.Case {
	cp := *c
	cp.Obstacles = append([]domain.Obstacle(nil), c.Obstacles...)
	// Deep-copy the plan and its tasks: task completion is mutated in place, so a
	// shared pointer would let a Save leak back into the "stored" copy and mask
	// whether persistence actually happened.
	if c.Plan != nil {
		plan := *c.Plan
		plan.Tasks = append([]domain.PlanTask(nil), c.Plan.Tasks...)
		cp.Plan = &plan
	}
	return &cp
}

func (r *fakeRepo) put(c *domain.Case) { r.cases[c.ID] = cloneCase(c) }

func (r *fakeRepo) Save(_ context.Context, c *domain.Case) error {
	r.cases[c.ID] = cloneCase(c)
	return nil
}

// ReplacePlan swaps the stored case's plan (the revise path). The naive fake
// persists the whole aggregate anyway, so this mirrors Save.
func (r *fakeRepo) ReplacePlan(_ context.Context, c *domain.Case) error {
	r.cases[c.ID] = cloneCase(c)
	return nil
}
func (r *fakeRepo) GetByID(_ context.Context, id string) (*domain.Case, error) {
	if c, ok := r.cases[id]; ok {
		return cloneCase(c), nil
	}
	return nil, domain.ErrCaseNotFound
}
func (r *fakeRepo) GetByProblem(_ context.Context, problemID string) (*domain.Case, error) {
	for _, c := range r.cases {
		if c.ProblemID == problemID {
			return cloneCase(c), nil
		}
	}
	return nil, domain.ErrCaseNotFound
}
func (r *fakeRepo) AddConfirmation(_ context.Context, c domain.Confirmation) error {
	r.confirmations = append(r.confirmations, c)
	return nil
}
func (r *fakeRepo) GetCaseByObstacle(_ context.Context, obstacleID string) (*domain.Case, error) {
	for _, c := range r.cases {
		for i := range c.Obstacles {
			if c.Obstacles[i].ID == obstacleID {
				return cloneCase(c), nil
			}
		}
	}
	return nil, domain.ErrObstacleNotFound
}
func (r *fakeRepo) UpdateObstacleAdjudication(_ context.Context, obstacleID string, verdict domain.Adjudication, at time.Time, resolvedAt *time.Time) error {
	for _, c := range r.cases {
		for i := range c.Obstacles {
			if c.Obstacles[i].ID == obstacleID {
				c.Obstacles[i].Adjudication = verdict
				c.Obstacles[i].AdjudicatedAt = &at
				c.Obstacles[i].ResolvedAt = resolvedAt
				return nil
			}
		}
	}
	return domain.ErrObstacleNotFound
}
func (r *fakeRepo) ListEscalationCandidates(_ context.Context) ([]domain.Case, error) {
	var out []domain.Case
	for _, c := range r.cases {
		out = append(out, *cloneCase(c))
	}
	return out, nil
}
func (r *fakeRepo) SetEscalation(_ context.Context, caseID string, level int, at time.Time) error {
	if c, ok := r.cases[caseID]; ok {
		c.EscalationLevel = level
		c.LastEscalatedAt = &at
	}
	r.escalations[caseID] = level
	return nil
}

// unused B5 methods (stubbed for the interface)
func (r *fakeRepo) ListByOfficial(context.Context, string) ([]domain.Case, error)   { return nil, nil }
func (r *fakeRepo) AddObstacleVote(context.Context, domain.ObstacleVote) error      { return nil }
func (r *fakeRepo) AddUnblockingPlan(context.Context, *domain.UnblockingPlan) error { return nil }
func (r *fakeRepo) GetUnblockingPlan(context.Context, string) (*domain.UnblockingPlan, error) {
	return nil, domain.ErrPlanNotFound
}
func (r *fakeRepo) ProblemForPlan(context.Context, string) (string, error) {
	return "", domain.ErrPlanNotFound
}
func (r *fakeRepo) ToggleUnblockingUpvote(context.Context, string, string) (*domain.UnblockingPlan, error) {
	return nil, domain.ErrPlanNotFound
}

type fakeProblems struct {
	reporter string
	status   string
}

func (p *fakeProblems) AreaID(context.Context, string) (string, error)     { return "union-1", nil }
func (p *fakeProblems) ReporterID(context.Context, string) (string, error) { return p.reporter, nil }
func (p *fakeProblems) SetStatus(_ context.Context, _, status string) error {
	p.status = status
	return nil
}

func (p *fakeProblems) TitlesByIDs(_ context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		out[id] = "title of " + id
	}
	return out, nil
}

type fakeAudit struct {
	actions []string
	// entries keeps the whole entry, not just its action: B19 asserts on the actor
	// and target of a monitor's note, since landing it on the PROBLEM is what puts
	// it on the public trail.
	entries []auditdomain.AuditEntry
}

func (a *fakeAudit) Append(_ context.Context, e auditdomain.AuditEntry) error {
	a.actions = append(a.actions, e.Action)
	a.entries = append(a.entries, e)
	return nil
}
func (a *fakeAudit) ListByTarget(context.Context, string, string) ([]auditdomain.AuditEntry, error) {
	return nil, nil
}

func (a *fakeAudit) ListByActions(context.Context, []string, int) ([]auditdomain.AuditEntry, error) {
	return nil, nil
}
func (a *fakeAudit) DeleteByTarget(context.Context, string, string) error { return nil }

func doneCase() *domain.Case {
	return &domain.Case{ID: "case-1", ProblemID: "prob-1", OfficialID: "off-1", Status: domain.StatusDone}
}

// --- confirm ---

func TestConfirmResolution_SolvedResolves(t *testing.T) {
	repo := newFakeRepo()
	repo.put(doneCase())
	problems := &fakeProblems{reporter: "res-1"}
	audit := &fakeAudit{}
	uc := application.NewConfirmResolution(repo, problems, audit, &fakeNotifier{})

	c, err := uc.Execute(context.Background(), "prob-1", "res-1", domain.ConfirmationSolved)
	if err != nil {
		t.Fatalf("confirm solved: %v", err)
	}
	if c.Status != domain.StatusResolved {
		t.Fatalf("status = %s, want Resolved", c.Status)
	}
	if problems.status != "Resolved" || len(repo.confirmations) != 1 {
		t.Fatalf("problem status %q, confirmations %d", problems.status, len(repo.confirmations))
	}
}

func TestConfirmResolution_NotSolvedReopens(t *testing.T) {
	repo := newFakeRepo()
	repo.put(doneCase())
	uc := application.NewConfirmResolution(repo, &fakeProblems{reporter: "res-1"}, &fakeAudit{}, &fakeNotifier{})

	c, err := uc.Execute(context.Background(), "prob-1", "res-1", domain.ConfirmationNotSolved)
	if err != nil {
		t.Fatalf("confirm not solved: %v", err)
	}
	if c.Status != domain.StatusReopened {
		t.Fatalf("status = %s, want Reopened", c.Status)
	}
}

func TestConfirmResolution_OnlyReporter(t *testing.T) {
	repo := newFakeRepo()
	repo.put(doneCase())
	uc := application.NewConfirmResolution(repo, &fakeProblems{reporter: "res-1"}, &fakeAudit{}, &fakeNotifier{})

	if _, err := uc.Execute(context.Background(), "prob-1", "someone-else", domain.ConfirmationSolved); err != domain.ErrNotReporter {
		t.Fatalf("non-reporter err = %v, want ErrNotReporter", err)
	}
}

func TestConfirmResolution_MustBeDone(t *testing.T) {
	repo := newFakeRepo()
	c := doneCase()
	c.Status = domain.StatusInProgress
	repo.put(c)
	uc := application.NewConfirmResolution(repo, &fakeProblems{reporter: "res-1"}, &fakeAudit{}, &fakeNotifier{})

	if _, err := uc.Execute(context.Background(), "prob-1", "res-1", domain.ConfirmationSolved); err != domain.ErrIllegalTransition {
		t.Fatalf("confirm on non-Done err = %v, want ErrIllegalTransition", err)
	}
}

// --- adjudicate ---

func blockedCase() *domain.Case {
	c := &domain.Case{ID: "case-1", ProblemID: "prob-1", OfficialID: "off-1", Status: domain.StatusBlocked}
	c.Obstacles = []domain.Obstacle{{ID: "blk-1", CaseID: "case-1", Category: domain.CategoryBudget, WhatBlocks: "no funds", WhoUnblocks: "engineer", Adjudication: domain.AdjudicationPending}}
	return c
}

func TestAdjudicate_ConfirmKeepsBlocked(t *testing.T) {
	repo := newFakeRepo()
	repo.put(blockedCase())
	uc := application.NewAdjudicateObstacle(repo, &fakeProblems{}, &fakeAudit{}, &fakeNotifier{})

	b, err := uc.Execute(context.Background(), "blk-1", "admin-1", true)
	if err != nil {
		t.Fatalf("adjudicate confirm: %v", err)
	}
	if b.Adjudication != domain.AdjudicationConfirmed {
		t.Fatalf("adjudication = %s, want confirmed", b.Adjudication)
	}
	if repo.cases["case-1"].Status != domain.StatusBlocked {
		t.Fatalf("confirm should keep the case Blocked, got %s", repo.cases["case-1"].Status)
	}
}

func TestAdjudicate_DenyResumes(t *testing.T) {
	repo := newFakeRepo()
	repo.put(blockedCase())
	problems := &fakeProblems{}
	uc := application.NewAdjudicateObstacle(repo, problems, &fakeAudit{}, &fakeNotifier{})

	b, err := uc.Execute(context.Background(), "blk-1", "admin-1", false)
	if err != nil {
		t.Fatalf("adjudicate deny: %v", err)
	}
	if b.Adjudication != domain.AdjudicationDenied || b.ResolvedAt == nil {
		t.Fatalf("deny should mark denied + resolved, got %s resolvedAt=%v", b.Adjudication, b.ResolvedAt)
	}
	if repo.cases["case-1"].Status != domain.StatusInProgress {
		t.Fatalf("deny should resume the case to InProgress, got %s", repo.cases["case-1"].Status)
	}
	if problems.status != "InProgress" {
		t.Fatalf("problem status = %q, want InProgress", problems.status)
	}
}

func TestAdjudicate_AlreadyRuled(t *testing.T) {
	repo := newFakeRepo()
	c := blockedCase()
	c.Obstacles[0].Adjudication = domain.AdjudicationConfirmed
	repo.put(c)
	uc := application.NewAdjudicateObstacle(repo, &fakeProblems{}, &fakeAudit{}, &fakeNotifier{})

	if _, err := uc.Execute(context.Background(), "blk-1", "admin-1", false); err != domain.ErrAlreadyAdjudicated {
		t.Fatalf("re-adjudicate err = %v, want ErrAlreadyAdjudicated", err)
	}
}

// --- escalate ---

func TestEscalateOverdue(t *testing.T) {
	const D = 24 * time.Hour
	const R = 72 * time.Hour
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()

	// Overdue silent case: deadline just over one window ago → target level 2.
	repo.put(&domain.Case{ID: "c-silent", ProblemID: "p-silent", OfficialID: "off-1", Status: domain.StatusInProgress, Deadline: now.Add(-D - time.Minute)})
	// On-time case: deadline in the future → no escalation.
	repo.put(&domain.Case{ID: "c-ontime", ProblemID: "p-ontime", OfficialID: "off-1", Status: domain.StatusInProgress, Deadline: now.Add(D)})
	// Terminal case: Done → skipped.
	repo.put(&domain.Case{ID: "c-done", ProblemID: "p-done", OfficialID: "off-1", Status: domain.StatusDone, Deadline: now.Add(-10 * D)})
	// Blocked case past its review window → escalates on the blocker anchor.
	blk := &domain.Case{ID: "c-blocked", ProblemID: "p-blocked", OfficialID: "off-1", Status: domain.StatusBlocked, Deadline: now}
	blk.Obstacles = []domain.Obstacle{{ID: "b-1", CaseID: "c-blocked", CreatedAt: now.Add(-R - time.Minute), Adjudication: domain.AdjudicationPending}}
	repo.put(blk)

	audit := &fakeAudit{}
	uc := application.NewEscalateOverdue(repo, audit, &fakeLadder{}, D, R)

	res, err := uc.Run(context.Background(), now)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Escalated != 2 {
		t.Fatalf("escalated %d cases, want 2 (silent + blocked)", res.Escalated)
	}
	if got := repo.escalations["c-silent"]; got != 2 {
		t.Fatalf("silent case level = %d, want 2", got)
	}
	if got := repo.escalations["c-blocked"]; got != 1 {
		t.Fatalf("blocked case level = %d, want 1", got)
	}
	if _, seen := repo.escalations["c-ontime"]; seen {
		t.Fatal("on-time case should not have escalated")
	}
	if _, seen := repo.escalations["c-done"]; seen {
		t.Fatal("done case should not have escalated")
	}

	// Idempotent: a second run at the same instant escalates nothing new.
	if res2, _ := uc.Run(context.Background(), now); res2.Escalated != 0 {
		t.Fatalf("second run escalated %d, want 0 (idempotent)", res2.Escalated)
	}
}
