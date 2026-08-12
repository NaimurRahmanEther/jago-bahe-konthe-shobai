package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
)

// ensureUnionAdmin enforces that the acting admin belongs to the problem's union.
//
// This is not belt-and-braces over the middleware: auth.RequireAdmin only checks
// the role claim and knows nothing about geography, so without this check any
// admin on the platform could take down any union's reports — which is precisely
// the capture the union scoping exists to prevent. The assignment context enforces
// the same rule the same way (assign_within_union.go).
func ensureUnionAdmin(ctx context.Context, areas areadomain.Repository, admins Admins, adminAccountID, problemAreaID string) error {
	problemUnion, err := areadomain.ResolveUnion(ctx, areas, problemAreaID)
	if err != nil {
		return domain.ErrAreaNotFound
	}
	adminUnion, err := admins.UnionOf(ctx, adminAccountID)
	if err != nil {
		return err
	}
	if adminUnion == "" || adminUnion != problemUnion {
		return domain.ErrNotUnionAdmin
	}
	return nil
}
