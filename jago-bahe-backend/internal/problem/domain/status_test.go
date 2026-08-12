package domain_test

import (
	"testing"

	"jago-bahe-backend/internal/problem/domain"
)

func TestStatusValid(t *testing.T) {
	all := []domain.Status{
		domain.StatusPendingApproval,
		domain.StatusReported,
		domain.StatusValidated,
		domain.StatusAssigned,
		domain.StatusInProgress,
		domain.StatusBlocked,
		domain.StatusDone,
		domain.StatusResolved,
		domain.StatusReopened,
		domain.StatusRejected,
		domain.StatusWithdrawn,
	}
	if len(all) != 11 {
		t.Fatalf("expected the eleven-state vocabulary, got %d", len(all))
	}
	for _, s := range all {
		if !s.Valid() {
			t.Errorf("Status(%q).Valid() = false, want true", s)
		}
	}

	for _, s := range []domain.Status{"", "Pending", "reported", "Nonsense"} {
		if s.Valid() {
			t.Errorf("Status(%q).Valid() = true, want false", s)
		}
	}
}

// TestPublicStatusesExcludesPending is the load-bearing one: a report is held
// invisible until its union admin approves it, so PendingApproval must be the one
// state absent from the feed's default set. Every other state is public — a valid
// state wrongly absent here would be a report nobody can find, and PendingApproval
// wrongly present would leak an unscreened report into the feed.
func TestPublicStatusesExcludesPending(t *testing.T) {
	public := domain.PublicStatuses()

	if len(public) != 10 {
		t.Fatalf("PublicStatuses() has %d entries, want 10 (everything but PendingApproval)", len(public))
	}

	seen := make(map[domain.Status]bool, len(public))
	for _, s := range public {
		if !s.Valid() {
			t.Errorf("PublicStatuses() contains unknown status %q", s)
		}
		seen[s] = true
	}
	if seen[domain.StatusPendingApproval] {
		t.Error("PublicStatuses() includes PendingApproval — an unscreened report would leak into the feed")
	}
	for _, s := range []domain.Status{
		domain.StatusReported, domain.StatusValidated, domain.StatusAssigned,
		domain.StatusInProgress, domain.StatusBlocked, domain.StatusDone,
		domain.StatusResolved, domain.StatusReopened, domain.StatusRejected,
		domain.StatusWithdrawn,
	} {
		if !seen[s] {
			t.Errorf("PublicStatuses() omits %q — it would be unreachable from the feed", s)
		}
	}
}

// TestPublicStatusesIncludesRejected pins the choice that survived the move to
// post-moderation: a rejection is public with its reason, so a wrongly-buried
// problem keeps a public witness.
func TestPublicStatusesIncludesRejected(t *testing.T) {
	var found bool
	for _, s := range domain.PublicStatuses() {
		if s == domain.StatusRejected {
			found = true
		}
	}
	if !found {
		t.Fatal("PublicStatuses() omits Rejected — rejections must stay publicly visible")
	}
}

func TestStatusPubliclyVisible(t *testing.T) {
	tests := []struct {
		status domain.Status
		want   bool
	}{
		{domain.StatusReported, true},
		{domain.StatusRejected, true},
		{domain.StatusValidated, true},
		{domain.StatusAssigned, true},
		{domain.StatusInProgress, true},
		{domain.StatusBlocked, true},
		{domain.StatusDone, true},
		{domain.StatusResolved, true},
		{domain.StatusReopened, true},
		{domain.StatusWithdrawn, true},
		{"PendingApproval", false},
		{"Nonsense", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.PubliclyVisible(); got != tt.want {
				t.Fatalf("Status(%q).PubliclyVisible() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// TestStatusRejectable fixes the takedown window. Rejection covers both the
// screening decision (from PendingApproval, before publication) and the narrow
// post-publication takedown (Reported/Validated). It is a remedy for spam and
// abuse, not a veto over an official's work: once a problem is assigned, a case
// exists and resolution owns the lifecycle, so a takedown would erase work already
// done in public.
func TestStatusRejectable(t *testing.T) {
	tests := []struct {
		status domain.Status
		want   bool
	}{
		{domain.StatusPendingApproval, true},
		{domain.StatusReported, true},
		{domain.StatusValidated, true},
		{domain.StatusAssigned, false},
		{domain.StatusInProgress, false},
		{domain.StatusBlocked, false},
		{domain.StatusDone, false},
		{domain.StatusResolved, false},
		{domain.StatusReopened, false},
		{domain.StatusRejected, false}, // rejecting twice is not a thing
		{domain.StatusWithdrawn, false},
		{"Nonsense", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.Rejectable(); got != tt.want {
				t.Fatalf("Status(%q).Rejectable() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// TestStatusWithdrawable mirrors Rejectable: the reporter may retract their own
// report only before it is assigned, for the same reason — once a case exists an
// official is working in public, and a withdrawal there would erase that work. It
// is a separate power from the admin's takedown, so it gets its own test.
func TestStatusWithdrawable(t *testing.T) {
	tests := []struct {
		status domain.Status
		want   bool
	}{
		{domain.StatusPendingApproval, true},
		{domain.StatusReported, true},
		{domain.StatusValidated, true},
		{domain.StatusAssigned, false},
		{domain.StatusInProgress, false},
		{domain.StatusBlocked, false},
		{domain.StatusDone, false},
		{domain.StatusResolved, false},
		{domain.StatusReopened, false},
		{domain.StatusRejected, false},
		{domain.StatusWithdrawn, false}, // withdrawing twice is not a thing
		{"Nonsense", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.Withdrawable(); got != tt.want {
				t.Fatalf("Status(%q).Withdrawable() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// TestEligibleForAssignment fixes the window in which an admin may forward a
// report to an official. It is deliberately NOT a vote threshold: B17 removed the
// rule that only a Validated problem could be assigned, so a Reported problem is
// assignable the moment its admin approves it — with zero validations if the admin
// judges it trustworthy. V still flips Reported to Validated, but that flip is now
// a public endorsement, not a gate. See CLAUDE.md A.3.1.
//
// The window is bounded on both sides, and both bounds are load-bearing:
// PendingApproval is out because screening comes first (an unscreened report is
// not public and must not reach an official), and every post-assignment state is
// out because a case already exists — assigning again would fork an official's
// public work.
func TestEligibleForAssignment(t *testing.T) {
	tests := []struct {
		status domain.Status
		want   bool
	}{
		// The window: public, and no case exists yet.
		{domain.StatusReported, true},  // assignable with zero votes — B17
		{domain.StatusValidated, true}, // the community vouched first

		// Before the window: not public, so not forwardable.
		{domain.StatusPendingApproval, false},

		// After the window: a case exists and resolution owns the lifecycle.
		{domain.StatusAssigned, false},
		{domain.StatusInProgress, false},
		{domain.StatusBlocked, false},
		{domain.StatusDone, false},
		{domain.StatusResolved, false},
		{domain.StatusReopened, false},

		// Terminal.
		{domain.StatusRejected, false},
		{domain.StatusWithdrawn, false},

		{"Nonsense", false},
	}
	if len(tests) != 12 {
		t.Fatalf("expected the eleven-state vocabulary plus one unknown, got %d cases", len(tests))
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.EligibleForAssignment(); got != tt.want {
				t.Fatalf("Status(%q).EligibleForAssignment() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
