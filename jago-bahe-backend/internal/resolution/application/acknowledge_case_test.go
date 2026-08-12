package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
)

func assignedCase(officialID string) *domain.Case {
	return &domain.Case{
		ID:         "case-1",
		ProblemID:  "prob-1",
		OfficialID: officialID,
		Status:     domain.StatusAssigned,
	}
}

// TestAcknowledgeRejectsUnknownDecision is the regression guard for a real bug:
// the handler used to map `decision == "accept"` onto a bool, so ANY other value
// — a typo, a missing field, an empty body, "Accept" — fell through to *dispute*.
//
// That was not a cosmetic mis-parse. A disputed case is excluded from the
// official's scorecard entirely (the fairness rule treats it as "not this
// official's case"), so a malformed request silently removed a case from a named
// person's public record. An unrecognised decision must be refused outright.
func TestAcknowledgeRejectsUnknownDecision(t *testing.T) {
	tests := []struct {
		name     string
		decision domain.Decision
	}{
		{"empty (a missing field)", ""},
		{"a typo", "acept"},
		{"wrong case", "Accept"},
		{"nonsense", "banana"},
		{"the bool that used to leak through", "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.put(assignedCase("off-1"))
			uc := application.NewAcknowledgeCase(repo, &fakeAudit{})

			_, err := uc.Execute(context.Background(), "case-1", "off-1", tt.decision, "")
			if !errors.Is(err, domain.ErrInvalidDecision) {
				t.Fatalf("Execute(%q) err = %v, want ErrInvalidDecision", tt.decision, err)
			}

			// The case must be untouched — in particular, not disputed.
			saved, _ := repo.GetByID(context.Background(), "case-1")
			if saved.Status != domain.StatusAssigned {
				t.Fatalf("an unrecognised decision changed the case to %s", saved.Status)
			}
		})
	}
}

// TestAcknowledgeDisputeRequiresReason enforces what the use case's own doc has
// always claimed. Disputing drops the case off the official's public record, so
// it is the single most consequential thing they can do to it — doing that
// without saying why is exactly the silent rejection the design forbids.
func TestAcknowledgeDisputeRequiresReason(t *testing.T) {
	for _, reason := range []string{"", "   ", "\t\n"} {
		repo := newFakeRepo()
		repo.put(assignedCase("off-1"))
		uc := application.NewAcknowledgeCase(repo, &fakeAudit{})

		_, err := uc.Execute(context.Background(), "case-1", "off-1", domain.DecisionDispute, reason)
		if !errors.Is(err, domain.ErrEmptyText) {
			t.Fatalf("dispute with reason %q: err = %v, want ErrEmptyText", reason, err)
		}
		saved, _ := repo.GetByID(context.Background(), "case-1")
		if saved.Status != domain.StatusAssigned {
			t.Fatalf("a reasonless dispute still moved the case to %s", saved.Status)
		}
	}
}

func TestAcknowledgeAccept(t *testing.T) {
	repo := newFakeRepo()
	repo.put(assignedCase("off-1"))
	audit := &fakeAudit{}
	uc := application.NewAcknowledgeCase(repo, audit)

	c, err := uc.Execute(context.Background(), "case-1", "off-1", domain.DecisionAccept, "")
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if c.Status != domain.StatusAcknowledged {
		t.Fatalf("status = %s, want Acknowledged", c.Status)
	}
	if c.AcknowledgedAt == nil {
		t.Fatal("AcknowledgedAt not stamped — the first-response clock depends on it")
	}
	if len(audit.actions) != 1 || audit.actions[0] != "acknowledged" {
		t.Fatalf("audit = %v, want [acknowledged]", audit.actions)
	}
}

func TestAcknowledgeDisputeWithReason(t *testing.T) {
	repo := newFakeRepo()
	repo.put(assignedCase("off-1"))
	audit := &fakeAudit{}
	uc := application.NewAcknowledgeCase(repo, audit)

	c, err := uc.Execute(context.Background(), "case-1", "off-1", domain.DecisionDispute, "  this road is under the upazila, not the union  ")
	if err != nil {
		t.Fatalf("dispute: %v", err)
	}
	if c.Status != domain.StatusDisputed {
		t.Fatalf("status = %s, want Disputed", c.Status)
	}
	if c.DisputeReason != "this road is under the upazila, not the union" {
		t.Fatalf("DisputeReason = %q (should be trimmed and kept)", c.DisputeReason)
	}
	if len(audit.actions) != 1 || audit.actions[0] != "disputed" {
		t.Fatalf("audit = %v, want [disputed]", audit.actions)
	}
}

func TestAcknowledgeOwnershipAndState(t *testing.T) {
	tests := []struct {
		name       string
		status     domain.Status
		officialID string
		wantErr    error
	}{
		{"another official's case", domain.StatusAssigned, "off-2", domain.ErrNotCaseOwner},
		{"already acknowledged", domain.StatusAcknowledged, "off-1", domain.ErrIllegalTransition},
		{"already planned", domain.StatusPlanned, "off-1", domain.ErrIllegalTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			c := assignedCase("off-1")
			c.Status = tt.status
			repo.put(c)

			uc := application.NewAcknowledgeCase(repo, &fakeAudit{})
			_, err := uc.Execute(context.Background(), "case-1", tt.officialID, domain.DecisionAccept, "")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecisionValid(t *testing.T) {
	tests := []struct {
		decision domain.Decision
		want     bool
	}{
		{domain.DecisionAccept, true},
		{domain.DecisionDispute, true},
		{"", false},
		{"Accept", false},
		{"acept", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.decision), func(t *testing.T) {
			if got := tt.decision.Valid(); got != tt.want {
				t.Fatalf("Decision(%q).Valid() = %v, want %v", tt.decision, got, tt.want)
			}
		})
	}
}
