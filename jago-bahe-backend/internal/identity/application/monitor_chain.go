package application

import (
	"context"

	"jago-bahe-backend/internal/identity/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// MonitorChain returns the officials who monitor the given official, nearest rung
// first, at most maxRungs long: a ward or union official is monitored by the
// enclosing upazila's chairman, an upazila official by the seat's MP, and the MP
// by nobody. It is the accountability ladder of Concept §3 read upward, and it is
// what makes an escalation land on a named person instead of an integer.
//
// It exists as one function for the same reason AdminUnion does: the walk crosses
// the officials directory and the area tree, identity owns both questions, and a
// second copy is how the assignment context's monitor and the worker's escalation
// ladder would come to disagree about who supervises whom. B4's one-rung MonitorFor
// is now this function with maxRungs = 1.
//
// Three behaviours are load-bearing:
//
//   - It DEGRADES rather than failing. An unresolvable area, a vacant office or an
//     empty upazila simply ends the chain, exactly as MonitorFor has always
//     returned ("", nil). The worker walks this for every open case in the seat on
//     every tick, and one malformed row must not abort the whole scan.
//   - It DEDUPES, and never returns the official themselves. Offices can be vacant
//     or double-held, which collapses two rungs onto one person; a chain listing
//     them twice would open two observations on one official for one case.
//   - It reads the directory ONCE. The per-rung lookup is over that snapshot, not a
//     repeated List — the pre-B19 adapter re-listed every official per rung.
func MonitorChain(
	ctx context.Context,
	officials domain.OfficialRepository,
	areas areadomain.Repository,
	officialID string,
	maxRungs int,
) ([]string, error) {
	if maxRungs <= 0 {
		return nil, nil
	}
	directory, err := officials.List(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]domain.Official, len(directory))
	for _, o := range directory {
		byID[o.ID] = o
	}
	byTierArea := func(tier valueobject.Tier, areaID string) string {
		for _, o := range directory {
			if o.Tier == tier && o.AreaID.String() == areaID {
				return o.ID
			}
		}
		return ""
	}
	byTier := func(tier valueobject.Tier) string {
		for _, o := range directory {
			if o.Tier == tier {
				return o.ID
			}
		}
		return ""
	}

	seen := map[string]bool{officialID: true}
	var chain []string
	current := officialID

	for len(chain) < maxRungs {
		o, ok := byID[current]
		if !ok {
			break
		}
		area, err := areas.GetByID(ctx, o.AreaID)
		if err != nil {
			break // no resolvable area → the ladder ends here rather than erroring
		}

		var next string
		switch area.Level {
		case areadomain.LevelWard, areadomain.LevelUnion:
			upazila, err := areadomain.UpazilaOfArea(ctx, areas, area)
			if err != nil || upazila == "" {
				return chain, nil
			}
			next = byTierArea(valueobject.TierUpazilaChairman, upazila)
		case areadomain.LevelUpazila:
			next = byTier(valueobject.TierMP)
		default: // seat (MP / minister) — the top of the ladder
			next = ""
		}

		// A vacant office ends the chain here. It deliberately does NOT skip to the
		// rung above: an absent upazila chairman must not silently promote the MP
		// into the direct-monitor slot, because that would put a case in front of
		// the MP a full window early and mislabel which rung it had climbed to.
		if next == "" || seen[next] {
			break
		}
		chain = append(chain, next)
		seen[next] = true
		current = next
	}
	return chain, nil
}
