package application

import (
	"context"

	"jago-bahe-backend/internal/assignment/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
)

// ListQueue is the acting admin's assignment queue: the problems in their own
// union that they may still forward to an official. Like the screening queue it is
// a derived view over problem status, not a table of its own.
//
// It carries BOTH Reported and Validated problems. Before B17 it was Validated
// alone, because only a validated problem could be assigned — which left the admin
// blind between approving a report and its reaching V, and the queue behaved as a
// binary: absent, then suddenly present. Now the community's validation count
// travels with each row and the admin watches it climb, deciding for themselves
// when the report is trustworthy enough to forward. The count gates nothing
// (A.3.1); it is evidence.
//
// It carries UNION-ROUTED problems only. An above-union report (MP, upazila
// chairman or vice chairman, minister) is the super admin's to forward, advised by
// the union admins — it appears on ListMyForwarding instead (B20, A.3.8).
//
// Assigning moves a problem to Assigned, out of both statuses, so it drops off the
// queue either way.
type ListQueue struct {
	problems  Problems
	officials Officials
	admins    Admins
	areas     areadomain.Repository
	svc       *domain.Service
}

// NewListQueue wires the use case.
func NewListQueue(p Problems, o Officials, ad Admins, areas areadomain.Repository, svc *domain.Service) *ListQueue {
	return &ListQueue{problems: p, officials: o, admins: ad, areas: areas, svc: svc}
}

// Execute returns the queue for one admin. An admin with no union on record gets
// an error rather than everyone's queue — failing closed, mirroring ListModeration.
// This route used to ignore the caller entirely and hand every admin every union's
// problems; scoping existed only at the action step (AssignWithinUnion's
// ensureUnionAdmin), so the list leaked what the action would have refused.
func (uc *ListQueue) Execute(ctx context.Context, adminAccountID string) ([]ProblemView, error) {
	adminUnion, err := uc.admins.UnionOf(ctx, adminAccountID)
	if err != nil {
		return nil, err
	}
	if adminUnion == "" {
		return nil, domain.ErrNotUnionAdmin
	}

	candidates, err := uc.problems.ListAssignable(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by union in the use case rather than the query: a problem's union is a
	// tree walk (a ward resolves to its parent), which the problem repository's SQL
	// cannot express. Same shape as ListModeration.
	out := make([]ProblemView, 0, len(candidates))
	for _, p := range candidates {
		union, err := areadomain.ResolveUnion(ctx, uc.areas, p.AreaID)
		if err != nil {
			continue // an orphaned location cannot be scoped; leave it out of every queue
		}
		if union != adminUnion {
			continue
		}

		// Routing is this context's own authority, resolved here so the client never
		// has to guess it (A.4.6). An official who has vanished from the directory
		// leaves the row unrouted rather than dropping the problem — the admin can
		// still see it and override the pointed official.
		if pointed, err := uc.officials.Get(ctx, p.PointedOfficialID); err == nil {
			route := uc.svc.Route(pointed.Tier)
			// An above-union report is not this admin's to assign — AssignWithinUnion
			// answers ErrWrongRoute for it — so it does not belong on the list of what
			// they may assign. Leaving it here is the exact shape of the leak B17 fixed
			// at the union boundary: a queue that shows what the action refuses. It
			// reaches them instead through ListMyForwarding, where they advise the
			// super admin (A.3.8), and that list is scoped by ADVICE SCOPE — the
			// upazila or the seat — which is a wider and different set than this one.
			if route.Kind == domain.RouteSuperAdmin {
				continue
			}
			p.Routing = string(route.Kind)
		}
		out = append(out, p)
	}
	return out, nil
}
