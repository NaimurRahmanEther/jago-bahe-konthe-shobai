package application

import (
	"context"

	"jago-bahe-backend/internal/assignment/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
)

// resolveUnion returns the union an area belongs to, mapping an unknown area onto
// this context's sentinel: an assignment only ever resolves a union for a problem
// it already loaded, so a missing area here means the problem's location is gone.
func resolveUnion(ctx context.Context, areas areadomain.Repository, areaID string) (string, error) {
	union, err := areadomain.ResolveUnion(ctx, areas, areaID)
	if err != nil {
		return "", domain.ErrProblemNotFound
	}
	return union, nil
}

// containsID reports whether id is present in ids.
func containsID(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
