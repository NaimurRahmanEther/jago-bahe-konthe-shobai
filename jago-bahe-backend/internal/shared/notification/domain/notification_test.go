package domain_test

import (
	"strings"
	"testing"

	"jago-bahe-backend/internal/shared/notification/domain"
)

// TestTypeValid walks AllTypes rather than restating the list, so a ninth constant
// added without a Valid() case fails here. That is the cheap half of guarding
// against the `adjudicated` failure (A.3.5 rule 2) — a type nothing recognises
// renders as a row nobody sees, which looks exactly like a seat where the event
// never happens.
func TestTypeValid(t *testing.T) {
	if len(domain.AllTypes) != 8 {
		t.Fatalf("AllTypes has %d entries, want 8 — the loop-closing set is eight events; "+
			"if you added one, add its bn.json key, its migration CHECK value and its entitlement row too",
			len(domain.AllTypes))
	}
	for _, ty := range domain.AllTypes {
		if !ty.Valid() {
			t.Errorf("Type(%q).Valid() = false, want true", ty)
		}
	}

	// The exact literals, spelled out. They are the DB CHECK values, the JSON the
	// frontend switches on, and the bn.json key suffixes; a renamed constant that
	// silently changes its string would pass the loop above and break all three.
	want := map[domain.Type]string{
		domain.TypeProblemApproved:       "problem_approved",
		domain.TypeProblemRejected:       "problem_rejected",
		domain.TypeProblemAssigned:       "problem_assigned",
		domain.TypeConfirmationRequested: "confirmation_requested",
		domain.TypeCaseAssigned:          "case_assigned",
		domain.TypeCaseReopened:          "case_reopened",
		domain.TypeObstacleAdjudicated:   "obstacle_adjudicated",
		domain.TypeObstacleDeclared:      "obstacle_declared",
	}
	for ty, literal := range want {
		if string(ty) != literal {
			t.Errorf("type literal drifted: got %q, want %q", ty, literal)
		}
	}

	for _, bad := range []domain.Type{"", "approved", "Problem_Approved", "adjudicated", "escalated"} {
		if bad.Valid() {
			t.Errorf("Type(%q).Valid() = true, want false", bad)
		}
	}
}

func TestKindValid(t *testing.T) {
	cases := []struct {
		kind domain.Kind
		want bool
	}{
		{domain.KindAccount, true},
		{domain.KindOfficial, true},
		{"", false},
		{"resident", false},
		{"Account", false},
	}
	for _, tc := range cases {
		if got := tc.kind.Valid(); got != tc.want {
			t.Errorf("Kind(%q).Valid() = %v, want %v", tc.kind, got, tc.want)
		}
	}
}

// TestConstructorsCarryTheIDSpace is the anti-transposition test. The two
// constructors exist precisely so an account id can never be written as an office
// id at a call site, and this pins that they set different kinds.
func TestConstructorsCarryTheIDSpace(t *testing.T) {
	res := domain.ForResident("acc-1", domain.TypeProblemApproved, "prob-1", "রাস্তার বাতি", "")
	if res.RecipientKind != domain.KindAccount {
		t.Errorf("ForResident kind = %q, want %q", res.RecipientKind, domain.KindAccount)
	}
	if res.RecipientID != "acc-1" {
		t.Errorf("ForResident id = %q, want acc-1", res.RecipientID)
	}

	off := domain.ForOfficial("off-mayor", domain.TypeCaseAssigned, "prob-1", "রাস্তার বাতি", "2026-08-01T00:00:00Z")
	if off.RecipientKind != domain.KindOfficial {
		t.Errorf("ForOfficial kind = %q, want %q", off.RecipientKind, domain.KindOfficial)
	}
	if off.RecipientID != "off-mayor" {
		t.Errorf("ForOfficial id = %q, want off-mayor", off.RecipientID)
	}
	if off.Detail != "2026-08-01T00:00:00Z" {
		t.Errorf("Detail = %q, want the deadline it was given", off.Detail)
	}
}

func TestNewNotificationDefaults(t *testing.T) {
	n := domain.ForResident("acc-1", domain.TypeProblemApproved, "prob-1", "শিরোনাম", "")

	if !strings.HasPrefix(n.ID, "notif-") {
		t.Errorf("ID = %q, want a notif- prefix", n.ID)
	}
	// Unread on creation, and ReadAt is a nil pointer rather than a zero time: the
	// DTO serializes it as literal null, and "when" is a fact while "whether"
	// derives from it.
	if n.ReadAt != nil {
		t.Errorf("ReadAt = %v, want nil on a new notification", n.ReadAt)
	}
	if !n.Unread() {
		t.Error("Unread() = false on a new notification, want true")
	}
	if n.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want the current time")
	}
	if loc := n.CreatedAt.Location(); loc.String() != "UTC" {
		t.Errorf("CreatedAt location = %s, want UTC", loc)
	}
}

// TestRecipientsEmpty pins the distinction the read path depends on: a caller with
// NO identity gets ErrNoCaller (a 401), while an official-less caller — every
// resident, admin and super admin — is a perfectly normal caller whose OfficialID
// is "" and must never match a blank recipient row.
func TestRecipientsEmpty(t *testing.T) {
	cases := []struct {
		name string
		r    domain.Recipients
		want bool
	}{
		{"anonymous", domain.Recipients{}, true},
		{"resident (no office)", domain.Recipients{AccountID: "acc-1"}, false},
		{"official", domain.Recipients{AccountID: "acc-2", OfficialID: "off-mayor"}, false},
		{"office only", domain.Recipients{OfficialID: "off-mayor"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.Empty(); got != tc.want {
				t.Errorf("Empty() = %v, want %v", got, tc.want)
			}
		})
	}
}
