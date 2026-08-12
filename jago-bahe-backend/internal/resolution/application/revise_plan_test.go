package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
)

// stalledCase is a case sitting in a status a replan is legal from.
func stalledCase(officialID string, status domain.Status) *domain.Case {
	return &domain.Case{
		ID:         "case-1",
		ProblemID:  "prob-1",
		OfficialID: officialID,
		Status:     status,
		Plan:       domain.NewPlan("case-1", "the first plan", []string{"first week"}, "", "adopting", "sug-1", "the old top"),
	}
}

// TestRevisePlanRestartsFromReopened is the core flow: a reopened case gets a fresh
// plan, moves to InProgress (active work again), re-snapshots today's top, records
// the pivot on the timeline, mirrors the problem status, and audits plan_revised.
func TestRevisePlanRestartsFromReopened(t *testing.T) {
	repo := newFakeRepo()
	repo.put(stalledCase("off-1", domain.StatusReopened))
	audit := &fakeAudit{}
	problems := &fakeProblems{}
	sug := fakeSuggestions{top: application.TopSuggestion{ID: "sug-9", Text: "a different fix entirely", Found: true}}

	uc := application.NewRevisePlan(repo, sug, problems, audit)
	c, err := uc.Execute(context.Background(), "case-1", "off-1", "try a new approach", []string{"survey", "rebuild"}, "", "adopting the new idea", "the first plan hit a wall")
	if err != nil {
		t.Fatalf("revise plan: %v", err)
	}
	if c.Status != domain.StatusInProgress {
		t.Fatalf("status = %s, want InProgress", c.Status)
	}
	if c.Plan == nil || c.Plan.Strategy != "try a new approach" || len(c.Plan.Tasks) != 2 {
		t.Fatalf("new plan not attached correctly: %+v", c.Plan)
	}
	// The new plan answers TODAY's top, not the stale snapshot the old plan carried.
	if c.Plan.AnsweredSuggestionID != "sug-9" || c.Plan.AnsweredSuggestionTxt != "a different fix entirely" {
		t.Fatalf("snapshot = %q/%q, want the re-snapshotted top", c.Plan.AnsweredSuggestionID, c.Plan.AnsweredSuggestionTxt)
	}
	// The restart is on the public timeline, carrying the reason.
	if len(c.Updates) != 1 || !strings.Contains(c.Updates[0].Text, "the first plan hit a wall") {
		t.Fatalf("updates = %+v, want a 'Plan revised' entry with the reason", c.Updates)
	}
	if c.Updates[0].Kind != domain.UpdateProgress {
		t.Fatalf("update kind = %s, want progress", c.Updates[0].Kind)
	}
	// The public problem status is mirrored to InProgress, so it never disagrees
	// with the case.
	if problems.status != string(domain.StatusInProgress) {
		t.Fatalf("problem status mirrored to %q, want InProgress", problems.status)
	}
	if len(audit.actions) != 1 || audit.actions[0] != "plan_revised" {
		t.Fatalf("audit = %v, want [plan_revised]", audit.actions)
	}

	// The new plan and the timeline entry must be persisted, not just returned.
	saved, _ := repo.GetByID(context.Background(), "case-1")
	if saved.Plan.Strategy != "try a new approach" || len(saved.Updates) != 1 {
		t.Fatalf("restart not persisted: plan=%+v updates=%+v", saved.Plan, saved.Updates)
	}
}

// TestRevisePlanRestartsFromInProgress covers the after-an-obstacle case: an
// obstacle was denied, the case is back at InProgress, and the official re-plans.
func TestRevisePlanRestartsFromInProgress(t *testing.T) {
	repo := newFakeRepo()
	repo.put(stalledCase("off-1", domain.StatusInProgress))
	uc := application.NewRevisePlan(repo, fakeSuggestions{top: application.TopSuggestion{}}, &fakeProblems{}, &fakeAudit{})

	c, err := uc.Execute(context.Background(), "case-1", "off-1", "new plan", []string{"week one"}, "", "here is the revised plan", "resumed after the budget cleared")
	if err != nil {
		t.Fatalf("revise plan: %v", err)
	}
	if c.Status != domain.StatusInProgress {
		t.Fatalf("status = %s, want InProgress", c.Status)
	}
	// No top suggestion is not an error — the response stands on its own, as in
	// SubmitPlan.
	if c.Plan.AnsweredSuggestionID != "" {
		t.Fatalf("expected empty snapshot with no top, got %q", c.Plan.AnsweredSuggestionID)
	}
}

func TestRevisePlanValidation(t *testing.T) {
	sug := fakeSuggestions{top: application.TopSuggestion{ID: "sug-1", Text: "x", Found: true}}

	tests := []struct {
		name               string
		status             domain.Status
		officialID         string
		strategy           string
		tasks              []string
		suggestionResponse string
		reason             string
		wantErr            error
	}{
		{"happy path from reopened", domain.StatusReopened, "off-1", "patch it", []string{"do the work"}, "adopting", "trying again", nil},
		{"happy path from in progress", domain.StatusInProgress, "off-1", "patch it", []string{"do the work"}, "adopting", "trying again", nil},
		// The restart reason is required, exactly as the strategy and the response are —
		// it is the public "what changed".
		{"empty reason is refused", domain.StatusReopened, "off-1", "patch it", []string{"do the work"}, "adopting", "   ", domain.ErrEmptyText},
		{"empty strategy is refused", domain.StatusReopened, "off-1", "  ", []string{"do the work"}, "adopting", "trying again", domain.ErrEmptyText},
		{"empty response is refused", domain.StatusReopened, "off-1", "patch it", []string{"do the work"}, "  ", "trying again", domain.ErrEmptyText},
		{"no weeks is refused", domain.StatusReopened, "off-1", "patch it", nil, "adopting", "trying again", domain.ErrEmptyText},
		{"a blank week task is refused", domain.StatusReopened, "off-1", "patch it", []string{"do the work", "  "}, "adopting", "trying again", domain.ErrEmptyText},
		{"another official's case is refused", domain.StatusReopened, "off-2", "patch it", []string{"do the work"}, "adopting", "trying again", domain.ErrNotCaseOwner},
		// A blocked case must be unblocked before it can be re-planned (A.x): replan
		// is not a way around adjudication.
		{"replan from blocked is illegal", domain.StatusBlocked, "off-1", "patch it", []string{"do the work"}, "adopting", "trying again", domain.ErrIllegalTransition},
		{"replan from acknowledged is illegal", domain.StatusAcknowledged, "off-1", "patch it", []string{"do the work"}, "adopting", "trying again", domain.ErrIllegalTransition},
		{"replan from planned is illegal", domain.StatusPlanned, "off-1", "patch it", []string{"do the work"}, "adopting", "trying again", domain.ErrIllegalTransition},
		{"replan from done is illegal", domain.StatusDone, "off-1", "patch it", []string{"do the work"}, "adopting", "trying again", domain.ErrIllegalTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			c := stalledCase("off-1", tt.status)
			repo.put(c)

			uc := application.NewRevisePlan(repo, sug, &fakeProblems{}, &fakeAudit{})
			_, err := uc.Execute(context.Background(), "case-1", tt.officialID, tt.strategy, tt.tasks, "", tt.suggestionResponse, tt.reason)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
