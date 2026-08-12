package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
)

// fakeAssignments is the problem context's Assignments port. It records whether
// it was consulted at all, which is what proves the visibility gate runs first.
type fakeAssignments struct {
	officialID     string
	overrideReason string
	err            error
	called         bool
}

func (f *fakeAssignments) For(_ context.Context, _ string) (application.AssignmentView, error) {
	f.called = true
	return application.AssignmentView{OfficialID: f.officialID, OverrideReason: f.overrideReason}, f.err
}

func (f *fakeAssignments) ForMany(_ context.Context, ids []string) (map[string]application.AssignmentView, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]application.AssignmentView, len(ids))
	for _, id := range ids {
		out[id] = application.AssignmentView{OfficialID: f.officialID, OverrideReason: f.overrideReason}
	}
	return out, nil
}

// newGetProblem wires GetProblem over a repo holding the given problem plus the
// shared area tree and admin map, with nothing assigned.
func newGetProblem(p *domain.Problem, admins fakeAdmins) *application.GetProblem {
	return newGetProblemWith(p, admins, &fakeAssignments{})
}

// newGetProblemWith is newGetProblem with a caller-supplied Assignments port.
func newGetProblemWith(p *domain.Problem, admins fakeAdmins, asgn *fakeAssignments) *application.GetProblem {
	return application.NewGetProblem(&fakeMutRepo{problem: p}, newFakeAreas(), admins, asgn, &fakeAudit{})
}

// TestGetProblemPublicIsReadableByAnyone pins that an approved (public) report is
// returned to an anonymous caller — the gate only withholds pending reports.
func TestGetProblemPublicIsReadableByAnyone(t *testing.T) {
	p := pendingProblem()
	p.Status = domain.StatusReported
	uc := newGetProblem(p, fakeAdmins{})

	got, _, _, err := uc.Execute(context.Background(), "prob-1", application.Caller{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got.ID != "prob-1" {
		t.Errorf("got %q, want prob-1", got.ID)
	}
}

// TestGetProblemPendingHiddenFromStranger is the no-oracle rule: a pending report
// answers ErrProblemNotFound to an anonymous caller. The refusal is not-found, not
// forbidden, so a stranger cannot confirm a hidden report exists at this id.
func TestGetProblemPendingHiddenFromStranger(t *testing.T) {
	uc := newGetProblem(pendingProblem(), fakeAdmins{})

	_, _, _, err := uc.Execute(context.Background(), "prob-1", application.Caller{})
	if !errors.Is(err, domain.ErrProblemNotFound) {
		t.Fatalf("err = %v, want ErrProblemNotFound", err)
	}
}

// TestGetProblemPendingVisibleToReporter pins that the person who filed a report
// can always see it, even before it is approved — otherwise a report would be
// visible nowhere to its own author.
func TestGetProblemPendingVisibleToReporter(t *testing.T) {
	uc := newGetProblem(pendingProblem(), fakeAdmins{})

	got, _, _, err := uc.Execute(context.Background(), "prob-1", application.Caller{AccountID: "reporter-1"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got.ID != "prob-1" {
		t.Errorf("got %q, want prob-1", got.ID)
	}
}

// TestGetProblemPendingVisibleToOwnUnionAdmin pins that the admin who must screen
// the report can read it — scoped to their own union.
func TestGetProblemPendingVisibleToOwnUnionAdmin(t *testing.T) {
	admins := fakeAdmins{union: map[string]string{"admin-1": "union-1"}}
	uc := newGetProblem(pendingProblem(), admins)

	got, _, _, err := uc.Execute(context.Background(), "prob-1", application.Caller{AccountID: "admin-1", Role: application.RoleAdmin})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got.ID != "prob-1" {
		t.Errorf("got %q, want prob-1", got.ID)
	}
}

// TestGetProblemPendingHiddenFromOtherUnionAdmin pins that an admin of a different
// union gets the same not-found a stranger does — screening authority is per union,
// and it must not become a seat-wide oracle over what is awaiting review.
func TestGetProblemPendingHiddenFromOtherUnionAdmin(t *testing.T) {
	admins := fakeAdmins{union: map[string]string{"admin-2": "union-2"}}
	uc := newGetProblem(pendingProblem(), admins)

	_, _, _, err := uc.Execute(context.Background(), "prob-1", application.Caller{AccountID: "admin-2", Role: application.RoleAdmin})
	if !errors.Is(err, domain.ErrProblemNotFound) {
		t.Fatalf("err = %v, want ErrProblemNotFound", err)
	}
}

// TestGetProblemReturnsAssignedOfficial pins that the official a problem was
// actually handed to comes back alongside it. Pointed and assigned are two
// different facts, and publishing the second is what makes an admin's override
// visible to the public rather than merely stored.
func TestGetProblemReturnsAssignedOfficial(t *testing.T) {
	p := pendingProblem()
	p.Status = domain.StatusAssigned
	uc := newGetProblemWith(p, fakeAdmins{}, &fakeAssignments{officialID: "off-chair-aranagar"})

	_, _, assigned, err := uc.Execute(context.Background(), "prob-1", application.Caller{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if assigned.OfficialID != "off-chair-aranagar" {
		t.Errorf("assigned = %q, want off-chair-aranagar", assigned.OfficialID)
	}
}

// TestGetProblemReturnsOverrideReason pins that an admin's public justification
// for passing over the public's nominee travels WITH the assignment. The gap
// between pointed and assigned is only accountable if the reason for it is
// readable; publishing the gap while withholding the reason is worse than
// publishing neither, because it invites the reader to assume the worst.
func TestGetProblemReturnsOverrideReason(t *testing.T) {
	p := pendingProblem()
	p.Status = domain.StatusAssigned
	uc := newGetProblemWith(p, fakeAdmins{}, &fakeAssignments{
		officialID:     "off-chair-aranagar",
		overrideReason: "এই কাজটি ইউনিয়ন পরিষদের এখতিয়ারভুক্ত",
	})

	_, _, assigned, err := uc.Execute(context.Background(), "prob-1", application.Caller{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if assigned.OverrideReason != "এই কাজটি ইউনিয়ন পরিষদের এখতিয়ারভুক্ত" {
		t.Errorf("overrideReason = %q, want the admin's reason", assigned.OverrideReason)
	}
}

// TestGetProblemUnassignedIsNotAnError pins that "nobody yet" is the normal
// answer. Most problems have never been assigned, so the adapter swallows the
// not-found sentinel into an empty string — a report awaiting an admin must not
// fail to load because no official is on it.
func TestGetProblemUnassignedIsNotAnError(t *testing.T) {
	p := pendingProblem()
	p.Status = domain.StatusReported
	uc := newGetProblemWith(p, fakeAdmins{}, &fakeAssignments{officialID: ""})

	got, _, assigned, err := uc.Execute(context.Background(), "prob-1", application.Caller{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if assigned.OfficialID != "" {
		t.Errorf("assigned = %q, want empty", assigned.OfficialID)
	}
	if got.ID != "prob-1" {
		t.Errorf("got %q, want prob-1", got.ID)
	}
}

// TestGetProblemAssignmentErrorSurfaces pins that a genuine port failure is
// reported rather than quietly flattened into "unassigned" — which would publish
// a report as nobody's responsibility whenever the lookup broke.
func TestGetProblemAssignmentErrorSurfaces(t *testing.T) {
	p := pendingProblem()
	p.Status = domain.StatusAssigned
	boom := errors.New("assignment lookup failed")
	uc := newGetProblemWith(p, fakeAdmins{}, &fakeAssignments{err: boom})

	if _, _, _, err := uc.Execute(context.Background(), "prob-1", application.Caller{}); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the port's error", err)
	}
}

// TestGetProblemPendingNeverReadsAssignment pins the ORDER, not just the outcome:
// a hidden report must cost nothing on its way to a 404, so the visibility gate
// runs before anything else is fetched. Asserting only on the returned error would
// pass even if the assignment were read first and then thrown away.
func TestGetProblemPendingNeverReadsAssignment(t *testing.T) {
	asgn := &fakeAssignments{officialID: "off-chair-aranagar"}
	uc := newGetProblemWith(pendingProblem(), fakeAdmins{}, asgn)

	if _, _, _, err := uc.Execute(context.Background(), "prob-1", application.Caller{}); !errors.Is(err, domain.ErrProblemNotFound) {
		t.Fatalf("err = %v, want ErrProblemNotFound", err)
	}
	if asgn.called {
		t.Error("assignment port was consulted for a report the caller may not see")
	}
}
