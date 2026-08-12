package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
)

// The public feed and the reporter's own view answer different questions — "what
// is on the public record" against "what did I file". They are two use cases so
// that each stays provable alone; these tests pin the feed's half of that split,
// and in particular that B12's Filter.ReporterID never leaks into it. A feed that
// quietly scoped to a reporter, or that accepted a reporter parameter, would
// become an index of what a named person filed.
func TestListProblemsNeverScopesToAReporter(t *testing.T) {
	repo := &fakeListRepo{}

	if _, err := application.NewListProblems(repo).Execute(context.Background(), "", "", ""); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if repo.got.ReporterID != "" {
		t.Errorf("Filter.ReporterID = %q, want empty — the public feed has no reporter filter", repo.got.ReporterID)
	}
}

// TestListProblemsDerivesPublicStatuses pins that the feed derives its own status
// set rather than passing the caller's through. PublicStatuses() covers every
// state today, so this withholds nothing — it is the seam that keeps the
// derivation in one place, and the reason an unrecognised ?status= is refused
// outright instead of quietly matching no rows.
func TestListProblemsDerivesPublicStatuses(t *testing.T) {
	repo := &fakeListRepo{}

	if _, err := application.NewListProblems(repo).Execute(context.Background(), "", "", ""); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if repo.got.Statuses == nil {
		t.Fatal("Filter.Statuses = nil — the feed must pass an explicit set, not lean on the repo's default")
	}
	if len(repo.got.Statuses) != len(domain.PublicStatuses()) {
		t.Errorf("Statuses = %v, want PublicStatuses()", repo.got.Statuses)
	}
}

func TestListProblemsStatusFilter(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr error
		want    []domain.Status
	}{
		{"absent status widens to the public set", "", nil, domain.PublicStatuses()},
		{"a public status narrows to it", "Reported", nil, []domain.Status{domain.StatusReported}},
		// Rejected is deliberately public — a rejection keeps its reason on the
		// record rather than disappearing.
		{"rejected is public and listable", "Rejected", nil, []domain.Status{domain.StatusRejected}},
		// The retired pre-publication state is now simply an unknown status. It is
		// refused rather than emptied, like any other unrecognised value.
		{"the retired pending state is refused", "PendingApproval", domain.ErrStatusNotPublic, nil},
		{"an unknown status is refused", "Bogus", domain.ErrStatusNotPublic, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeListRepo{}
			_, err := application.NewListProblems(repo).Execute(context.Background(), "", tt.status, "")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if len(repo.got.Statuses) != len(tt.want) {
				t.Fatalf("Statuses = %v, want %v", repo.got.Statuses, tt.want)
			}
		})
	}
}
