package application_test

import (
	"context"
	"testing"
	"time"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
)

// TestGetProgressNullShapedBranches is the no-oracle rule, and it is the test
// most likely to look wrong to a reviewer: every one of these cases returns "no
// case, no error" where a REST instinct expects a 404 for the unknown id.
//
// That uniformity is the point. GetProgress never reads the problem row, so it
// genuinely cannot tell an unknown id from a real problem nobody has been assigned
// to yet. Distinguishing them would turn this route into an oracle over problem
// ids for no benefit — the plan behind an assigned case is already public per
// official (Scaffold §5, "Progress is null-shaped, never not-found").
func TestGetProgressNullShapedBranches(t *testing.T) {
	tests := []struct {
		name string
		seed *domain.Case // nil = nothing in storage for this problem
		id   string
	}{
		{
			// An id nobody ever reported. Indistinguishable, deliberately.
			name: "unknown problem id",
			id:   "prob-does-not-exist",
		},
		{
			// Awaiting screening: never assigned, so never has a case. This is the
			// branch the oracle would target.
			name: "problem awaiting screening",
			id:   "prob-pending",
		},
		{
			// Public but not yet assigned by an admin.
			name: "problem not yet assigned",
			id:   "prob-validated",
		},
		{
			// Assigned, but no official has opened it yet — cases materialize lazily,
			// so the row does not exist yet. A 404 here would be a lie about a real,
			// public, assigned problem.
			name: "assigned but case not yet materialized",
			id:   "prob-assigned",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tt.seed != nil {
				repo.put(tt.seed)
			}

			c, ok, err := application.NewGetProgress(repo).Execute(context.Background(), tt.id)
			if err != nil {
				t.Fatalf("err = %v, want nil — a missing case is not an error", err)
			}
			if ok {
				t.Fatalf("ok = true, want false")
			}
			if c != nil {
				t.Fatalf("case = %+v, want nil", c)
			}
		})
	}
}

// TestGetProgressProjectsThePlan pins what a resident actually came for: the plan
// their official published, paired with the snapshotted question it answered.
func TestGetProgressProjectsThePlan(t *testing.T) {
	now := time.Now().UTC()
	repo := newFakeRepo()
	repo.put(&domain.Case{
		ID:         "case-1",
		ProblemID:  "prob-1",
		OfficialID: "off-1",
		Status:     domain.StatusInProgress,
		Plan: &domain.Plan{
			ID:                    "plan-1",
			CaseID:                "case-1",
			Strategy:              "resurface the lane",
			TimelineWeeks:         3,
			Obstacles:             "monsoon",
			SuggestionResponse:    "adopting this",
			AnsweredSuggestionID:  "sug-9",
			AnsweredSuggestionTxt: "resurface the whole lane",
			CreatedAt:             now,
		},
		Updates:  []domain.ProgressUpdate{{ID: "upd-1", CaseID: "case-1", Text: "tender issued", CreatedAt: now}},
		Evidence: []domain.Evidence{{ID: "evd-1", CaseID: "case-1", BeforeImageURL: "b.jpg", AfterImageURL: "a.jpg"}},
	})

	c, ok, err := application.NewGetProgress(repo).Execute(context.Background(), "prob-1")
	if err != nil || !ok {
		t.Fatalf("execute: ok=%v err=%v", ok, err)
	}
	if c.Plan == nil {
		t.Fatal("plan not projected")
	}
	if c.Plan.Strategy != "resurface the lane" || c.Plan.TimelineWeeks != 3 {
		t.Errorf("plan = %+v", c.Plan)
	}
	// The snapshot, not today's top: pairing the response with a question the
	// official was never asked would misrepresent them (Concept §10).
	if c.Plan.AnsweredSuggestionTxt != "resurface the whole lane" {
		t.Errorf("AnsweredSuggestionTxt = %q, want the stored snapshot", c.Plan.AnsweredSuggestionTxt)
	}
	if len(c.Updates) != 1 || len(c.Evidence) != 1 {
		t.Errorf("updates=%d evidence=%d, want 1 and 1", len(c.Updates), len(c.Evidence))
	}
}

// TestGetProgressReportsTheFairnessFlag pins that a case waiting on a confirmed
// obstacle says so, rather than reading as neglect.
func TestGetProgressReportsTheFairnessFlag(t *testing.T) {
	tests := []struct {
		name string
		adj  domain.Adjudication
		want bool
	}{
		{"confirmed real is fairly blocked", domain.AdjudicationConfirmed, true},
		{"not yet adjudicated is not", domain.AdjudicationPending, false},
		{"judged an excuse is not", domain.AdjudicationDenied, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.put(&domain.Case{
				ID: "case-1", ProblemID: "prob-1", Status: domain.StatusBlocked,
				Obstacles: []domain.Obstacle{{ID: "blk-1", CaseID: "case-1", Adjudication: tt.adj}},
			})

			c, ok, err := application.NewGetProgress(repo).Execute(context.Background(), "prob-1")
			if err != nil || !ok {
				t.Fatalf("execute: ok=%v err=%v", ok, err)
			}
			if got := c.BlockedOnHigherAuthority(); got != tt.want {
				t.Errorf("BlockedOnHigherAuthority() = %v, want %v", got, tt.want)
			}
		})
	}
}
