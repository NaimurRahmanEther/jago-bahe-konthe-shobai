package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"jago-bahe-backend/internal/assignment/application"
	"jago-bahe-backend/internal/assignment/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// --- in-memory fakes ---

type fakeAssignRepo struct {
	assignments map[string]*domain.Assignment  // by problemID
	suggestions map[string][]domain.Suggestion // by problemID
}

func newFakeRepo() *fakeAssignRepo {
	return &fakeAssignRepo{assignments: map[string]*domain.Assignment{}, suggestions: map[string][]domain.Suggestion{}}
}

func (r *fakeAssignRepo) CreateAssignment(_ context.Context, a *domain.Assignment) error {
	if _, ok := r.assignments[a.ProblemID]; ok {
		return domain.ErrAlreadyAssigned
	}
	r.assignments[a.ProblemID] = a
	return nil
}
func (r *fakeAssignRepo) GetAssignmentByProblem(_ context.Context, problemID string) (*domain.Assignment, error) {
	if a, ok := r.assignments[problemID]; ok {
		return a, nil
	}
	return nil, domain.ErrAssignmentNotFound
}
func (r *fakeAssignRepo) AssignmentsByProblems(_ context.Context, problemIDs []string) (map[string]domain.Assignment, error) {
	out := make(map[string]domain.Assignment)
	for _, id := range problemIDs {
		if a, ok := r.assignments[id]; ok {
			out[id] = *a
		}
	}
	return out, nil
}
func (r *fakeAssignRepo) ListByOfficial(_ context.Context, officialID string) ([]domain.Assignment, error) {
	var out []domain.Assignment
	for _, a := range r.assignments {
		if a.OfficialID == officialID {
			out = append(out, *a)
		}
	}
	return out, nil
}

// UpsertSuggestion mirrors the real unique index: one row per admin per problem,
// so a second suggestion REPLACES the first rather than adding one.
func (r *fakeAssignRepo) UpsertSuggestion(_ context.Context, s *domain.Suggestion) error {
	existing := r.suggestions[s.ProblemID]
	for i := range existing {
		if existing[i].AdminAccountID == s.AdminAccountID {
			existing[i] = *s
			r.suggestions[s.ProblemID] = existing
			return nil
		}
	}
	r.suggestions[s.ProblemID] = append(existing, *s)
	return nil
}

func (r *fakeAssignRepo) SuggestionsByProblem(_ context.Context, problemID string) ([]domain.Suggestion, error) {
	return append([]domain.Suggestion(nil), r.suggestions[problemID]...), nil
}

func (r *fakeAssignRepo) SuggestionsByProblems(_ context.Context, problemIDs []string) (map[string][]domain.Suggestion, error) {
	out := make(map[string][]domain.Suggestion, len(problemIDs))
	for _, id := range problemIDs {
		if s, ok := r.suggestions[id]; ok {
			out[id] = append([]domain.Suggestion(nil), s...)
		}
	}
	return out, nil
}

type fakeProblems struct {
	view     application.ProblemView
	assigned bool
}

func (p *fakeProblems) Get(context.Context, string) (application.ProblemView, error) {
	return p.view, nil
}
func (p *fakeProblems) ListAssignable(context.Context) ([]application.ProblemView, error) {
	return []application.ProblemView{p.view}, nil
}
func (p *fakeProblems) MarkAssigned(_ context.Context, _ string) error {
	p.assigned = true
	p.view.Status = "Assigned"
	p.view.EligibleForAssignment = false // a case exists now; it leaves the queue
	return nil
}

type fakeOfficials struct {
	byID    map[string]application.OfficialView
	monitor string
}

func (o *fakeOfficials) Get(_ context.Context, id string) (application.OfficialView, error) {
	if v, ok := o.byID[id]; ok {
		return v, nil
	}
	return application.OfficialView{}, domain.ErrOfficialNotFound
}
func (o *fakeOfficials) MonitorFor(context.Context, string) (string, error) { return o.monitor, nil }

type fakeAdmins struct {
	unionOf  string
	eligible []string
}

func (a *fakeAdmins) UnionOf(context.Context, string) (string, error) { return a.unionOf, nil }
func (a *fakeAdmins) EligibleAdvisers(context.Context, domain.AdviceScope, string) ([]string, error) {
	return a.eligible, nil
}

type fakeAudit struct{ entries []auditdomain.AuditEntry }

func (f *fakeAudit) Append(_ context.Context, e auditdomain.AuditEntry) error {
	f.entries = append(f.entries, e)
	return nil
}
func (f *fakeAudit) ListByTarget(context.Context, string, string) ([]auditdomain.AuditEntry, error) {
	return f.entries, nil
}

func (f *fakeAudit) ListByActions(context.Context, []string, int) ([]auditdomain.AuditEntry, error) {
	return f.entries, nil
}
func (f *fakeAudit) DeleteByTarget(context.Context, string, string) error { return nil }
func (f *fakeAudit) lastAction() string {
	if len(f.entries) == 0 {
		return ""
	}
	return f.entries[len(f.entries)-1].Action
}

// fakeAreas resolves union-1 as a union under upazila-1, sufficient for the
// resolveUnion / fallback paths the use cases exercise.
type fakeAreas struct{}

func (fakeAreas) GetByID(_ context.Context, id valueobject.AreaID) (*areadomain.Area, error) {
	return &areadomain.Area{ID: id, Level: areadomain.LevelUnion, ParentID: "upazila-1"}, nil
}
func (fakeAreas) List(context.Context) ([]areadomain.Area, error) { return nil, nil }
func (fakeAreas) Children(context.Context, valueobject.AreaID) ([]areadomain.Area, error) {
	return nil, nil
}

// --- fixtures ---

const (
	unionTier    = valueobject.TierUnionChairman
	aboveTier    = valueobject.TierMP
	problemUnion = "union-1"
)

func unionProblem() application.ProblemView {
	return application.ProblemView{ID: "prob-1", Title: "t", Status: "Validated", PointedOfficialID: "off-union", AreaID: problemUnion, ReporterID: "reporter-1", EligibleForAssignment: true}
}

func unionOfficials() *fakeOfficials {
	return &fakeOfficials{
		byID: map[string]application.OfficialView{
			"off-union": {ID: "off-union", Tier: unionTier, AreaID: problemUnion, Name: "করিম উদ্দিন"},
			"off-other": {ID: "off-other", Tier: unionTier, AreaID: problemUnion, Name: "রহিম মিয়া"},
			"off-chair": {ID: "off-chair", Tier: unionTier, AreaID: problemUnion, Name: "সালেহা বেগম"},
			"off-mp":    {ID: "off-mp", Tier: aboveTier, AreaID: "seat-1", Name: "শহীদুজ্জামান সরকার"},
		},
		monitor: "off-upazila",
	}
}

// --- union-level assignment ---

func TestAssignWithinUnion_Confirm(t *testing.T) {
	repo := newFakeRepo()
	problems := &fakeProblems{view: unionProblem()}
	audit := &fakeAudit{}
	uc := application.NewAssignWithinUnion(repo, problems, unionOfficials(), &fakeAdmins{unionOf: problemUnion}, fakeAreas{}, domain.NewService(), audit, &fakeNotifier{}, 7*24*time.Hour)

	a, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-union", "", nil, "")
	if err != nil {
		t.Fatalf("confirm assign: %v", err)
	}
	if a.OfficialID != "off-union" || a.OverrideReason != "" {
		t.Fatalf("unexpected assignment: %+v", a)
	}
	if !problems.assigned {
		t.Fatal("problem was not marked assigned")
	}
	if audit.lastAction() != "assigned" {
		t.Fatalf("audit action = %q, want assigned", audit.lastAction())
	}
}

func TestAssignWithinUnion_OverrideRequiresReason(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewAssignWithinUnion(repo, &fakeProblems{view: unionProblem()}, unionOfficials(), &fakeAdmins{unionOf: problemUnion}, fakeAreas{}, domain.NewService(), &fakeAudit{}, &fakeNotifier{}, time.Hour)

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-other", "", nil, ""); err != domain.ErrMissingOverrideReason {
		t.Fatalf("override without reason err = %v, want ErrMissingOverrideReason", err)
	}
}

func TestAssignWithinUnion_OverrideWithReason(t *testing.T) {
	repo := newFakeRepo()
	audit := &fakeAudit{}
	uc := application.NewAssignWithinUnion(repo, &fakeProblems{view: unionProblem()}, unionOfficials(), &fakeAdmins{unionOf: problemUnion}, fakeAreas{}, domain.NewService(), audit, &fakeNotifier{}, time.Hour)

	a, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-other", "high", nil, "closer to the site")
	if err != nil {
		t.Fatalf("override assign: %v", err)
	}
	if a.OfficialID != "off-other" || a.OverrideReason != "closer to the site" {
		t.Fatalf("unexpected override assignment: %+v", a)
	}
	if audit.lastAction() != "assignment_overridden" {
		t.Fatalf("audit action = %q, want assignment_overridden", audit.lastAction())
	}
}

// TestAssignWithinUnion_AssignsReportedWithNoVotes is the B17 rule at the action
// step: an approved report is forwardable at once, with zero validations, if the
// admin judges it trustworthy. Before B17 this returned ErrNotValidated and the
// report could not reach an official until V distinct residents had validated it.
// The count is now evidence the admin weighs, not a condition they wait on (A.3.1).
func TestAssignWithinUnion_AssignsReportedWithNoVotes(t *testing.T) {
	repo := newFakeRepo()
	audit := &fakeAudit{}
	reported := application.ProblemView{
		ID: "prob-1", Title: "t", Status: "Reported", PointedOfficialID: "off-union",
		AreaID: problemUnion, ValidCount: 0, EligibleForAssignment: true,
	}
	problems := &fakeProblems{view: reported}
	uc := application.NewAssignWithinUnion(repo, problems, unionOfficials(), &fakeAdmins{unionOf: problemUnion}, fakeAreas{}, domain.NewService(), audit, &fakeNotifier{}, time.Hour)

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-union", "", nil, ""); err != nil {
		t.Fatalf("assigning a zero-vote Reported problem: %v", err)
	}
	if !problems.assigned {
		t.Fatal("problem was not marked assigned")
	}
	if audit.lastAction() != "assigned" {
		t.Fatalf("audit action = %q, want assigned", audit.lastAction())
	}
}

// TestAssignWithinUnion_RefusesIneligible covers the other side of the window.
// The per-status truth table lives in problem/domain's status_test.go, which owns
// the vocabulary; here we only prove the use case honours the flag it is handed —
// an unscreened, terminal, or already-assigned problem never reaches an official.
func TestAssignWithinUnion_RefusesIneligible(t *testing.T) {
	pending := application.ProblemView{
		ID: "prob-1", Title: "t", Status: "PendingApproval", PointedOfficialID: "off-union",
		AreaID: problemUnion, EligibleForAssignment: false,
	}
	uc := application.NewAssignWithinUnion(newFakeRepo(), &fakeProblems{view: pending}, unionOfficials(), &fakeAdmins{unionOf: problemUnion}, fakeAreas{}, domain.NewService(), &fakeAudit{}, &fakeNotifier{}, time.Hour)

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-union", "", nil, ""); !errors.Is(err, domain.ErrNotAssignable) {
		t.Fatalf("assigning an ineligible problem err = %v, want ErrNotAssignable", err)
	}
}

// The above-union half of this gate moved with the vote: it is now
// TestSuggestForwarding_RefusesAnIneligibleProblem in forwarding_test.go. The two
// paths still share one rule and still must move together — they were two
// separate copies of the same string compare before B17.

func TestAssignWithinUnion_RejectsAboveUnionTier(t *testing.T) {
	problems := &fakeProblems{view: application.ProblemView{ID: "prob-1", Status: "Validated", PointedOfficialID: "off-mp", AreaID: problemUnion, EligibleForAssignment: true}}
	uc := application.NewAssignWithinUnion(newFakeRepo(), problems, unionOfficials(), &fakeAdmins{unionOf: problemUnion}, fakeAreas{}, domain.NewService(), &fakeAudit{}, &fakeNotifier{}, time.Hour)

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-mp", "", nil, ""); err != domain.ErrWrongRoute {
		t.Fatalf("above-union assign err = %v, want ErrWrongRoute", err)
	}
}

func TestAssignWithinUnion_RejectsForeignUnionAdmin(t *testing.T) {
	uc := application.NewAssignWithinUnion(newFakeRepo(), &fakeProblems{view: unionProblem()}, unionOfficials(), &fakeAdmins{unionOf: "union-2"}, fakeAreas{}, domain.NewService(), &fakeAudit{}, &fakeNotifier{}, time.Hour)

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-union", "", nil, ""); err != domain.ErrNotUnionAdmin {
		t.Fatalf("foreign-union admin err = %v, want ErrNotUnionAdmin", err)
	}
}
