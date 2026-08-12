package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
)

// These reuse fakeMutRepo and fakeAudit from update_problem_test.go — the same
// reporter self-management fakes the edit and withdraw tests are built on.

// TestDeleteProblemErasesRowAndAuditTrail pins the happy path, including the half
// that is easy to forget: the audit entries are erased too, and with the right
// target. A delete that left the trail behind would leave dangling entries, since
// audit_entries has no FK to problems.
func TestDeleteProblemErasesRowAndAuditTrail(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem()}
	audit := &fakeAudit{}
	uc := application.NewDeleteProblem(repo, audit)

	if err := uc.Execute(context.Background(), "prob-1", "acct-1"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if repo.deleteCalls != 1 || repo.deletedID != "prob-1" {
		t.Errorf("Delete calls = %d, id = %q; want 1, %q", repo.deleteCalls, repo.deletedID, "prob-1")
	}
	if audit.deleteCalls != 1 {
		t.Fatalf("audit DeleteByTarget called %d times, want 1 — the trail must not outlive the row", audit.deleteCalls)
	}
	if audit.deletedTarget.targetType != "problem" || audit.deletedTarget.targetID != "prob-1" {
		t.Errorf("audit target = %+v, want {problem prob-1}", audit.deletedTarget)
	}
	// Nothing is appended: this path erases the record rather than recording the
	// erasure. Pinned so a later "let's at least log it" change is a deliberate one.
	if len(audit.entries) != 0 {
		t.Errorf("audit entries = %+v, want none appended", audit.entries)
	}
}

// TestDeleteProblemRefusesNonReporter pins that ownership is checked before
// anything is written. Asserting on the call counts, not just the error, is the
// point: a delete that returned the error after erasing the row would still pass
// an error-only assertion.
func TestDeleteProblemRefusesNonReporter(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem()}
	audit := &fakeAudit{}

	err := application.NewDeleteProblem(repo, audit).Execute(context.Background(), "prob-1", "acct-2")
	if !errors.Is(err, domain.ErrNotReporter) {
		t.Fatalf("err = %v, want ErrNotReporter", err)
	}
	if repo.deleteCalls != 0 {
		t.Errorf("Delete called %d times, want 0 — a stranger must never erase a report", repo.deleteCalls)
	}
	if audit.deleteCalls != 0 {
		t.Errorf("audit DeleteByTarget called %d times, want 0", audit.deleteCalls)
	}
}

// TestDeleteProblemRefusesEmptyCaller pins that an absent caller is a non-reporter,
// never an accidental owner — the same rule the edit and withdraw paths hold.
func TestDeleteProblemRefusesEmptyCaller(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem()}
	audit := &fakeAudit{}

	err := application.NewDeleteProblem(repo, audit).Execute(context.Background(), "prob-1", "")
	if !errors.Is(err, domain.ErrNotReporter) {
		t.Fatalf("err = %v, want ErrNotReporter", err)
	}
	if repo.deleteCalls != 0 || audit.deleteCalls != 0 {
		t.Errorf("delete reached storage on an empty caller (repo %d, audit %d)", repo.deleteCalls, audit.deleteCalls)
	}
}

// TestDeleteProblemUnknownID pins that a missing problem surfaces as not-found from
// the load, before ownership is even considered.
func TestDeleteProblemUnknownID(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem(), getErr: domain.ErrProblemNotFound}
	audit := &fakeAudit{}

	err := application.NewDeleteProblem(repo, audit).Execute(context.Background(), "nope", "acct-1")
	if !errors.Is(err, domain.ErrProblemNotFound) {
		t.Fatalf("err = %v, want ErrProblemNotFound", err)
	}
	if repo.deleteCalls != 0 || audit.deleteCalls != 0 {
		t.Errorf("delete attempted for an unknown problem (repo %d, audit %d)", repo.deleteCalls, audit.deleteCalls)
	}
}

// TestDeleteProblemHasNoStatusWindow is the test that will look wrong to a
// reviewer, so read this before "fixing" it.
//
// Every other reporter power is bounded by a window — edit closes once a report is
// endorsed, withdraw closes once it is assigned — precisely so a reporter cannot
// erase work that other people have done in public. Delete is deliberately NOT
// bounded: it succeeds on a Resolved problem, taking the official's plan, updates
// and evidence with it via the cascade, and dropping that case from the official's
// scorecard. That is the decision recorded in CLAUDE.md A.3.3, not an oversight.
//
// If a status guard is ever added, this test is where the change must be argued.
func TestDeleteProblemHasNoStatusWindow(t *testing.T) {
	for _, status := range []domain.Status{
		domain.StatusPendingApproval,
		domain.StatusReported,
		domain.StatusValidated,
		domain.StatusAssigned,
		domain.StatusInProgress,
		domain.StatusBlocked,
		domain.StatusDone,
		domain.StatusResolved,
	} {
		t.Run(string(status), func(t *testing.T) {
			p := editableProblem()
			p.Status = status
			repo := &fakeMutRepo{problem: p}
			audit := &fakeAudit{}

			if err := application.NewDeleteProblem(repo, audit).Execute(context.Background(), "prob-1", "acct-1"); err != nil {
				t.Fatalf("execute at %s: %v", status, err)
			}
			if repo.deleteCalls != 1 || audit.deleteCalls != 1 {
				t.Errorf("at %s: repo deletes = %d, audit deletes = %d; want 1 and 1", status, repo.deleteCalls, audit.deleteCalls)
			}
		})
	}
}
