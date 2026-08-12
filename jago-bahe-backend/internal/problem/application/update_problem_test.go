package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// fakeMutRepo is a full domain.Repository for the reporter self-management tests.
// It replays a fixed problem from GetByID and records whether the mutating call
// (Update / SetWithdrawn / Delete) ever reached storage — the guard tests pin that
// a refused request never writes.
type fakeMutRepo struct {
	problem    *domain.Problem
	getErr     error
	updateErr  error
	setWithErr error
	deleteErr  error

	updated     *domain.Problem
	updateCalls int
	withdrawn   struct {
		id   string
		from domain.Status
	}
	withdrawCalls int

	deletedID   string
	deleteCalls int
}

func (r *fakeMutRepo) GetByID(_ context.Context, _ string) (*domain.Problem, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	// Hand back a copy so the use case's mutations don't retroactively change the
	// fixture the test asserts the "before" of.
	p := *r.problem
	return &p, nil
}

func (r *fakeMutRepo) Update(_ context.Context, p *domain.Problem) error {
	r.updateCalls++
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updated = p
	return nil
}

func (r *fakeMutRepo) SetWithdrawn(_ context.Context, id string, from domain.Status) error {
	r.withdrawCalls++
	if r.setWithErr != nil {
		return r.setWithErr
	}
	r.withdrawn.id = id
	r.withdrawn.from = from
	return nil
}

func (r *fakeMutRepo) Delete(_ context.Context, id string) error {
	r.deleteCalls++
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deletedID = id
	return nil
}

// The rest of the port is unused by these use cases.
func (r *fakeMutRepo) Create(context.Context, *domain.Problem) error { return nil }
func (r *fakeMutRepo) List(context.Context, domain.Filter) ([]domain.Problem, error) {
	return nil, nil
}
func (r *fakeMutRepo) AddVote(context.Context, domain.ValidationVote) error { return nil }
func (r *fakeMutRepo) ListVotes(context.Context, string) ([]domain.ValidationVote, error) {
	return nil, nil
}
func (r *fakeMutRepo) UpdateValidation(context.Context, string, int, domain.Status) error {
	return nil
}
func (r *fakeMutRepo) SetStatus(context.Context, string, domain.Status) error { return nil }
func (r *fakeMutRepo) SetScreening(context.Context, string, domain.Status, domain.Status, domain.RejectionReason, string, time.Time) error {
	return nil
}
func (r *fakeMutRepo) VotesByViewer(context.Context, string, []string) (map[string]domain.VoteChoice, error) {
	return nil, nil
}
func (r *fakeMutRepo) TitlesByIDs(context.Context, []string) (map[string]string, error) {
	return nil, nil
}

var _ domain.Repository = (*fakeMutRepo)(nil)

// fakeAudit records appended entries so a test can assert an action was written,
// and records erasures so the hard-delete tests can pin that the trail was removed.
type fakeAudit struct {
	entries []auditdomain.AuditEntry

	deletedTarget struct{ targetType, targetID string }
	deleteCalls   int
	deleteErr     error
}

func (a *fakeAudit) Append(_ context.Context, e auditdomain.AuditEntry) error {
	a.entries = append(a.entries, e)
	return nil
}
func (a *fakeAudit) ListByTarget(context.Context, string, string) ([]auditdomain.AuditEntry, error) {
	return nil, nil
}
func (a *fakeAudit) ListByActions(context.Context, []string, int) ([]auditdomain.AuditEntry, error) {
	return nil, nil
}
func (a *fakeAudit) DeleteByTarget(_ context.Context, targetType, targetID string) error {
	a.deleteCalls++
	if a.deleteErr != nil {
		return a.deleteErr
	}
	a.deletedTarget.targetType = targetType
	a.deletedTarget.targetID = targetID
	return nil
}

var _ auditdomain.Repository = (*fakeAudit)(nil)

func editableProblem() *domain.Problem {
	return &domain.Problem{
		ID:                "prob-1",
		Title:             "old title",
		Description:       "old description",
		Location:          valueobject.Location{AreaID: valueobject.AreaID("ward-1"), Address: "old addr"},
		ReporterID:        "acct-1",
		PointedOfficialID: "off-1",
		Status:            domain.StatusReported,
		ValidCount:        0,
	}
}

// TestUpdateProblemAppliesEditAndAudits pins the happy path: the reporter's own
// still-editable report takes the new content, the write reaches storage, and an
// "edited" audit entry is appended by the reporter.
func TestUpdateProblemAppliesEditAndAudits(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem()}
	audit := &fakeAudit{}
	uc := application.NewUpdateProblem(repo, audit)

	out, err := uc.Execute(context.Background(), application.UpdateProblemInput{
		ProblemID:        "prob-1",
		CallerID:         "acct-1",
		Title:            "new title",
		Description:      "new description",
		Address:          "new addr",
		ProposedSolution: "fix it",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if out.Title != "new title" || out.Description != "new description" ||
		out.Location.Address != "new addr" || out.ProposedSolution != "fix it" {
		t.Errorf("edit not applied: %+v", out)
	}
	if repo.updateCalls != 1 || repo.updated == nil {
		t.Errorf("Update reached storage %d times, want 1", repo.updateCalls)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "edited" || audit.entries[0].Actor != "acct-1" {
		t.Errorf("audit = %+v, want one 'edited' entry by acct-1", audit.entries)
	}
}

// TestUpdateProblemRefusesNonReporter pins that ownership is checked before the
// window: a stranger is told they are not the reporter (403) and no write is ever
// attempted, so they never learn the problem's window state.
func TestUpdateProblemRefusesNonReporter(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem()}
	uc := application.NewUpdateProblem(repo, &fakeAudit{})

	_, err := uc.Execute(context.Background(), application.UpdateProblemInput{
		ProblemID: "prob-1",
		CallerID:  "acct-2",
		Title:     "hijack",
	})
	if !errors.Is(err, domain.ErrNotReporter) {
		t.Fatalf("err = %v, want ErrNotReporter", err)
	}
	if repo.updateCalls != 0 {
		t.Errorf("Update called %d times, want 0 — a stranger's edit must never write", repo.updateCalls)
	}
}

// TestUpdateProblemRefusesEmptyCaller pins that an absent caller is a non-reporter,
// never an accidental owner.
func TestUpdateProblemRefusesEmptyCaller(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem()}
	_, err := application.NewUpdateProblem(repo, &fakeAudit{}).Execute(
		context.Background(), application.UpdateProblemInput{ProblemID: "prob-1", CallerID: ""})
	if !errors.Is(err, domain.ErrNotReporter) {
		t.Fatalf("err = %v, want ErrNotReporter", err)
	}
	if repo.updateCalls != 0 {
		t.Errorf("Update called %d times, want 0", repo.updateCalls)
	}
}

// TestUpdateProblemRefusesPastWindow pins that once the report has been endorsed
// (a valid vote landed, so ValidCount > 0), the reporter can no longer rewrite the
// text residents validated — the use case returns ErrNotEditable.
func TestUpdateProblemRefusesPastWindow(t *testing.T) {
	endorsed := editableProblem()
	endorsed.ValidCount = 1
	repo := &fakeMutRepo{problem: endorsed}
	uc := application.NewUpdateProblem(repo, &fakeAudit{})

	_, err := uc.Execute(context.Background(), application.UpdateProblemInput{
		ProblemID: "prob-1", CallerID: "acct-1", Title: "too late",
	})
	if !errors.Is(err, domain.ErrNotEditable) {
		t.Fatalf("err = %v, want ErrNotEditable", err)
	}
	if repo.updateCalls != 0 {
		t.Errorf("Update called %d times, want 0 — a frozen report must never write", repo.updateCalls)
	}
}
