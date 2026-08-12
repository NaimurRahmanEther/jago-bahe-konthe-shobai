package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// fakeAreas resolves a fixed area tree for the screening-scope tests. ward-1 sits
// in union-1; that is all ensureUnionAdmin needs to walk.
type fakeAreas struct{ byID map[string]*areadomain.Area }

func newFakeAreas() fakeAreas {
	return fakeAreas{byID: map[string]*areadomain.Area{
		"ward-1":  {ID: valueobject.AreaID("ward-1"), Level: areadomain.LevelWard, ParentID: valueobject.AreaID("union-1")},
		"union-1": {ID: valueobject.AreaID("union-1"), Level: areadomain.LevelUnion, ParentID: valueobject.AreaID("upazila-1")},
	}}
}

func (f fakeAreas) GetByID(_ context.Context, id valueobject.AreaID) (*areadomain.Area, error) {
	a, ok := f.byID[id.String()]
	if !ok {
		return nil, areadomain.ErrAreaNotFound
	}
	return a, nil
}
func (fakeAreas) List(context.Context) ([]areadomain.Area, error) { return nil, nil }
func (fakeAreas) Children(context.Context, valueobject.AreaID) ([]areadomain.Area, error) {
	return nil, nil
}

var _ areadomain.Repository = fakeAreas{}

// fakeAdmins maps an admin account id to the union they administer. An id absent
// from the map has no union on record ("").
type fakeAdmins struct{ union map[string]string }

func (f fakeAdmins) UnionOf(_ context.Context, adminAccountID string) (string, error) {
	return f.union[adminAccountID], nil
}

var _ application.Admins = fakeAdmins{}

func pendingProblem() *domain.Problem {
	return &domain.Problem{
		ID:         "prob-1",
		Title:      "broken culvert",
		Location:   valueobject.Location{AreaID: valueobject.AreaID("ward-1")},
		ReporterID: "reporter-1",
		Status:     domain.StatusPendingApproval,
	}
}

// TestApproveProblemPublishesAndAudits pins the happy path: the union's own admin
// approves a pending report, it becomes Reported (public but unvalidated), the
// screening write is issued from PendingApproval, and an "approved" audit entry is
// appended by the admin with no rejection reason.
func TestApproveProblemPublishesAndAudits(t *testing.T) {
	repo := &fakeMutRepo{problem: pendingProblem()}
	audit := &fakeAudit{}
	notify := &fakeNotifier{}
	admins := fakeAdmins{union: map[string]string{"admin-1": "union-1"}}
	uc := application.NewApproveProblem(repo, newFakeAreas(), admins, audit, notify, domain.NewScreening())

	out, err := uc.Execute(context.Background(), "prob-1", "admin-1")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if out.Status != domain.StatusReported {
		t.Errorf("status = %q, want Reported", out.Status)
	}
	if out.ReviewedBy != "admin-1" {
		t.Errorf("ReviewedBy = %q, want admin-1", out.ReviewedBy)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "approved" || audit.entries[0].Actor != "admin-1" {
		t.Errorf("audit = %+v, want one 'approved' entry by admin-1", audit.entries)
	}

	// B21: the reporter is told, and told as an ACCOUNT. The type is asserted as its
	// literal rather than its constant, because that string is also the DB CHECK
	// value, the JSON the frontend switches on, and the bn.json key suffix — a
	// renamed constant that silently changed it would break all three.
	if len(notify.sent) != 1 {
		t.Fatalf("notifications = %d, want exactly 1", len(notify.sent))
	}
	n := notify.sent[0]
	if n.RecipientID != "reporter-1" || n.RecipientKind != notificationdomain.KindAccount {
		t.Errorf("recipient = %s/%s, want reporter-1/account", n.RecipientKind, n.RecipientID)
	}
	if string(n.Type) != "problem_approved" {
		t.Errorf("type = %q, want problem_approved", n.Type)
	}
	if n.ProblemID != "prob-1" || n.ProblemTitle != "broken culvert" {
		t.Errorf("problem = %q/%q, want prob-1 with its title snapshotted", n.ProblemID, n.ProblemTitle)
	}
}

// TestApproveProblemRefusesForeignUnionAdmin pins the union scoping: an admin of a
// different union cannot approve this report, so one admin can never release
// another union's reports.
func TestApproveProblemRefusesForeignUnionAdmin(t *testing.T) {
	repo := &fakeMutRepo{problem: pendingProblem()}
	notify := &fakeNotifier{}
	admins := fakeAdmins{union: map[string]string{"admin-2": "union-2"}}
	uc := application.NewApproveProblem(repo, newFakeAreas(), admins, &fakeAudit{}, notify, domain.NewScreening())

	_, err := uc.Execute(context.Background(), "prob-1", "admin-2")
	if !errors.Is(err, domain.ErrNotUnionAdmin) {
		t.Fatalf("err = %v, want ErrNotUnionAdmin", err)
	}
	// A REFUSED path tells nobody. Asserting the count — not merely that the error
	// came back — is what catches a notify line written above the guard instead of
	// below it, which would announce a publication that never happened.
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 on a refused approval", len(notify.sent))
	}
}

// TestApproveProblemRefusesNonPending pins that approve only publishes a report
// awaiting screening: an already-public report answers ErrNotPending, so a
// double-approve is a no-op rather than a silent status rewrite.
func TestApproveProblemRefusesNonPending(t *testing.T) {
	already := pendingProblem()
	already.Status = domain.StatusReported
	repo := &fakeMutRepo{problem: already}
	notify := &fakeNotifier{}
	admins := fakeAdmins{union: map[string]string{"admin-1": "union-1"}}
	uc := application.NewApproveProblem(repo, newFakeAreas(), admins, &fakeAudit{}, notify, domain.NewScreening())

	_, err := uc.Execute(context.Background(), "prob-1", "admin-1")
	if !errors.Is(err, domain.ErrNotPending) {
		t.Fatalf("err = %v, want ErrNotPending", err)
	}
	// The double-approve case, and the reason 000023 needs no unique index: this
	// event cannot repeat because the transition guard already stops it, so a
	// second "your report was published" is impossible without a second publication.
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 on a double approve", len(notify.sent))
	}
}
