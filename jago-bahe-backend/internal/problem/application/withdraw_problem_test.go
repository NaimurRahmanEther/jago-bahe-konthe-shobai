package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
)

// TestWithdrawProblemSoftDeletesAndAudits pins the happy path: the reporter's own
// still-withdrawable report becomes Withdrawn, the compare-and-set is issued from
// the loaded status, and a "withdrawn" audit entry carries the reporter's note.
func TestWithdrawProblemSoftDeletesAndAudits(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem()}
	audit := &fakeAudit{}
	uc := application.NewWithdrawProblem(repo, audit)

	out, err := uc.Execute(context.Background(), "prob-1", "acct-1", "filed by mistake")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if out.Status != domain.StatusWithdrawn {
		t.Errorf("status = %q, want Withdrawn", out.Status)
	}
	if repo.withdrawCalls != 1 || repo.withdrawn.from != domain.StatusReported {
		t.Errorf("SetWithdrawn called %d times from %q, want 1 from Reported", repo.withdrawCalls, repo.withdrawn.from)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "withdrawn" ||
		audit.entries[0].Actor != "acct-1" || audit.entries[0].Reason != "filed by mistake" {
		t.Errorf("audit = %+v, want one 'withdrawn' entry by acct-1 with the note", audit.entries)
	}
}

// TestWithdrawProblemRefusesNonReporter pins that ownership is checked before the
// window, so a stranger cannot withdraw someone else's report and never learns its
// state.
func TestWithdrawProblemRefusesNonReporter(t *testing.T) {
	repo := &fakeMutRepo{problem: editableProblem()}
	_, err := application.NewWithdrawProblem(repo, &fakeAudit{}).Execute(
		context.Background(), "prob-1", "acct-2", "")
	if !errors.Is(err, domain.ErrNotReporter) {
		t.Fatalf("err = %v, want ErrNotReporter", err)
	}
	if repo.withdrawCalls != 0 {
		t.Errorf("SetWithdrawn called %d times, want 0", repo.withdrawCalls)
	}
}

// TestWithdrawProblemRefusesPastWindow pins that once a problem is assigned a case
// exists and an official is working in public, so the reporter can no longer
// withdraw it out from under them — the use case returns ErrNotWithdrawable.
func TestWithdrawProblemRefusesPastWindow(t *testing.T) {
	assigned := editableProblem()
	assigned.Status = domain.StatusAssigned
	repo := &fakeMutRepo{problem: assigned}
	_, err := application.NewWithdrawProblem(repo, &fakeAudit{}).Execute(
		context.Background(), "prob-1", "acct-1", "")
	if !errors.Is(err, domain.ErrNotWithdrawable) {
		t.Fatalf("err = %v, want ErrNotWithdrawable", err)
	}
	if repo.withdrawCalls != 0 {
		t.Errorf("SetWithdrawn called %d times, want 0 — an assigned problem must never be withdrawn", repo.withdrawCalls)
	}
}
