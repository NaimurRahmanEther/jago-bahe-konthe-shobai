package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"jago-bahe-backend/internal/assignment/application"
	"jago-bahe-backend/internal/assignment/domain"
)

// --- B20: above-union forwarding ---
//
// The vote these replaced is gone: there is no quorum to reach, no window to
// beat, and no no-majority fallback to the Union Chairman. The union admins
// advise; the super admin decides (CLAUDE.md A.3.8).

func aboveProblem() application.ProblemView {
	return application.ProblemView{
		ID: "prob-1", Title: "t", Status: "Validated",
		PointedOfficialID: "off-mp", AreaID: problemUnion, ReporterID: "reporter-1",
		EligibleForAssignment: true,
	}
}

func forwardingOfficials() *fakeOfficials {
	o := unionOfficials()
	o.byID["off-upz"] = application.OfficialView{ID: "off-upz", Tier: "upazila_chairman", AreaID: "upazila-1"}
	return o
}

func newSuggest(repo *fakeAssignRepo, problems *fakeProblems, admins *fakeAdmins, audit *fakeAudit) *application.SuggestForwarding {
	return application.NewSuggestForwarding(repo, problems, forwardingOfficials(), admins, domain.NewService(), audit)
}

// newForward wires the use case with a throwaway notifier. Tests that care what was
// SENT use newForwardNotifying instead and keep the recorder.
func newForward(repo *fakeAssignRepo, problems *fakeProblems, audit *fakeAudit) *application.ForwardProblem {
	return newForwardNotifying(repo, problems, audit, &fakeNotifier{})
}

func newForwardNotifying(repo *fakeAssignRepo, problems *fakeProblems, audit *fakeAudit, notify *fakeNotifier) *application.ForwardProblem {
	return application.NewForwardProblem(repo, problems, forwardingOfficials(), domain.NewService(), audit, notify, time.Hour)
}

// --- advising ---

func TestSuggestForwarding_RecordsAdviceAndAudits(t *testing.T) {
	repo := newFakeRepo()
	audit := &fakeAudit{}
	uc := newSuggest(repo, &fakeProblems{view: aboveProblem()}, &fakeAdmins{eligible: []string{"admin-1"}}, audit)

	s, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-upz", "closer to the works department")
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if s.SuggestedOfficialID != "off-upz" {
		t.Fatalf("suggested = %q, want off-upz", s.SuggestedOfficialID)
	}
	if audit.lastAction() != "forwarding_suggested" {
		t.Fatalf("audit action = %q, want forwarding_suggested", audit.lastAction())
	}
}

// An admin outside the scope must be refused AND leave no row: a rows-only
// assertion would pass even if the guard ran after the write.
func TestSuggestForwarding_RefusesAnAdminWithoutStanding(t *testing.T) {
	repo := newFakeRepo()
	uc := newSuggest(repo, &fakeProblems{view: aboveProblem()}, &fakeAdmins{eligible: []string{"admin-1", "admin-2"}}, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-outsider", "off-mp", ""); !errors.Is(err, domain.ErrNotEligibleAdviser) {
		t.Fatalf("err = %v, want ErrNotEligibleAdviser", err)
	}
	if got, _ := repo.SuggestionsByProblem(context.Background(), "prob-1"); len(got) != 0 {
		t.Fatalf("an outsider's advice was stored: %+v", got)
	}
}

// Advice settles nothing, so an admin persuaded by the others may change it. The
// ballot this replaced refused a second cast, because that WOULD have changed a
// result.
func TestSuggestForwarding_AdviceIsChangeable(t *testing.T) {
	repo := newFakeRepo()
	uc := newSuggest(repo, &fakeProblems{view: aboveProblem()}, &fakeAdmins{eligible: []string{"admin-1"}}, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-mp", ""); err != nil {
		t.Fatalf("first advice: %v", err)
	}
	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-upz", "changed my mind"); err != nil {
		t.Fatalf("second advice: %v", err)
	}
	got, _ := repo.SuggestionsByProblem(context.Background(), "prob-1")
	if len(got) != 1 {
		t.Fatalf("changing advice left %d rows, want 1", len(got))
	}
	if got[0].SuggestedOfficialID != "off-upz" {
		t.Fatalf("advice = %q, want the replacement off-upz", got[0].SuggestedOfficialID)
	}
}

func TestSuggestForwarding_RejectsUnionLevelProblem(t *testing.T) {
	uc := newSuggest(newFakeRepo(), &fakeProblems{view: unionProblem()}, &fakeAdmins{eligible: []string{"admin-1"}}, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-mp", ""); !errors.Is(err, domain.ErrWrongRoute) {
		t.Fatalf("err = %v, want ErrWrongRoute — a union-level report is its own admin's", err)
	}
}

// Advising a union-level official would be advice the super admin cannot act on.
func TestSuggestForwarding_RejectsAUnionLevelTarget(t *testing.T) {
	uc := newSuggest(newFakeRepo(), &fakeProblems{view: aboveProblem()}, &fakeAdmins{eligible: []string{"admin-1"}}, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-union", ""); !errors.Is(err, domain.ErrWrongRoute) {
		t.Fatalf("err = %v, want ErrWrongRoute", err)
	}
}

func TestSuggestForwarding_RefusesAnIneligibleProblem(t *testing.T) {
	pending := application.ProblemView{
		ID: "prob-1", Status: "PendingApproval", PointedOfficialID: "off-mp",
		AreaID: problemUnion, EligibleForAssignment: false,
	}
	uc := newSuggest(newFakeRepo(), &fakeProblems{view: pending}, &fakeAdmins{eligible: []string{"admin-1"}}, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-mp", ""); !errors.Is(err, domain.ErrNotAssignable) {
		t.Fatalf("err = %v, want ErrNotAssignable", err)
	}
}

// --- forwarding ---

// The signal-not-a-gate rule (A.3.1.1, applied to advice): the super admin may
// forward a report nobody has advised on. The old vote could not settle without
// quorum, which is exactly the gate this removes.
func TestForwardProblem_ForwardsWithNoAdviceAtAll(t *testing.T) {
	repo := newFakeRepo()
	problems := &fakeProblems{view: aboveProblem()}
	audit := &fakeAudit{}
	uc := newForward(repo, problems, audit)

	a, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-mp", "", nil, "")
	if err != nil {
		t.Fatalf("forwarding with no advice: %v", err)
	}
	if a.OfficialID != "off-mp" {
		t.Fatalf("forwarded to %q, want off-mp", a.OfficialID)
	}
	if !problems.assigned {
		t.Fatal("problem was not marked assigned")
	}
	// Matches the reporter's pointed official and there is no top, so nothing was
	// departed from and the plain `assigned` literal is correct.
	if audit.lastAction() != "assigned" {
		t.Fatalf("audit action = %q, want assigned", audit.lastAction())
	}
	// The shared assigner ran: the monitor is set, so B19's observation ladder
	// works on a forwarded case exactly as on a union-assigned one.
	if a.MonitorOfficialID != "off-upazila" {
		t.Fatalf("monitor = %q, want the ladder's answer", a.MonitorOfficialID)
	}
}

func TestForwardProblem_DepartingFromTheReporterNeedsAReason(t *testing.T) {
	repo := newFakeRepo()
	uc := newForward(repo, &fakeProblems{view: aboveProblem()}, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-upz", "", nil, ""); !errors.Is(err, domain.ErrReasonRequired) {
		t.Fatalf("err = %v, want ErrReasonRequired", err)
	}
}

func TestForwardProblem_DepartingFromTheAdvisersNeedsAReason(t *testing.T) {
	repo := newFakeRepo()
	problems := &fakeProblems{view: aboveProblem()}
	suggest := newSuggest(repo, problems, &fakeAdmins{eligible: []string{"admin-1", "admin-2"}}, &fakeAudit{})
	// Both advisers say off-upz; the reporter pointed at off-mp. Now NO choice
	// satisfies both, so every forward needs a reason — the intended consequence.
	_, _ = suggest.Execute(context.Background(), "prob-1", "admin-1", "off-upz", "")
	_, _ = suggest.Execute(context.Background(), "prob-1", "admin-2", "off-upz", "")

	uc := newForward(repo, problems, &fakeAudit{})
	if _, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-mp", "", nil, ""); !errors.Is(err, domain.ErrReasonRequired) {
		t.Fatalf("forwarding to the reporter's pick against the advisers: err = %v, want ErrReasonRequired", err)
	}
	if _, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-upz", "", nil, ""); !errors.Is(err, domain.ErrReasonRequired) {
		t.Fatalf("forwarding to the advisers' pick against the reporter: err = %v, want ErrReasonRequired", err)
	}
}

func TestForwardProblem_WithAReasonIsAuditedAsAnOverride(t *testing.T) {
	repo := newFakeRepo()
	audit := &fakeAudit{}
	uc := newForward(repo, &fakeProblems{view: aboveProblem()}, audit)

	a, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-upz", "", nil, "the upazila holds the budget line")
	if err != nil {
		t.Fatalf("forward with reason: %v", err)
	}
	if a.OverrideReason != "the upazila holds the budget line" {
		t.Fatalf("reason = %q, want it on the public record", a.OverrideReason)
	}
	if audit.lastAction() != "assignment_overridden" {
		t.Fatalf("audit action = %q, want assignment_overridden", audit.lastAction())
	}
}

// Agreeing with everyone needs no reason, and must NOT be logged as an override —
// the public feed filters on these exact literals (A.3.5).
func TestForwardProblem_AgreementNeedsNoReasonAndIsNotAnOverride(t *testing.T) {
	repo := newFakeRepo()
	problems := &fakeProblems{view: aboveProblem()}
	suggest := newSuggest(repo, problems, &fakeAdmins{eligible: []string{"admin-1"}}, &fakeAudit{})
	_, _ = suggest.Execute(context.Background(), "prob-1", "admin-1", "off-mp", "")

	audit := &fakeAudit{}
	if _, err := newForward(repo, problems, audit).Execute(context.Background(), "prob-1", "super-1", "off-mp", "", nil, ""); err != nil {
		t.Fatalf("forwarding in agreement: %v", err)
	}
	if audit.lastAction() != "assigned" {
		t.Fatalf("audit action = %q, want assigned", audit.lastAction())
	}
}

func TestForwardProblem_RejectsAUnionLevelProblem(t *testing.T) {
	uc := newForward(newFakeRepo(), &fakeProblems{view: unionProblem()}, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-mp", "", nil, "x"); !errors.Is(err, domain.ErrWrongRoute) {
		t.Fatalf("err = %v, want ErrWrongRoute — union-level reports are not the super admin's", err)
	}
}

func TestForwardProblem_RejectsAUnionLevelTarget(t *testing.T) {
	uc := newForward(newFakeRepo(), &fakeProblems{view: aboveProblem()}, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-union", "", nil, "x"); !errors.Is(err, domain.ErrWrongRoute) {
		t.Fatalf("err = %v, want ErrWrongRoute", err)
	}
}

// --- the queues ---

func TestListForwardingQueue_HoldsOnlyAboveUnionReports(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewListForwardingQueue(repo, &fakeProblems{view: aboveProblem()}, forwardingOfficials(), domain.NewService())
	rows, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if len(rows) != 1 || rows[0].Problem.ID != "prob-1" {
		t.Fatalf("queue = %+v, want the above-union report", rows)
	}

	unionOnly := application.NewListForwardingQueue(repo, &fakeProblems{view: unionProblem()}, forwardingOfficials(), domain.NewService())
	if rows, _ := unionOnly.Execute(context.Background()); len(rows) != 0 {
		t.Fatalf("a union-level report reached the super admin's queue: %+v", rows)
	}
}

func TestListForwardingQueue_CarriesTheAdvisoryTally(t *testing.T) {
	repo := newFakeRepo()
	problems := &fakeProblems{view: aboveProblem()}
	suggest := newSuggest(repo, problems, &fakeAdmins{eligible: []string{"admin-1", "admin-2"}}, &fakeAudit{})
	_, _ = suggest.Execute(context.Background(), "prob-1", "admin-1", "off-upz", "")
	_, _ = suggest.Execute(context.Background(), "prob-1", "admin-2", "off-upz", "")

	rows, _ := application.NewListForwardingQueue(repo, problems, forwardingOfficials(), domain.NewService()).Execute(context.Background())
	if len(rows) != 1 {
		t.Fatalf("queue = %+v, want one row", rows)
	}
	if rows[0].TopOfficialID != "off-upz" || rows[0].TopCount != 2 {
		t.Fatalf("tally = (%q, %d), want (off-upz, 2)", rows[0].TopOfficialID, rows[0].TopCount)
	}
	// The super admin advises on nothing, so their rows never carry an own-advice.
	if rows[0].MySuggestion != nil {
		t.Fatalf("the super admin's row carried MySuggestion = %+v", rows[0].MySuggestion)
	}
}

func TestListMyForwarding_ScopesToAdvisersAndCarriesTheirOwnAdvice(t *testing.T) {
	repo := newFakeRepo()
	problems := &fakeProblems{view: aboveProblem()}
	admins := &fakeAdmins{eligible: []string{"admin-1"}}
	suggest := newSuggest(repo, problems, admins, &fakeAudit{})
	_, _ = suggest.Execute(context.Background(), "prob-1", "admin-1", "off-upz", "nearer the works dept")

	uc := application.NewListMyForwarding(repo, problems, forwardingOfficials(), admins, domain.NewService())

	rows, err := uc.Execute(context.Background(), "admin-1")
	if err != nil {
		t.Fatalf("my forwarding: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("an eligible adviser sees %d rows, want 1", len(rows))
	}
	if rows[0].MySuggestion == nil || rows[0].MySuggestion.SuggestedOfficialID != "off-upz" {
		t.Fatalf("MySuggestion = %+v, want the caller's own advice", rows[0].MySuggestion)
	}

	// An admin without standing sees none of it.
	outsider, err := uc.Execute(context.Background(), "admin-outsider")
	if err != nil {
		t.Fatalf("outsider list: %v", err)
	}
	if len(outsider) != 0 {
		t.Fatalf("an admin without standing saw %d rows, want 0", len(outsider))
	}
}

func TestListMyForwarding_RefusesAnEmptyCaller(t *testing.T) {
	uc := application.NewListMyForwarding(newFakeRepo(), &fakeProblems{view: aboveProblem()}, forwardingOfficials(), &fakeAdmins{}, domain.NewService())
	if _, err := uc.Execute(context.Background(), ""); !errors.Is(err, domain.ErrNotEligibleAdviser) {
		t.Fatalf("err = %v, want ErrNotEligibleAdviser", err)
	}
}

// Reads are never audited: an entry per view would publish who is looking at what.
func TestForwardingQueues_AreNeverAudited(t *testing.T) {
	repo := newFakeRepo()
	problems := &fakeProblems{view: aboveProblem()}
	audit := &fakeAudit{}
	admins := &fakeAdmins{eligible: []string{"admin-1"}}

	_, _ = application.NewListForwardingQueue(repo, problems, forwardingOfficials(), domain.NewService()).Execute(context.Background())
	_, _ = application.NewListMyForwarding(repo, problems, forwardingOfficials(), admins, domain.NewService()).Execute(context.Background(), "admin-1")

	if len(audit.entries) != 0 {
		t.Fatalf("the queues wrote %d audit entries, want none", len(audit.entries))
	}
}
