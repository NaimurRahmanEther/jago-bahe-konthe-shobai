package domain

import (
	"context"

	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// UnionOfArea returns the union an area sits in: a ward resolves to its parent
// union, and every other level is returned as-is (a union is its own union; a
// seat or upazila is returned unchanged rather than erroring, since callers ask
// this of problem locations, which are always at ward or union level).
//
// This is the pure half — it needs no repository. Callers holding only an area id
// want ResolveUnion instead.
func UnionOfArea(a *Area) string {
	if a.Level == LevelWard {
		return a.ParentID.String()
	}
	return a.ID.String()
}

// UpazilaOfArea returns the upazila an area sits in: an upazila is its own, a
// union resolves to its parent, and a ward walks up through its union. A seat (the
// root) has no enclosing upazila and resolves to "".
//
// Unlike UnionOfArea this needs the repository, because a ward is two hops from
// its upazila. Every failure collapses to "" rather than an error, matching
// ResolveUnion: callers treat "" as "no enclosing upazila" and stop there.
//
// It is the second hop of the monitor ladder (a union official is monitored by
// the upazila chairman), which is why it lives here beside the union walk rather
// than being re-derived by the contexts that climb it.
func UpazilaOfArea(ctx context.Context, areas Repository, a *Area) (string, error) {
	switch a.Level {
	case LevelUpazila:
		return a.ID.String(), nil
	case LevelUnion:
		return a.ParentID.String(), nil
	case LevelWard:
		union, err := areas.GetByID(ctx, a.ParentID)
		if err != nil {
			return "", nil
		}
		return union.ParentID.String(), nil
	default: // seat (the root) — nothing encloses it
		return "", nil
	}
}

// ResolveUnion loads an area by id and returns the union it sits in, or
// ErrAreaNotFound if the id is unknown.
//
// Union membership is the scoping question four contexts ask — problem (who may
// validate), suggestion (who may suggest), assignment (which admin decides), and
// resolution (who may judge a blocker) — and B9 added a fifth, screening. Each
// context wraps this and maps ErrAreaNotFound onto its own sentinel, so the tree
// walk lives here once rather than being re-derived per context.
func ResolveUnion(ctx context.Context, areas Repository, areaID string) (string, error) {
	a, err := areas.GetByID(ctx, valueobject.AreaID(areaID))
	if err != nil {
		return "", ErrAreaNotFound
	}
	return UnionOfArea(a), nil
}
