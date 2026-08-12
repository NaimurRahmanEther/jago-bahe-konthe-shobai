package domain_test

import (
	"errors"
	"testing"

	"jago-bahe-backend/internal/problem/domain"
)

func TestRejectionReasonValid(t *testing.T) {
	tests := []struct {
		name   string
		reason domain.RejectionReason
		want   bool
	}{
		{"spam", domain.ReasonSpam, true},
		{"abusive", domain.ReasonAbusive, true},
		{"duplicate", domain.ReasonDuplicate, true},
		{"wrong area", domain.ReasonWrongArea, true},
		{"empty is not a ground", "", false},
		// The grounds are a closed enum precisely so an admin cannot invent a
		// merit-based reason: rejecting on merit is the community's call via V.
		// This survived the move to post-moderation unchanged — publishing on report
		// removed the admin's power to delay, not their duty to justify a takedown.
		{"free text is not a ground", "i do not think this is important", false},
		{"unimportant is not a ground", "unimportant", false},
		{"untrue is not a ground", "untrue", false},
		{"wrong case", "Spam", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.reason.Valid(); got != tt.want {
				t.Fatalf("RejectionReason(%q).Valid() = %v, want %v", tt.reason, got, tt.want)
			}
		})
	}
}

func TestScreeningReject(t *testing.T) {
	scr := domain.NewScreening()
	tests := []struct {
		name    string
		current domain.Status
		reason  domain.RejectionReason
		want    domain.Status
		wantErr error
	}{
		// A pending problem is rejected during screening — this is the admin's
		// decision on a report before it is ever public.
		{"a pending problem rejects on spam", domain.StatusPendingApproval, domain.ReasonSpam, domain.StatusRejected, nil},
		{"a pending problem rejects on abuse", domain.StatusPendingApproval, domain.ReasonAbusive, domain.StatusRejected, nil},

		// A published problem is still takeable-down before assignment: a report
		// that slipped through approval, or a duplicate that only later surfaces.
		{"a reported problem rejects on spam", domain.StatusReported, domain.ReasonSpam, domain.StatusRejected, nil},
		{"a reported problem rejects on abuse", domain.StatusReported, domain.ReasonAbusive, domain.StatusRejected, nil},
		{"a reported problem rejects as duplicate", domain.StatusReported, domain.ReasonDuplicate, domain.StatusRejected, nil},
		{"a reported problem rejects as wrong area", domain.StatusReported, domain.ReasonWrongArea, domain.StatusRejected, nil},
		{"a validated problem still rejects on spam", domain.StatusValidated, domain.ReasonSpam, domain.StatusRejected, nil},

		{"reason must be a known ground", domain.StatusReported, "not-a-ground", "", domain.ErrInvalidRejectionReason},
		{"reason is required", domain.StatusReported, "", "", domain.ErrInvalidRejectionReason},

		// Past assignment the case belongs to resolution and an official is working
		// in public. A takedown there would erase their work, so the remedy stops.
		{"an assigned problem cannot be rejected", domain.StatusAssigned, domain.ReasonSpam, "", domain.ErrNotRejectable},
		{"an in-progress problem cannot be rejected", domain.StatusInProgress, domain.ReasonSpam, "", domain.ErrNotRejectable},
		{"a blocked problem cannot be rejected", domain.StatusBlocked, domain.ReasonSpam, "", domain.ErrNotRejectable},
		{"a done problem cannot be rejected", domain.StatusDone, domain.ReasonSpam, "", domain.ErrNotRejectable},
		{"a resolved problem cannot be rejected", domain.StatusResolved, domain.ReasonSpam, "", domain.ErrNotRejectable},
		{"a reopened problem cannot be rejected", domain.StatusReopened, domain.ReasonSpam, "", domain.ErrNotRejectable},
		{"rejecting twice is refused", domain.StatusRejected, domain.ReasonSpam, "", domain.ErrNotRejectable},
		{"a withdrawn problem cannot be rejected", domain.StatusWithdrawn, domain.ReasonSpam, "", domain.ErrNotRejectable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scr.Reject(tt.current, tt.reason)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Reject(%s, %q) err = %v, want %v", tt.current, tt.reason, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("Reject(%s, %q) = %s, want %s", tt.current, tt.reason, got, tt.want)
			}
		})
	}
}

// TestScreeningApprove pins the gate's publish step: only a PendingApproval report
// may be approved into public view, and it lands at Reported (public but still
// unvalidated). Approving anything already public — or already terminal — is
// ErrNotPending, so a double-approve or an approve of a rejected report is refused.
func TestScreeningApprove(t *testing.T) {
	scr := domain.NewScreening()
	tests := []struct {
		name    string
		current domain.Status
		want    domain.Status
		wantErr error
	}{
		{"a pending report approves to Reported", domain.StatusPendingApproval, domain.StatusReported, nil},
		{"an already-public report is not pending", domain.StatusReported, "", domain.ErrNotPending},
		{"a validated report is not pending", domain.StatusValidated, "", domain.ErrNotPending},
		{"a rejected report is not pending", domain.StatusRejected, "", domain.ErrNotPending},
		{"an assigned report is not pending", domain.StatusAssigned, "", domain.ErrNotPending},
		{"an unknown status is not pending", "Nonsense", "", domain.ErrNotPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scr.Approve(tt.current)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Approve(%s) err = %v, want %v", tt.current, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("Approve(%s) = %s, want %s", tt.current, got, tt.want)
			}
		})
	}
}

// TestScreeningRejectChecksStatusBeforeReason pins the precedence: a problem past
// the takedown window is refused as not-rejectable even when the ground is also
// invalid, so the error an admin sees names the real obstacle rather than sending
// them to fix a reason that was never going to be accepted.
func TestScreeningRejectChecksStatusBeforeReason(t *testing.T) {
	scr := domain.NewScreening()
	if _, err := scr.Reject(domain.StatusResolved, "nonsense"); !errors.Is(err, domain.ErrNotRejectable) {
		t.Fatalf("Reject(Resolved, nonsense) err = %v, want ErrNotRejectable", err)
	}
}
