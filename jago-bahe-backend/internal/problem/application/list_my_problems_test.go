package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// fakeListRepo records the Filter it was handed and replays a fixed set of rows.
//
// Capturing the Filter is the point, not a convenience: the own-view use case is
// only correct because a nil Statuses (every state) is paired with a non-empty
// ReporterID. A test that asserts on returned rows alone would pass even if the
// filter went out unscoped, because this fake — like a stub repo, unlike Postgres
// — has no other reporters' rows to wrongly return.
type fakeListRepo struct {
	rows []domain.Problem
	got  domain.Filter
	// calls counts List invocations, so a test can pin that a guard refused
	// *before* touching storage.
	calls int
}

func (r *fakeListRepo) List(_ context.Context, f domain.Filter) ([]domain.Problem, error) {
	r.calls++
	r.got = f
	return r.rows, nil
}

// The rest of the port is unused here; the own-view use case reads only List.
func (r *fakeListRepo) Create(context.Context, *domain.Problem) error { return nil }
func (r *fakeListRepo) GetByID(context.Context, string) (*domain.Problem, error) {
	return nil, domain.ErrProblemNotFound
}
func (r *fakeListRepo) AddVote(context.Context, domain.ValidationVote) error { return nil }
func (r *fakeListRepo) ListVotes(context.Context, string) ([]domain.ValidationVote, error) {
	return nil, nil
}
func (r *fakeListRepo) UpdateValidation(context.Context, string, int, domain.Status) error {
	return nil
}
func (r *fakeListRepo) SetStatus(context.Context, string, domain.Status) error { return nil }
func (r *fakeListRepo) Update(context.Context, *domain.Problem) error          { return nil }
func (r *fakeListRepo) SetWithdrawn(context.Context, string, domain.Status) error {
	return nil
}
func (r *fakeListRepo) SetScreening(context.Context, string, domain.Status, domain.Status, domain.RejectionReason, string, time.Time) error {
	return nil
}
func (r *fakeListRepo) Delete(context.Context, string) error { return nil }
func (r *fakeListRepo) VotesByViewer(context.Context, string, []string) (map[string]domain.VoteChoice, error) {
	return nil, nil
}
func (r *fakeListRepo) TitlesByIDs(context.Context, []string) (map[string]string, error) {
	return nil, nil
}

var _ domain.Repository = (*fakeListRepo)(nil)

func problemOf(id, reporterID string, status domain.Status) domain.Problem {
	return domain.Problem{
		ID:         id,
		Title:      "broken culvert",
		Location:   valueobject.Location{AreaID: valueobject.AreaID("ward-1")},
		ReporterID: reporterID,
		Status:     status,
	}
}

// TestListMyProblemsScopesToTheCaller pins the filter the use case builds. The
// pairing is the invariant: Statuses nil means "every state", which is correct
// *only* because ReporterID scopes the query to one person's own rows.
func TestListMyProblemsScopesToTheCaller(t *testing.T) {
	repo := &fakeListRepo{}
	uc := application.NewListMyProblems(repo)

	if _, err := uc.Execute(context.Background(), "acct-1"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if repo.got.ReporterID != "acct-1" {
		t.Errorf("Filter.ReporterID = %q, want acct-1 — an unscoped filter would list the whole seat", repo.got.ReporterID)
	}
	if repo.got.Statuses != nil {
		t.Errorf("Filter.Statuses = %v, want nil (every state — the reporter's list drops nothing)", repo.got.Statuses)
	}
	if repo.got.AreaID != "" || repo.got.OfficialID != "" {
		t.Errorf("Filter = %+v, want no area/official narrowing", repo.got)
	}
}

// TestListMyProblemsDropsNoStatus pins the nil-Statuses half of the filter: the
// reporter's own list spans every state, including a Rejected report the public
// feed also shows. The feed derives its own status set and this view must not
// inherit that habit — "everything I filed" is a different question from "what is
// on the public record", and a status filter here would quietly answer the wrong
// one.
func TestListMyProblemsDropsNoStatus(t *testing.T) {
	repo := &fakeListRepo{rows: []domain.Problem{
		problemOf("prob-1", "acct-1", domain.StatusRejected),
		problemOf("prob-2", "acct-1", domain.StatusReported),
		problemOf("prob-3", "acct-1", domain.StatusResolved),
	}}
	uc := application.NewListMyProblems(repo)

	out, err := uc.Execute(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("got %d problems, want 3 — every state the reporter filed", len(out))
	}
}

// TestListMyProblemsReturnsOwnRejectedWithReason pins that a rejection stays
// legible to the person it happened to: the ground travels with the row, so a
// wrongly-buried report has a witness (Scaffold §2 Moderation).
func TestListMyProblemsReturnsOwnRejectedWithReason(t *testing.T) {
	rejected := problemOf("prob-3", "acct-1", domain.StatusRejected)
	rejected.RejectionReason = domain.ReasonSpam
	repo := &fakeListRepo{rows: []domain.Problem{rejected}}

	out, err := application.NewListMyProblems(repo).Execute(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d problems, want 1", len(out))
	}
	if out[0].RejectionReason != domain.ReasonSpam {
		t.Errorf("RejectionReason = %q, want spam to travel with the row", out[0].RejectionReason)
	}
}

// TestListMyProblemsRefusesEmptyCaller is the guard that makes the nil-Statuses
// filter safe. An anonymous call must fail closed: with no reporter scope, the
// same filter matches every problem in the seat, and this endpoint would answer
// "everything anyone ever filed" to the question "what did I file". It must refuse
// *before* reaching storage, so the unscoped query is never issued at all.
func TestListMyProblemsRefusesEmptyCaller(t *testing.T) {
	repo := &fakeListRepo{rows: []domain.Problem{
		problemOf("prob-1", "someone-else", domain.StatusReported),
	}}

	out, err := application.NewListMyProblems(repo).Execute(context.Background(), "")
	if !errors.Is(err, domain.ErrNoCaller) {
		t.Fatalf("err = %v, want ErrNoCaller", err)
	}
	if out != nil {
		t.Errorf("out = %v, want nil — an anonymous caller must never receive rows", out)
	}
	if repo.calls != 0 {
		t.Errorf("List called %d times, want 0 — the guard must refuse before querying", repo.calls)
	}
}
