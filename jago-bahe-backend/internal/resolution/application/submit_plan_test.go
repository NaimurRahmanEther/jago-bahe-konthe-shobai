package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
)

// fakeSuggestions stands in for the suggestion context's ranking authority.
type fakeSuggestions struct {
	top application.TopSuggestion
	err error
}

func (s fakeSuggestions) Top(context.Context, string) (application.TopSuggestion, error) {
	return s.top, s.err
}

// acknowledgedCase is a case sitting where a plan is legal.
func acknowledgedCase(officialID string) *domain.Case {
	return &domain.Case{
		ID:         "case-1",
		ProblemID:  "prob-1",
		OfficialID: officialID,
		Status:     domain.StatusAcknowledged,
	}
}

// TestSubmitPlanSnapshotsTopSuggestion is the point of the snapshot: the plan must
// record the question it was answering, at the moment it was answered, so later
// upvotes cannot rewrite what an official appears to have replied to.
func TestSubmitPlanSnapshotsTopSuggestion(t *testing.T) {
	repo := newFakeRepo()
	repo.put(acknowledgedCase("off-1"))
	audit := &fakeAudit{}
	sug := fakeSuggestions{top: application.TopSuggestion{ID: "sug-9", Text: "resurface the whole lane", Found: true}}

	uc := application.NewSubmitPlan(repo, sug, audit)
	c, err := uc.Execute(context.Background(), "case-1", "off-1", "patch it", []string{"clear inlet", "lay pipe", "repave"}, "", "adopting this")
	if err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	if c.Status != domain.StatusPlanned {
		t.Fatalf("status = %s, want Planned", c.Status)
	}
	if c.Plan == nil {
		t.Fatal("plan not attached")
	}
	// The weekly checklist is stored, and TimelineWeeks is derived from it — never a
	// separate input — so the two can never disagree.
	if len(c.Plan.Tasks) != 3 || c.Plan.TimelineWeeks != 3 {
		t.Fatalf("tasks = %d, timelineWeeks = %d, want 3 and 3", len(c.Plan.Tasks), c.Plan.TimelineWeeks)
	}
	if c.Plan.Tasks[0].WeekNumber != 1 || c.Plan.Tasks[0].Task != "clear inlet" || c.Plan.Tasks[0].Completed {
		t.Fatalf("first task = %+v, want week 1 'clear inlet' not completed", c.Plan.Tasks[0])
	}
	if c.Plan.AnsweredSuggestionID != "sug-9" {
		t.Errorf("AnsweredSuggestionID = %q, want sug-9", c.Plan.AnsweredSuggestionID)
	}
	if c.Plan.AnsweredSuggestionTxt != "resurface the whole lane" {
		t.Errorf("AnsweredSuggestionTxt = %q, want the snapshotted text", c.Plan.AnsweredSuggestionTxt)
	}
	if c.Plan.SuggestionResponse != "adopting this" {
		t.Errorf("SuggestionResponse = %q", c.Plan.SuggestionResponse)
	}

	// The snapshot must be persisted, not just returned — the public record reads
	// it back from storage.
	saved, err := repo.GetByID(context.Background(), "case-1")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if saved.Plan == nil || saved.Plan.AnsweredSuggestionID != "sug-9" {
		t.Fatalf("snapshot not persisted, got %+v", saved.Plan)
	}
}

// TestSubmitPlanWithNoTopSuggestion pins that a problem nobody upvoted does not
// block an official from planning. Zero upvotes means no top (the B3 rule), and
// the response then simply stands on its own.
func TestSubmitPlanWithNoTopSuggestion(t *testing.T) {
	repo := newFakeRepo()
	repo.put(acknowledgedCase("off-1"))
	audit := &fakeAudit{}
	sug := fakeSuggestions{top: application.TopSuggestion{}} // Found: false

	uc := application.NewSubmitPlan(repo, sug, audit)
	c, err := uc.Execute(context.Background(), "case-1", "off-1", "patch it", []string{"week one"}, "", "nothing was suggested, here is my plan")
	if err != nil {
		t.Fatalf("a problem with no top suggestion must not block a plan: %v", err)
	}
	if c.Plan.AnsweredSuggestionID != "" || c.Plan.AnsweredSuggestionTxt != "" {
		t.Fatalf("expected an empty snapshot, got %q / %q", c.Plan.AnsweredSuggestionID, c.Plan.AnsweredSuggestionTxt)
	}
	if c.Plan.SuggestionResponse == "" {
		t.Fatal("the response must still be recorded")
	}
}

func TestSubmitPlanValidation(t *testing.T) {
	sug := fakeSuggestions{top: application.TopSuggestion{ID: "sug-1", Text: "x", Found: true}}

	tests := []struct {
		name               string
		status             domain.Status
		officialID         string
		strategy           string
		tasks              []string
		suggestionResponse string
		wantErr            error
	}{
		{"happy path", domain.StatusAcknowledged, "off-1", "patch it", []string{"do the work"}, "adopting", nil},
		// The plan must answer the top suggestion — that requirement is the whole
		// reason the response field exists (Concept §4, Figure 3).
		{"empty response is refused", domain.StatusAcknowledged, "off-1", "patch it", []string{"do the work"}, "   ", domain.ErrEmptyText},
		{"empty strategy is refused", domain.StatusAcknowledged, "off-1", "  ", []string{"do the work"}, "adopting", domain.ErrEmptyText},
		// A plan is a week-by-week checklist: no weeks is not a plan, and a listed
		// week with no task is a hole in it.
		{"no weeks is refused", domain.StatusAcknowledged, "off-1", "patch it", nil, "adopting", domain.ErrEmptyText},
		{"a blank week task is refused", domain.StatusAcknowledged, "off-1", "patch it", []string{"do the work", "  "}, "adopting", domain.ErrEmptyText},
		{"another official's case is refused", domain.StatusAcknowledged, "off-2", "patch it", []string{"do the work"}, "adopting", domain.ErrNotCaseOwner},
		{"planning before acknowledging is illegal", domain.StatusAssigned, "off-1", "patch it", []string{"do the work"}, "adopting", domain.ErrIllegalTransition},
		{"planning twice is illegal", domain.StatusPlanned, "off-1", "patch it", []string{"do the work"}, "adopting", domain.ErrIllegalTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			c := acknowledgedCase("off-1")
			c.Status = tt.status
			repo.put(c)

			uc := application.NewSubmitPlan(repo, sug, &fakeAudit{})
			_, err := uc.Execute(context.Background(), "case-1", tt.officialID, tt.strategy, tt.tasks, "", tt.suggestionResponse)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestSubmitPlanAudits pins that planning is written to the public trail.
func TestSubmitPlanAudits(t *testing.T) {
	repo := newFakeRepo()
	repo.put(acknowledgedCase("off-1"))
	audit := &fakeAudit{}
	sug := fakeSuggestions{top: application.TopSuggestion{ID: "sug-1", Text: "x", Found: true}}

	uc := application.NewSubmitPlan(repo, sug, audit)
	if _, err := uc.Execute(context.Background(), "case-1", "off-1", "patch it", []string{"do the work"}, "", "adopting"); err != nil {
		t.Fatalf("submit plan: %v", err)
	}
	if len(audit.actions) != 1 || audit.actions[0] != "plan_submitted" {
		t.Fatalf("audit actions = %v, want [plan_submitted]", audit.actions)
	}
}
