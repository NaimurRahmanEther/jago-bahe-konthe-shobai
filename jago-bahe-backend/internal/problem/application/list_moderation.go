package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
)

// ListModeration is the screening queue: the reports in the acting admin's own
// union that are awaiting their approve/reject decision. Like the assignment queue
// it is a derived view over problem status, not a table of its own.
//
// Every report here is PendingApproval — invisible to the public until this admin
// approves it. Reports do wait on the admin: an unopened queue leaves them hidden,
// which is the accountable cost of the gate. Post-publication takedowns of already
// public reports (Reported/Validated) are reached from the problem itself via the
// reject endpoint, not from this list.
type ListModeration struct {
	problems domain.Repository
	areas    areadomain.Repository
	admins   Admins
}

// NewListModeration wires the use case.
func NewListModeration(p domain.Repository, a areadomain.Repository, ad Admins) *ListModeration {
	return &ListModeration{problems: p, areas: a, admins: ad}
}

// Execute returns the reports this admin must screen. An admin with no union on
// record gets an error rather than everyone's queue — failing closed, since the
// alternative hands one union's admin the screening list for the whole seat.
func (uc *ListModeration) Execute(ctx context.Context, adminAccountID string) ([]domain.Problem, error) {
	adminUnion, err := uc.admins.UnionOf(ctx, adminAccountID)
	if err != nil {
		return nil, err
	}
	if adminUnion == "" {
		return nil, domain.ErrNotUnionAdmin
	}

	// Reports awaiting a screening decision. Only PendingApproval: an approved
	// report is public and off this queue, and a takedown of an already-public
	// report is initiated from the problem itself, not here.
	candidates, err := uc.problems.List(ctx, domain.Filter{
		Statuses: []domain.Status{domain.StatusPendingApproval},
	})
	if err != nil {
		return nil, err
	}

	// Filter by union in the use case rather than the query: a problem's union is
	// a tree walk (a ward resolves to its parent), which SQL here cannot express.
	out := make([]domain.Problem, 0, len(candidates))
	for i := range candidates {
		union, err := areadomain.ResolveUnion(ctx, uc.areas, candidates[i].Location.AreaID.String())
		if err != nil {
			continue // an orphaned location cannot be scoped; leave it out of every queue
		}
		if union == adminUnion {
			out = append(out, candidates[i])
		}
	}
	return out, nil
}
