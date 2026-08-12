package domain_test

import (
	"testing"

	"jago-bahe-backend/internal/scorecard/domain"
)

// TestFairnessFlag pins the public record's blocked-on-higher-authority marker to
// the same rule the scorecard's numbers use. If these two ever disagree, an
// official's profile would contradict their own scorecard — a case counted as
// "fairly blocked" in the stats but rendered as plain silence in the list.
func TestFairnessFlag(t *testing.T) {
	tests := []struct {
		name   string
		status string
		adj    domain.Adjudication
		want   bool
	}{
		{"blocked and confirmed real is protected", "Blocked", domain.AdjConfirmed, true},
		// Only the named authority's confirmation protects an official. A blocker
		// nobody has judged yet is not yet an excuse...
		{"blocked but not yet adjudicated is not protected", "Blocked", domain.AdjPending, false},
		// ...and one judged an excuse bounced back to them.
		{"blocked but denied is not protected", "Blocked", domain.AdjDenied, false},
		{"blocked with no adjudication is not protected", "Blocked", domain.AdjNone, false},
		// The flag is about being blocked, not about having once been blocked.
		{"in progress is not protected", "InProgress", domain.AdjConfirmed, false},
		{"resolved is not protected", "Resolved", domain.AdjConfirmed, false},
		{"assigned is not protected", "Assigned", domain.AdjNone, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.FairnessFlag(tt.status, tt.adj); got != tt.want {
				t.Fatalf("FairnessFlag(%q, %q) = %v, want %v", tt.status, tt.adj, got, tt.want)
			}
		})
	}
}

// TestBuildRecordIsFactual guards the guardrail: the record must carry cases
// through untouched, in the order the query gave them. Any sorting by "best" or
// filtering of unflattering cases here would be the platform editorializing about
// a named person (Concept §10), so there is nothing to sort and nothing to drop.
func TestBuildRecordIsFactual(t *testing.T) {
	cases := []domain.CaseRecord{
		{CaseID: "case-1", Status: "Resolved"},
		{CaseID: "case-2", Status: "Reopened"},
		{CaseID: "case-3", Status: "Blocked", BlockedOnHigherAuthority: true},
	}
	rec := domain.BuildRecord("off-1", cases)

	if rec.OfficialID != "off-1" {
		t.Fatalf("OfficialID = %q", rec.OfficialID)
	}
	if len(rec.Cases) != len(cases) {
		t.Fatalf("record dropped cases: got %d, want %d", len(rec.Cases), len(cases))
	}
	for i := range cases {
		if rec.Cases[i].CaseID != cases[i].CaseID {
			t.Fatalf("case %d reordered: got %s, want %s", i, rec.Cases[i].CaseID, cases[i].CaseID)
		}
	}
}

func TestBuildRecordWithNoCases(t *testing.T) {
	rec := domain.BuildRecord("off-1", nil)
	if rec.OfficialID != "off-1" {
		t.Fatalf("OfficialID = %q", rec.OfficialID)
	}
	if len(rec.Cases) != 0 {
		t.Fatalf("expected an empty record, got %d cases", len(rec.Cases))
	}
}
