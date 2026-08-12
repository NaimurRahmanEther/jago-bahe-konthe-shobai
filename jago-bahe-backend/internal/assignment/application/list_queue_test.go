package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/assignment/application"
	"jago-bahe-backend/internal/assignment/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// --- fakes local to the queue tests ---

// queueProblems returns a fixed roster, unlike flow_test.go's single-view
// fakeProblems. The queue is the one use case that reads more than one row.
type queueProblems struct{ rows []application.ProblemView }

func (p *queueProblems) Get(context.Context, string) (application.ProblemView, error) {
	return application.ProblemView{}, domain.ErrProblemNotFound
}
func (p *queueProblems) ListAssignable(context.Context) ([]application.ProblemView, error) {
	return p.rows, nil
}
func (p *queueProblems) MarkAssigned(context.Context, string) error { return nil }

// brokenAreas fails to resolve exactly one area id, standing in for a problem
// whose location row has gone missing.
type brokenAreas struct{ orphan string }

func (b brokenAreas) GetByID(_ context.Context, id valueobject.AreaID) (*areadomain.Area, error) {
	if id.String() == b.orphan {
		return nil, areadomain.ErrAreaNotFound
	}
	return &areadomain.Area{ID: id, Level: areadomain.LevelUnion, ParentID: "upazila-1"}, nil
}
func (brokenAreas) List(context.Context) ([]areadomain.Area, error) { return nil, nil }
func (brokenAreas) Children(context.Context, valueobject.AreaID) ([]areadomain.Area, error) {
	return nil, nil
}

func queueRow(id, status, areaID, officialID string, validCount int) application.ProblemView {
	return application.ProblemView{
		ID:                    id,
		Title:                 "t",
		Status:                status,
		AreaID:                areaID,
		PointedOfficialID:     officialID,
		ValidCount:            validCount,
		EligibleForAssignment: true,
	}
}

func ids(views []application.ProblemView) []string {
	out := make([]string, 0, len(views))
	for _, v := range views {
		out = append(out, v.ID)
	}
	return out
}

// --- tests ---

// TestListQueue_ScopesToAdminsOwnUnion is the leak regression. Before B17 the
// queue ignored the caller entirely and returned every union's problems to every
// admin; scoping lived only in AssignWithinUnion, so the list handed out exactly
// what the action would have refused.
func TestListQueue_ScopesToAdminsOwnUnion(t *testing.T) {
	problems := &queueProblems{rows: []application.ProblemView{
		queueRow("mine", "Validated", "union-1", "off-union", 5),
		queueRow("theirs", "Validated", "union-2", "off-union", 5),
	}}
	uc := application.NewListQueue(problems, unionOfficials(), &fakeAdmins{unionOf: "union-1"}, fakeAreas{}, domain.NewService())

	got, err := uc.Execute(context.Background(), "admin-1")
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(got) != 1 || got[0].ID != "mine" {
		t.Fatalf("queue = %v, want [mine] — another union's problem leaked", ids(got))
	}
}

// TestListQueue_NoUnionFailsClosed asserts the empty-union guard returns an error
// AND no rows. A rows-only assertion would pass on an unscoped listing that simply
// happened to be empty, so check both: this is the branch that decides whether an
// admin with no linked official sees the whole seat.
func TestListQueue_NoUnionFailsClosed(t *testing.T) {
	problems := &queueProblems{rows: []application.ProblemView{
		queueRow("p1", "Validated", "union-1", "off-union", 5),
		queueRow("p2", "Reported", "union-2", "off-union", 0),
	}}
	uc := application.NewListQueue(problems, unionOfficials(), &fakeAdmins{unionOf: ""}, fakeAreas{}, domain.NewService())

	got, err := uc.Execute(context.Background(), "admin-nounion")
	if !errors.Is(err, domain.ErrNotUnionAdmin) {
		t.Fatalf("Execute() err = %v, want ErrNotUnionAdmin", err)
	}
	if len(got) != 0 {
		t.Fatalf("queue = %v, want none — an unscoped admin must not receive rows", ids(got))
	}
}

// TestListQueue_IncludesReportedAndValidated is the B17 rule: the admin watches a
// report climb toward V and decides for themselves when to forward it, so an
// approved report belongs in the queue from the moment it is public — including
// one with zero validations.
func TestListQueue_IncludesReportedAndValidated(t *testing.T) {
	problems := &queueProblems{rows: []application.ProblemView{
		queueRow("reported-zero", "Reported", "union-1", "off-union", 0),
		queueRow("reported-partial", "Reported", "union-1", "off-union", 3),
		queueRow("validated", "Validated", "union-1", "off-union", 5),
	}}
	uc := application.NewListQueue(problems, unionOfficials(), &fakeAdmins{unionOf: "union-1"}, fakeAreas{}, domain.NewService())

	got, err := uc.Execute(context.Background(), "admin-1")
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("queue = %v, want all three — a Reported problem is assignable since B17", ids(got))
	}
	for _, v := range got {
		if v.ID == "reported-partial" && v.ValidCount != 3 {
			t.Errorf("ValidCount = %d, want 3 — the count the admin judges by must survive the queue", v.ValidCount)
		}
	}
}

// TestListQueue_ExcludesAboveUnion is the B20 boundary. An above-union report is
// the super admin's to forward, so it must not sit in the list of what a union
// admin may ASSIGN — AssignWithinUnion answers ErrWrongRoute for it, and a queue
// that offers what the action refuses is the same leak B17 fixed at the union
// boundary. The admin still sees it, on ListMyForwarding, where they advise.
//
// The surviving row keeps Routing = "union" (not "vote": B20 moved above-union
// forwarding off the admin vote entirely). The literal is what the UI branches on,
// so it is pinned here as well as in the domain's Route test.
func TestListQueue_ExcludesAboveUnion(t *testing.T) {
	problems := &queueProblems{rows: []application.ProblemView{
		queueRow("union-level", "Validated", "union-1", "off-union", 5),
		queueRow("above-union", "Validated", "union-1", "off-mp", 5),
	}}
	uc := application.NewListQueue(problems, unionOfficials(), &fakeAdmins{unionOf: "union-1"}, fakeAreas{}, domain.NewService())

	got, err := uc.Execute(context.Background(), "admin-1")
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(got) != 1 || got[0].ID != "union-level" {
		t.Fatalf("queue = %v, want [union-level] — an above-union report is not this admin's to assign", ids(got))
	}
	if got[0].Routing != "union" {
		t.Errorf("routing = %q, want %q", got[0].Routing, "union")
	}
}

// TestListQueue_SkipsUnresolvableArea mirrors ListModeration: a problem whose
// location cannot be walked up to a union belongs to no admin's queue. Dropping it
// is deliberate — the alternative is showing it to every admin.
func TestListQueue_SkipsUnresolvableArea(t *testing.T) {
	problems := &queueProblems{rows: []application.ProblemView{
		queueRow("ok", "Validated", "union-1", "off-union", 5),
		queueRow("orphan", "Validated", "gone", "off-union", 5),
	}}
	uc := application.NewListQueue(problems, unionOfficials(), &fakeAdmins{unionOf: "union-1"}, brokenAreas{orphan: "gone"}, domain.NewService())

	got, err := uc.Execute(context.Background(), "admin-1")
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if len(got) != 1 || got[0].ID != "ok" {
		t.Fatalf("queue = %v, want [ok]", ids(got))
	}
}
