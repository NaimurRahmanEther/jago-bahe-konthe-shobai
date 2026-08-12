package application_test

import (
	"context"
	"testing"
	"time"

	"jago-bahe-backend/internal/assignment/application"
	"jago-bahe-backend/internal/assignment/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// byType indexes a recorder's notifications by their type literal, so an assertion
// names the string the DB, the JSON and bn.json all agree on.
func byType(sent []notificationdomain.Notification) map[string]notificationdomain.Notification {
	out := make(map[string]notificationdomain.Notification, len(sent))
	for _, n := range sent {
		out[string(n.Type)] = n
	}
	return out
}

// assertAssignmentNotifications is the shared expectation of BOTH assignment
// routes. `assigner.create` is the one place either produces an assignment, so a
// party told on one path can never be silently missed on the other — and this
// helper is what makes that claim testable rather than merely commented.
func assertAssignmentNotifications(t *testing.T, sent []notificationdomain.Notification, wantOfficialID, wantOfficialName string) {
	t.Helper()

	if len(sent) != 2 {
		t.Fatalf("notifications = %d, want 2 (the reporter and the official)", len(sent))
	}
	got := byType(sent)

	// The reporter — an ACCOUNT id. The official's NAME rides along, because "your
	// report was assigned" is only useful if it says to whom.
	reporter, ok := got["problem_assigned"]
	if !ok {
		t.Fatalf("no problem_assigned notification; got %v", sent)
	}
	if reporter.RecipientKind != notificationdomain.KindAccount || reporter.RecipientID != "reporter-1" {
		t.Errorf("reporter notified as %s/%s, want account/reporter-1", reporter.RecipientKind, reporter.RecipientID)
	}
	if reporter.Detail != wantOfficialName {
		t.Errorf("problem_assigned detail = %q, want the official's name %q", reporter.Detail, wantOfficialName)
	}

	// The official — a DIRECTORY OFFICE id, never an account. The two id spaces
	// crossing here is the mistake the ForResident/ForOfficial split exists to make
	// impossible, and this is where it would show.
	official, ok := got["case_assigned"]
	if !ok {
		t.Fatalf("no case_assigned notification; got %v", sent)
	}
	if official.RecipientKind != notificationdomain.KindOfficial || official.RecipientID != wantOfficialID {
		t.Errorf("official notified as %s/%s, want official/%s", official.RecipientKind, official.RecipientID, wantOfficialID)
	}
	if _, err := time.Parse(time.RFC3339, official.Detail); err != nil {
		t.Errorf("case_assigned detail = %q, want an RFC3339 deadline: %v", official.Detail, err)
	}
}

// TestAssignWithinUnion_NotifiesBothParties covers the union route.
func TestAssignWithinUnion_NotifiesBothParties(t *testing.T) {
	notify := &fakeNotifier{}
	uc := application.NewAssignWithinUnion(newFakeRepo(), &fakeProblems{view: unionProblem()}, unionOfficials(),
		&fakeAdmins{unionOf: problemUnion}, fakeAreas{}, domain.NewService(), &fakeAudit{}, notify, time.Hour)

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-union", "", nil, ""); err != nil {
		t.Fatalf("assign: %v", err)
	}
	assertAssignmentNotifications(t, notify.sent, "off-union", "করিম উদ্দিন")
}

// TestAssignWithinUnion_OverrideNotifiesTheChosenOfficialNotThePointedOne pins that
// the notification follows the DECISION, not the request: an override tells the
// reporter who actually got it. The admin's public reason deliberately does NOT
// travel — it belongs on the audit trail, where accountability for passing over the
// public's choice lives (A.3.5).
func TestAssignWithinUnion_OverrideNotifiesTheChosenOfficialNotThePointedOne(t *testing.T) {
	notify := &fakeNotifier{}
	uc := application.NewAssignWithinUnion(newFakeRepo(), &fakeProblems{view: unionProblem()}, unionOfficials(),
		&fakeAdmins{unionOf: problemUnion}, fakeAreas{}, domain.NewService(), &fakeAudit{}, notify, time.Hour)

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-1", "off-other", "", nil, "the ward member owns this drain"); err != nil {
		t.Fatalf("override assign: %v", err)
	}
	assertAssignmentNotifications(t, notify.sent, "off-other", "রহিম মিয়া")

	for _, n := range notify.sent {
		if n.Detail == "the ward member owns this drain" {
			t.Error("the admin's override reason leaked into a notification; it belongs on the audit trail")
		}
	}
}

// TestAssignWithinUnion_RefusedAssignmentNotifiesNobody is the negative half. A
// report the admin has no standing over must produce no message at all — a
// notification is not a draft, and "your report was assigned" arriving after a
// refusal is worse than silence.
func TestAssignWithinUnion_RefusedAssignmentNotifiesNobody(t *testing.T) {
	notify := &fakeNotifier{}
	uc := application.NewAssignWithinUnion(newFakeRepo(), &fakeProblems{view: unionProblem()}, unionOfficials(),
		&fakeAdmins{unionOf: "union-2"}, fakeAreas{}, domain.NewService(), &fakeAudit{}, notify, time.Hour)

	if _, err := uc.Execute(context.Background(), "prob-1", "admin-2", "off-union", "", nil, ""); err == nil {
		t.Fatal("expected a refusal for an admin outside the problem's union")
	}
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 on a refused assignment", len(notify.sent))
	}
}

// TestForwardProblem_NotifiesBothParties covers the SUPER ADMIN route through the
// same assertions. That both routes satisfy one helper is the point: B20 reused
// `assigner` so the monitor, deadline and priority are set identically, and B21
// puts the notifications in the same place for the same reason.
func TestForwardProblem_NotifiesBothParties(t *testing.T) {
	notify := &fakeNotifier{}
	uc := newForwardNotifying(newFakeRepo(), &fakeProblems{view: aboveProblem()}, &fakeAudit{}, notify)

	if _, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-mp", "", nil, ""); err != nil {
		t.Fatalf("forward: %v", err)
	}
	assertAssignmentNotifications(t, notify.sent, "off-mp", "শহীদুজ্জামান সরকার")
}

// TestForwardProblem_RefusedForwardNotifiesNobody: a union-level report is not the
// super admin's to forward, and a refusal must stay silent.
func TestForwardProblem_RefusedForwardNotifiesNobody(t *testing.T) {
	notify := &fakeNotifier{}
	uc := newForwardNotifying(newFakeRepo(), &fakeProblems{view: unionProblem()}, &fakeAudit{}, notify)

	if _, err := uc.Execute(context.Background(), "prob-1", "super-1", "off-mp", "", nil, ""); err == nil {
		t.Fatal("expected ErrWrongRoute for a union-level report")
	}
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 on a refused forward", len(notify.sent))
	}
}
