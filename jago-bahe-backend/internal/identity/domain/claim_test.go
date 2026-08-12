package domain_test

import (
	"testing"

	"jago-bahe-backend/internal/identity/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// TestRouteClaim is the rule that decides who may verify a person's claim to an
// office. It deliberately reuses the same union-level split B4 routes assignment
// on, so "who decides union matters" has one answer across the platform.
func TestRouteClaim(t *testing.T) {
	tests := []struct {
		name string
		tier valueobject.Tier
		want domain.ClaimRoute
	}{
		// Union-level: the union's own admin knows these people personally.
		{"ward member", valueobject.TierWardMember, domain.ClaimRouteUnionAdmin},
		{"women member", valueobject.TierWomenMember, domain.ClaimRouteUnionAdmin},
		{"union chairman", valueobject.TierUnionChairman, domain.ClaimRouteUnionAdmin},
		{"pourashava councillor", valueobject.TierPourashavaCouncillor, domain.ClaimRouteUnionAdmin},
		{"pourashava mayor", valueobject.TierPourashavaMayor, domain.ClaimRouteUnionAdmin},

		// Above the union: no single union admin has standing over a seat-wide
		// office, so the super admin answers these.
		{"upazila chairman", valueobject.TierUpazilaChairman, domain.ClaimRouteSuperAdmin},
		{"upazila vice chairman", valueobject.TierUpazilaViceChairman, domain.ClaimRouteSuperAdmin},
		{"mp", valueobject.TierMP, domain.ClaimRouteSuperAdmin},
		{"minister", valueobject.TierMinister, domain.ClaimRouteSuperAdmin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.RouteClaim(tt.tier); got != tt.want {
				t.Fatalf("RouteClaim(%s) = %s, want %s", tt.tier, got, tt.want)
			}
		})
	}
}

// TestRouteClaimMatchesAssignmentSplit pins the two routings together: every tier
// that IsUnionLevel() must route to the union admin and no other. If these ever
// drift, an official could be verified by someone with no authority to assign
// them work — or vice versa.
func TestRouteClaimMatchesAssignmentSplit(t *testing.T) {
	all := []valueobject.Tier{
		valueobject.TierWardMember, valueobject.TierWomenMember, valueobject.TierUnionChairman,
		valueobject.TierPourashavaCouncillor, valueobject.TierPourashavaMayor,
		valueobject.TierUpazilaChairman, valueobject.TierUpazilaViceChairman,
		valueobject.TierMP, valueobject.TierMinister,
	}
	for _, tier := range all {
		want := domain.ClaimRouteSuperAdmin
		if tier.IsUnionLevel() {
			want = domain.ClaimRouteUnionAdmin
		}
		if got := domain.RouteClaim(tier); got != want {
			t.Errorf("RouteClaim(%s) = %s, but IsUnionLevel()=%v implies %s", tier, got, tier.IsUnionLevel(), want)
		}
	}
}

func TestClaimStatusValid(t *testing.T) {
	for _, s := range []domain.ClaimStatus{domain.ClaimPending, domain.ClaimApproved, domain.ClaimRejected} {
		if !s.Valid() {
			t.Errorf("ClaimStatus(%q).Valid() = false, want true", s)
		}
	}
	for _, s := range []domain.ClaimStatus{"", "pending", "Approved ", "nonsense"} {
		if s.Valid() {
			t.Errorf("ClaimStatus(%q).Valid() = true, want false", s)
		}
	}
}

// TestClaimDecide pins the legal transitions: a claim is decided exactly once.
// Re-deciding an approved claim would silently transfer an office between people.
func TestClaimDecide(t *testing.T) {
	tests := []struct {
		name    string
		current domain.ClaimStatus
		to      domain.ClaimStatus
		wantErr error
	}{
		{"pending approves", domain.ClaimPending, domain.ClaimApproved, nil},
		{"pending rejects", domain.ClaimPending, domain.ClaimRejected, nil},
		{"approved cannot be re-decided", domain.ClaimApproved, domain.ClaimRejected, domain.ErrClaimNotPending},
		{"rejected cannot be re-decided", domain.ClaimRejected, domain.ClaimApproved, domain.ErrClaimNotPending},
		{"cannot decide back to pending", domain.ClaimPending, domain.ClaimPending, domain.ErrInvalidClaimStatus},
		{"cannot decide to nonsense", domain.ClaimPending, "nonsense", domain.ErrInvalidClaimStatus},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.DecideClaim(tt.current, tt.to)
			if err != tt.wantErr {
				t.Fatalf("DecideClaim(%s -> %s) = %v, want %v", tt.current, tt.to, err, tt.wantErr)
			}
		})
	}
}

func TestRoleValidIncludesSuperAdmin(t *testing.T) {
	if !domain.RoleSuperAdmin.Valid() {
		t.Fatal("super_admin must be a valid role")
	}
	if domain.RoleSuperAdmin == domain.RoleAdmin {
		t.Fatal("super_admin must be distinct from admin — it is not a superset")
	}
}
