package domain_test

import (
	"testing"

	"jago-bahe-backend/internal/assignment/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// TestRoute covers all nine tiers. The five union-level rows are the regression
// that matters most in B20: above-union assignment moved to the super admin, and
// nothing about the union path was supposed to change with it.
//
// TestTally went with the vote. It exercised quorum Q and majority arithmetic
// that no longer exists — above-union forwarding is decided by the super admin on
// advice that settles nothing, so there is no tally to compute (CLAUDE.md A.3.8).
// The advisory tally's own rules are pinned by forwarding_test.go.
func TestRoute(t *testing.T) {
	svc := domain.NewService()
	tests := []struct {
		name      string
		tier      valueobject.Tier
		wantKind  domain.RouteKind
		wantScope domain.AdviceScope
	}{
		{"ward member is union-level", valueobject.TierWardMember, domain.RouteUnion, ""},
		{"women member is union-level", valueobject.TierWomenMember, domain.RouteUnion, ""},
		{"union chairman is union-level", valueobject.TierUnionChairman, domain.RouteUnion, ""},
		{"pourashava councillor is union-level", valueobject.TierPourashavaCouncillor, domain.RouteUnion, ""},
		{"pourashava mayor is union-level", valueobject.TierPourashavaMayor, domain.RouteUnion, ""},
		{"upazila chairman goes to the super admin, advised at upazila scope",
			valueobject.TierUpazilaChairman, domain.RouteSuperAdmin, domain.ScopeUpazila},
		{"upazila vice-chairman goes to the super admin, advised at upazila scope",
			valueobject.TierUpazilaViceChairman, domain.RouteSuperAdmin, domain.ScopeUpazila},
		{"MP goes to the super admin, advised at seat scope",
			valueobject.TierMP, domain.RouteSuperAdmin, domain.ScopeSeat},
		{"minister goes to the super admin, advised at seat scope",
			valueobject.TierMinister, domain.RouteSuperAdmin, domain.ScopeSeat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.Route(tt.tier)
			if got.Kind != tt.wantKind {
				t.Fatalf("Route(%s).Kind = %q, want %q", tt.tier, got.Kind, tt.wantKind)
			}
			if got.Kind == domain.RouteSuperAdmin && got.Scope != tt.wantScope {
				t.Fatalf("Route(%s).Scope = %q, want %q", tt.tier, got.Scope, tt.wantScope)
			}
		})
	}
}
