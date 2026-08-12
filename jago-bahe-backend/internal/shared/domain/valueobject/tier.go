package valueobject

// Tier is an elected official's rung on the seat's ladder. Which tier a problem
// points to decides who assigns it: union-level tiers are handled by that union's
// admin alone; above-union tiers go to an admin vote (Scaffold Spec §2).
type Tier string

const (
	TierWardMember           Tier = "ward_member"
	TierWomenMember          Tier = "women_member"
	TierUnionChairman        Tier = "union_chairman"
	TierPourashavaCouncillor Tier = "pourashava_councillor"
	TierPourashavaMayor      Tier = "pourashava_mayor"
	TierUpazilaChairman      Tier = "upazila_chairman"
	TierUpazilaViceChairman  Tier = "upazila_vice_chairman"
	TierMP                   Tier = "mp"
	TierMinister             Tier = "minister"
)

// unionLevel is the set of tiers a single union admin decides alone.
var unionLevel = map[Tier]bool{
	TierWardMember:           true,
	TierWomenMember:          true,
	TierUnionChairman:        true,
	TierPourashavaCouncillor: true,
	TierPourashavaMayor:      true,
}

// IsUnionLevel reports whether this tier is decided by a single union admin
// (true) or routed to an admin vote (false).
func (t Tier) IsUnionLevel() bool { return unionLevel[t] }

// Valid reports whether t is a known tier.
func (t Tier) Valid() bool {
	switch t {
	case TierWardMember, TierWomenMember, TierUnionChairman, TierPourashavaCouncillor,
		TierPourashavaMayor, TierUpazilaChairman, TierUpazilaViceChairman, TierMP, TierMinister:
		return true
	}
	return false
}
