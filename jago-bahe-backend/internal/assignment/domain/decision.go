// Package domain (assignment) owns how a validated problem reaches an official:
// the routing decision (a single union admin vs. the super admin, advised by the
// union admins) and the advisory tally that informs the latter. Pure logic — no
// database, HTTP, or other contexts. Routing is backend-owned authority (Scaffold
// Spec §2); the http/UI layers only display the result.
package domain

// RouteKind is how a problem's assignment is decided, chosen from the pointed
// official's tier.
type RouteKind string

const (
	// RouteUnion — a union-level tier: that union's admin decides alone.
	RouteUnion RouteKind = "union"
	// RouteSuperAdmin — above the union: the super admin forwards it, advised by
	// the union admins of the relevant scope.
	//
	// This was RouteVote until B20, when the binding admin vote (quorum Q, majority
	// within window W, no-majority fallback to the Union Chairman) was replaced by
	// advice plus one accountable decider. The constant was renamed rather than
	// re-pointed so that every call site had to be visited (CLAUDE.md A.3.8).
	RouteSuperAdmin RouteKind = "super_admin"
)

// AdviceScope is the set of union admins entitled to advise on an above-union
// report: the admins of one upazila, or every admin in the seat.
//
// It was VoteScope until B20. The scope rule itself is unchanged — an upazila-tier
// report is advised by that upazila's admins, a seat-tier report by all of them —
// because who has standing to speak on a problem did not change when the decision
// moved; only what their input does changed.
type AdviceScope string

const (
	ScopeUpazila AdviceScope = "upazila"
	ScopeSeat    AdviceScope = "seat"
)

// Valid reports whether s is a known scope.
func (s AdviceScope) Valid() bool { return s == ScopeUpazila || s == ScopeSeat }

// Route is the outcome of the routing decision. Scope is meaningful only when
// Kind is RouteSuperAdmin.
type Route struct {
	Kind  RouteKind
	Scope AdviceScope
}
