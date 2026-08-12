package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// TestRejectProblemNotifiesTheGroundNotTheNote pins what travels in a rejection
// notification, which is the one decision in the eight where the payload is not
// obvious.
//
// The GROUND (a closed enum an admin was permitted to reject on) travels; the
// admin's free-text note does NOT. The frontend renders the four grounds in Bangla
// from problem.rejectionReason.*, and prose in a notification would be an
// untranslatable string on a Bangla-only surface. The note is on the public audit
// entry, which the reporter reaches from the report itself — a Rejected report
// stays publicly visible with its reason precisely so a wrongly-buried one keeps a
// witness.
func TestRejectProblemNotifiesTheGroundNotTheNote(t *testing.T) {
	repo := &fakeMutRepo{problem: pendingProblem()}
	audit := &fakeAudit{}
	notify := &fakeNotifier{}
	admins := fakeAdmins{union: map[string]string{"admin-1": "union-1"}}
	uc := application.NewRejectProblem(repo, newFakeAreas(), admins, audit, notify, domain.NewScreening())

	out, err := uc.Execute(context.Background(), "prob-1", "admin-1", domain.ReasonSpam, "posted the same thing four times")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if out.Status != domain.StatusRejected {
		t.Errorf("status = %q, want Rejected", out.Status)
	}

	if len(notify.sent) != 1 {
		t.Fatalf("notifications = %d, want exactly 1", len(notify.sent))
	}
	n := notify.sent[0]
	if n.RecipientID != "reporter-1" || n.RecipientKind != notificationdomain.KindAccount {
		t.Errorf("recipient = %s/%s, want reporter-1/account", n.RecipientKind, n.RecipientID)
	}
	if string(n.Type) != "problem_rejected" {
		t.Errorf("type = %q, want problem_rejected", n.Type)
	}
	if n.Detail != string(domain.ReasonSpam) {
		t.Errorf("detail = %q, want the enum ground %q", n.Detail, domain.ReasonSpam)
	}
	if n.Detail == "posted the same thing four times" {
		t.Error("the admin's free-text note leaked into the notification; only the ground travels")
	}
}

// TestRejectProblemRefusesForeignUnionAdminAndNotifiesNobody is the union-scope
// half, asserted the way the approve test asserts it: a refused takedown must tell
// the reporter nothing, or the platform announces a rejection that did not happen.
func TestRejectProblemRefusesForeignUnionAdminAndNotifiesNobody(t *testing.T) {
	repo := &fakeMutRepo{problem: pendingProblem()}
	notify := &fakeNotifier{}
	admins := fakeAdmins{union: map[string]string{"admin-2": "union-2"}}
	uc := application.NewRejectProblem(repo, newFakeAreas(), admins, &fakeAudit{}, notify, domain.NewScreening())

	_, err := uc.Execute(context.Background(), "prob-1", "admin-2", domain.ReasonSpam, "")
	if !errors.Is(err, domain.ErrNotUnionAdmin) {
		t.Fatalf("err = %v, want ErrNotUnionAdmin", err)
	}
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 on a refused rejection", len(notify.sent))
	}
}
