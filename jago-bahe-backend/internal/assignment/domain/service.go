package domain

import "jago-bahe-backend/internal/shared/domain/valueobject"

// Service holds the routing rule — the assignment context's authority. It is
// pure: no persistence, no clock beyond what callers pass in.
type Service struct{}

// NewService constructs the assignment domain service.
func NewService() *Service { return &Service{} }

// Route decides how a problem pointed at the given tier is assigned. Union-level
// tiers are handled by that union's admin alone; above-union tiers go to the super
// admin, advised by the union admins of the upazila (upazila chairman /
// vice-chairman) or of the whole seat (MP / minister). See Tier.IsUnionLevel
// (Scaffold Spec §2).
//
// The split itself is unchanged since B4 — what changed in B20 is only what
// happens on the far side of it. Above-union used to mean a binding vote of the
// admins; it now means the super admin decides with their advice in front of them
// (CLAUDE.md A.3.8). The five union-level tiers are untouched.
func (Service) Route(tier valueobject.Tier) Route {
	if tier.IsUnionLevel() {
		return Route{Kind: RouteUnion}
	}
	scope := ScopeSeat
	switch tier {
	case valueobject.TierUpazilaChairman, valueobject.TierUpazilaViceChairman:
		scope = ScopeUpazila
	}
	return Route{Kind: RouteSuperAdmin, Scope: scope}
}
