package application

import (
	"context"

	"jago-bahe-backend/internal/identity/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
)

// AdminUnion resolves the union an admin account is scoped to, by walking their
// account → their linked directory official → that official's area → its union.
//
// The walk exists because admins carry no union of their own: accounts.union_id
// is a resident field and is NULL for them, so their scope is inherited from the
// office they are bound to. An official may sit at ward level (the seeded ward
// member does), which is why the last hop resolves through the area tree rather
// than reading area_id directly.
//
// Every failure collapses to "" rather than an error, matching the behaviour the
// assignment context has relied on since B4: callers treat "" as "belongs to no
// union" and refuse the action, so an admin whose scope cannot be resolved is
// denied rather than admitted.
//
// It is exported because the composition root's adapters answer the same question
// on behalf of the problem and assignment contexts. Identity owns accounts and
// officials, so this is identity's question to answer — once, here.
func AdminUnion(
	ctx context.Context,
	accounts domain.AccountRepository,
	officials domain.OfficialRepository,
	areas areadomain.Repository,
	adminAccountID string,
) (string, error) {
	acc, err := accounts.GetByID(ctx, adminAccountID)
	if err != nil {
		return "", nil
	}
	if acc.OfficialID == "" {
		return "", nil
	}
	o, err := officials.GetByID(ctx, acc.OfficialID)
	if err != nil {
		return "", nil
	}
	union, err := areadomain.ResolveUnion(ctx, areas, o.AreaID.String())
	if err != nil {
		return "", nil
	}
	return union, nil
}
