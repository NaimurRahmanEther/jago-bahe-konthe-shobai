package application_test

import (
	"context"
	"testing"
	"time"

	"jago-bahe-backend/internal/shared/audit/application"
	"jago-bahe-backend/internal/shared/audit/domain"
)

// fakeAudit records the actions it was asked for, so a test can assert on the
// QUERY rather than only on the rows. Asserting on rows alone would pass even if
// the use case asked for every action and the fake happened to return few.
type fakeAudit struct {
	askedActions []string
	askedLimit   int
	entries      []domain.AuditEntry
}

func (f *fakeAudit) Append(context.Context, domain.AuditEntry) error { return nil }
func (f *fakeAudit) ListByTarget(context.Context, string, string) ([]domain.AuditEntry, error) {
	return nil, nil
}
func (f *fakeAudit) DeleteByTarget(context.Context, string, string) error { return nil }
func (f *fakeAudit) ListByActions(_ context.Context, actions []string, limit int) ([]domain.AuditEntry, error) {
	f.askedActions = actions
	f.askedLimit = limit
	return f.entries, nil
}

var _ domain.Repository = (*fakeAudit)(nil)

// THE test for this feature. The public feed is a narrower slice of the same audit
// log the super admin reads, and the three actions it withholds are withheld for
// safety reasons, not tidiness:
//
//   - `verified` would build a public index of who is a verified resident of which
//     union — the person-graph A.3.2 rule 1 protects.
//   - `claim_rejected` would publish "this person claimed to be the MP and was
//     refused", an accusation about an unproven identity claim.
//
// Asserted BY NAME, not by count, so adding a sixth public action does not silently
// satisfy it.
func TestPublicActionsWithholdsPersonActions(t *testing.T) {
	public := map[string]bool{}
	for _, a := range application.PublicActions() {
		public[a] = true
	}

	for _, forbidden := range []string{"verified", "claim_approved", "claim_rejected"} {
		if public[forbidden] {
			t.Errorf("%q is about a PERSON and must never be in the public feed — see list_activity.go", forbidden)
		}
	}

	for _, required := range []string{"approved", "rejected", "assigned", "forwarding_suggested"} {
		if !public[required] {
			t.Errorf("%q is about a problem and belongs in the public feed, but is missing", required)
		}
	}
}

// Found by running the feed against real data rather than by reading the code: the
// action set has to match the strings the code WRITES, and two of them are not the
// obvious ones.
//
// `assignment_overridden` is written INSTEAD of `assigned` when an admin passes
// over the official the public asked for (assign_within_union.go:75-77). Missing it
// hid the single most accountability-relevant assignment event from the
// accountability feed.
//
// `adjudicated` is written by NOTHING — adjudicate_blocker.go writes
// `blocker_confirmed` / `blocker_denied` — which is why the super admin's oversight
// feed showed zero adjudications from B11 until this was found. A feed missing a row
// looks exactly like a seat where nobody acted, which is how it stayed hidden.
func TestPublicActionsUseTheStringsTheCodeActuallyWrites(t *testing.T) {
	public := map[string]bool{}
	for _, a := range application.PublicActions() {
		public[a] = true
	}

	if !public["assignment_overridden"] {
		t.Error("an admin overriding the public's choice must appear in the public record")
	}
	if !public["blocker_confirmed"] || !public["blocker_denied"] {
		t.Error("adjudications are written as blocker_confirmed/blocker_denied, not 'adjudicated'")
	}
	if public["adjudicated"] {
		t.Error(`"adjudicated" is written by nothing — asking for it silently returns no adjudications`)
	}
	// B20 deleted the admin vote, so these two are written by nothing now. Left in
	// the set they would be the `adjudicated` bug again: a feed with no vote rows
	// looks identical to a seat where nobody voted.
	if public["vote_opened"] || public["vote_no_majority"] || public["vote_cast"] {
		t.Error("the admin vote is deleted (A.3.8) — no vote action is written by anything any more")
	}
	if !public["forwarding_suggested"] {
		t.Error("forwarding advice is what replaced the vote and belongs in the public record")
	}
}

// The ballots-vs-tally rule REVERSED in B20, and the reason it reversed is the
// whole rule. `vote_cast` was withheld because an admin whose BINDING vote is
// published votes differently — secrecy is owed to a choice that settles something.
// `forwarding_suggested` settles nothing: the super admin decides and may forward
// against all of it. So the adviser is named, and that is what makes a forward
// against the seat's admins visible as such.
//
// If advice ever becomes binding again, the old rule comes back with it.
func TestPublicActionsNamesAdvisersBecauseAdviceDoesNotBind(t *testing.T) {
	public := map[string]bool{}
	for _, a := range application.PublicActions() {
		public[a] = true
	}
	if !public["forwarding_suggested"] {
		t.Error("advice is attributed and public — see A.3.8 rule 4")
	}
}

// `reported` is deliberately absent too, and for a different reason than the person
// actions: it is not a moderator decision at all. Including it would turn the
// accountability record into a second copy of the problem feed.
func TestPublicActionsExcludesNonModeratorActions(t *testing.T) {
	for _, a := range application.PublicActions() {
		if a == "reported" || a == "validated" {
			t.Errorf("%q is a resident's action, not a moderator decision", a)
		}
	}
}

func TestExecuteQueriesOnlyThePublicActions(t *testing.T) {
	repo := &fakeAudit{}
	uc := application.NewListActivity(repo)

	if _, err := uc.Execute(context.Background(), 25); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(repo.askedActions) != len(application.PublicActions()) {
		t.Fatalf("asked for %d actions, want the %d public ones: %v",
			len(repo.askedActions), len(application.PublicActions()), repo.askedActions)
	}
	for _, a := range repo.askedActions {
		if a == "verified" || a == "claim_approved" || a == "claim_rejected" {
			t.Fatalf("the query itself asked for %q — the split must hold at the QUERY, not be filtered afterwards", a)
		}
	}
	if repo.askedLimit != 25 {
		t.Errorf("limit = %d, want 25 passed through", repo.askedLimit)
	}
}

func TestExecuteReturnsWhatTheRepositoryGives(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeAudit{entries: []domain.AuditEntry{
		{ID: "audit-1", TargetType: "problem", TargetID: "prob-1", Actor: "acct-admin-1", Action: "approved", CreatedAt: now},
	}}

	got, err := application.NewListActivity(repo).Execute(context.Background(), 0)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(got) != 1 || got[0].Action != "approved" {
		t.Fatalf("got %+v, want the one approved entry", got)
	}
}
