package application_test

import (
	"context"
	"testing"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// only returns the single notification a use case sent, failing loudly on any
// other count. Most of these paths send exactly one or exactly none, and the
// distinction is the whole assertion.
func only(t *testing.T, notify *fakeNotifier) notificationdomain.Notification {
	t.Helper()
	if len(notify.sent) != 1 {
		t.Fatalf("notifications = %d, want exactly 1: %v", len(notify.sent), notify.sent)
	}
	return notify.sent[0]
}

// --- mark done: the highest-value notification in the platform ---

// TestMarkDone_AsksTheReporterToConfirm. The reporting resident is the ONLY party
// who can move a Done case to Resolved — an official may never close their own —
// so a loop that ends here ends because nobody told them they were being asked.
func TestMarkDone_AsksTheReporterToConfirm(t *testing.T) {
	repo := newFakeRepo()
	c := doneCase()
	c.Status = domain.StatusInProgress
	c.Evidence = []domain.Evidence{{ID: "ev-1", CaseID: "case-1", BeforeImageURL: "b", AfterImageURL: "a"}}
	repo.put(c)
	notify := &fakeNotifier{}
	uc := application.NewMarkDone(repo, &fakeProblems{reporter: "res-1"}, &fakeAudit{}, notify)

	if _, err := uc.Execute(context.Background(), "case-1", "off-1"); err != nil {
		t.Fatalf("mark done: %v", err)
	}

	n := only(t, notify)
	if n.RecipientKind != notificationdomain.KindAccount || n.RecipientID != "res-1" {
		t.Errorf("recipient = %s/%s, want account/res-1 (the reporter, not the official)", n.RecipientKind, n.RecipientID)
	}
	if string(n.Type) != "confirmation_requested" {
		t.Errorf("type = %q, want confirmation_requested", n.Type)
	}
	if n.ProblemTitle != "title of prob-1" {
		t.Errorf("title = %q, want the snapshotted problem title", n.ProblemTitle)
	}
}

// TestMarkDone_WithoutEvidenceNotifiesNobody. Done is evidence-gated, and the
// refusal must be silent: a "please confirm" for a fix with no before/after photo
// would ask a resident to vouch for something the platform never published.
func TestMarkDone_WithoutEvidenceNotifiesNobody(t *testing.T) {
	repo := newFakeRepo()
	c := doneCase()
	c.Status = domain.StatusInProgress // no evidence attached
	repo.put(c)
	notify := &fakeNotifier{}
	uc := application.NewMarkDone(repo, &fakeProblems{reporter: "res-1"}, &fakeAudit{}, notify)

	if _, err := uc.Execute(context.Background(), "case-1", "off-1"); err == nil {
		t.Fatal("expected ErrEvidenceRequired")
	}
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 when Done was refused", len(notify.sent))
	}
}

// --- confirm: only the reopen is notified ---

// TestConfirmResolution_ReopenNotifiesTheAssignee. A reopen is a demand for more
// work on a case the official believed finished, and they have no surface that says
// so. The recipient is a DIRECTORY OFFICE id, not an account.
func TestConfirmResolution_ReopenNotifiesTheAssignee(t *testing.T) {
	repo := newFakeRepo()
	repo.put(doneCase())
	notify := &fakeNotifier{}
	uc := application.NewConfirmResolution(repo, &fakeProblems{reporter: "res-1"}, &fakeAudit{}, notify)

	if _, err := uc.Execute(context.Background(), "prob-1", "res-1", domain.ConfirmationNotSolved); err != nil {
		t.Fatalf("confirm not solved: %v", err)
	}

	n := only(t, notify)
	if n.RecipientKind != notificationdomain.KindOfficial || n.RecipientID != "off-1" {
		t.Errorf("recipient = %s/%s, want official/off-1", n.RecipientKind, n.RecipientID)
	}
	if string(n.Type) != "case_reopened" {
		t.Errorf("type = %q, want case_reopened", n.Type)
	}
}

// TestConfirmResolution_SolvedNotifiesNobody pins a DECIDED asymmetry rather than
// an oversight: a `solved` confirmation is good news the official reads on their own
// dashboard, while the reopen is the one the loop stalls without. If a later phase
// wants to notify on success too, change this test deliberately — do not assume it
// was forgotten.
func TestConfirmResolution_SolvedNotifiesNobody(t *testing.T) {
	repo := newFakeRepo()
	repo.put(doneCase())
	notify := &fakeNotifier{}
	uc := application.NewConfirmResolution(repo, &fakeProblems{reporter: "res-1"}, &fakeAudit{}, notify)

	if _, err := uc.Execute(context.Background(), "prob-1", "res-1", domain.ConfirmationSolved); err != nil {
		t.Fatalf("confirm solved: %v", err)
	}
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 on a solved confirmation (decided, not overlooked)", len(notify.sent))
	}
}

// TestConfirmResolution_NonReporterNotifiesNobody. Only the reporting resident may
// confirm; a stranger's attempt must not reach the official.
func TestConfirmResolution_NonReporterNotifiesNobody(t *testing.T) {
	repo := newFakeRepo()
	repo.put(doneCase())
	notify := &fakeNotifier{}
	uc := application.NewConfirmResolution(repo, &fakeProblems{reporter: "res-1"}, &fakeAudit{}, notify)

	if _, err := uc.Execute(context.Background(), "prob-1", "someone-else", domain.ConfirmationNotSolved); err == nil {
		t.Fatal("expected ErrNotReporter")
	}
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 when the confirmer was not the reporter", len(notify.sent))
	}
}

// --- obstacle: the monitor, not the named authority ---

func plannedCaseWithMonitor(monitor string) *domain.Case {
	return &domain.Case{
		ID: "case-1", ProblemID: "prob-1", OfficialID: "off-1",
		MonitorOfficialID: monitor, Status: domain.StatusInProgress,
	}
}

// TestReportObstacle_NotifiesTheMonitorWithWhoUnblocks is the documented DEVIATION
// from the one notification the design docs promise outright.
//
// Concept §8, Scaffold §2 and Backend Plan B5 all say the platform "notifies the
// named higher authority". Obstacle.WhoUnblocks is FREE TEXT, so that party is not
// addressable — there is no id to write a row to. The monitor is, so the monitor is
// told, and whoUnblocks travels as Detail. See CLAUDE.md A.3.9 constraint 5.
func TestReportObstacle_NotifiesTheMonitorWithWhoUnblocks(t *testing.T) {
	repo := newFakeRepo()
	repo.put(plannedCaseWithMonitor("off-upazila"))
	notify := &fakeNotifier{}
	uc := application.NewReportObstacle(repo, &fakeProblems{}, &fakeAudit{}, notify)

	_, err := uc.Execute(context.Background(), "case-1", "off-1",
		domain.CategoryBudget, "no allocation this quarter", "the upazila engineer", "wrote twice in June")
	if err != nil {
		t.Fatalf("report obstacle: %v", err)
	}

	n := only(t, notify)
	if n.RecipientKind != notificationdomain.KindOfficial || n.RecipientID != "off-upazila" {
		t.Errorf("recipient = %s/%s, want official/off-upazila (the MONITOR, not the assignee)", n.RecipientKind, n.RecipientID)
	}
	if string(n.Type) != "obstacle_declared" {
		t.Errorf("type = %q, want obstacle_declared", n.Type)
	}
	// The free-text authority the official named rides along as context, which is
	// the honest half of the promise the platform cannot address directly.
	if n.Detail != "the upazila engineer" {
		t.Errorf("detail = %q, want the free-text whoUnblocks", n.Detail)
	}
}

// TestReportObstacle_TopOfLadderNotifiesNobody is the guard that stops a private
// message reaching everyone.
//
// MonitorFor returns "" when the assignee is at the top of the ladder — the MP has
// nobody above them — and the read query matches the caller's account id AND
// official id, the latter being "" for every resident, admin and super admin. One
// blank-recipient row would therefore be delivered to every non-official in the
// seat. Writing nothing is the correct outcome, and the case still succeeds: having
// no monitor is a fact about the ladder, not a failure to declare an obstacle.
func TestReportObstacle_TopOfLadderNotifiesNobody(t *testing.T) {
	repo := newFakeRepo()
	repo.put(plannedCaseWithMonitor("")) // the MP: nobody above
	notify := &fakeNotifier{}
	uc := application.NewReportObstacle(repo, &fakeProblems{}, &fakeAudit{}, notify)

	c, err := uc.Execute(context.Background(), "case-1", "off-1",
		domain.CategoryBudget, "no allocation", "the ministry", "wrote twice")
	if err != nil {
		t.Fatalf("report obstacle must still succeed with no monitor: %v", err)
	}
	if c.Status != domain.StatusBlocked {
		t.Errorf("status = %s, want Blocked — the obstacle is recorded either way", c.Status)
	}
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 when the assignee has no monitor", len(notify.sent))
	}
}

// --- adjudication: one type, two opposite outcomes ---

func blockedCaseWithObstacle() *domain.Case {
	return &domain.Case{
		ID: "case-1", ProblemID: "prob-1", OfficialID: "off-1", Status: domain.StatusBlocked,
		Obstacles: []domain.Obstacle{{
			ID: "blk-1", CaseID: "case-1", Category: domain.CategoryBudget,
			WhatBlocks: "no allocation", WhoUnblocks: "the upazila engineer",
			Adjudication: domain.AdjudicationPending,
		}},
	}
}

// TestAdjudicateObstacle_NotifiesTheAssigneeWithTheVerdict. The two outcomes are
// opposite — confirm protects the official on the scorecard and moves
// responsibility up, deny bounces the work back with the clock restarted — so the
// verdict itself must travel in Detail rather than being inferred.
func TestAdjudicateObstacle_NotifiesTheAssigneeWithTheVerdict(t *testing.T) {
	for _, tc := range []struct {
		name       string
		confirm    bool
		wantDetail string
	}{
		{"confirmed", true, "confirmed"},
		{"denied", false, "denied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.put(blockedCaseWithObstacle())
			notify := &fakeNotifier{}
			uc := application.NewAdjudicateObstacle(repo, &fakeProblems{}, &fakeAudit{}, notify)

			if _, err := uc.Execute(context.Background(), "blk-1", "admin-1", tc.confirm); err != nil {
				t.Fatalf("adjudicate: %v", err)
			}

			n := only(t, notify)
			// The ASSIGNEE, never the adjudicator: the admin knows what they just
			// decided, and the official is the one who has to act on it.
			if n.RecipientKind != notificationdomain.KindOfficial || n.RecipientID != "off-1" {
				t.Errorf("recipient = %s/%s, want official/off-1", n.RecipientKind, n.RecipientID)
			}
			if string(n.Type) != "obstacle_adjudicated" {
				t.Errorf("type = %q, want obstacle_adjudicated", n.Type)
			}
			if n.Detail != tc.wantDetail {
				t.Errorf("detail = %q, want %q", n.Detail, tc.wantDetail)
			}
		})
	}
}

// TestAdjudicateObstacle_SecondVerdictNotifiesNobody. ErrAlreadyAdjudicated is the
// guard that makes this event single-shot, which is why 000023 needs no unique
// index on it.
func TestAdjudicateObstacle_SecondVerdictNotifiesNobody(t *testing.T) {
	repo := newFakeRepo()
	c := blockedCaseWithObstacle()
	c.Obstacles[0].Adjudication = domain.AdjudicationConfirmed
	repo.put(c)
	notify := &fakeNotifier{}
	uc := application.NewAdjudicateObstacle(repo, &fakeProblems{}, &fakeAudit{}, notify)

	if _, err := uc.Execute(context.Background(), "blk-1", "admin-1", false); err == nil {
		t.Fatal("expected ErrAlreadyAdjudicated")
	}
	if len(notify.sent) != 0 {
		t.Errorf("notifications = %d, want 0 on a second verdict", len(notify.sent))
	}
}
