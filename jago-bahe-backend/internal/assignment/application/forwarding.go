package application

import (
	"context"

	"jago-bahe-backend/internal/assignment/domain"
)

// This file holds what the two above-union forwarding surfaces share: finding the
// reports that are waiting for a forward, and decorating them with the union
// admins' advice. See CLAUDE.md A.3.8.

// ForwardingCandidate is one above-union report awaiting the super admin, with
// everything either surface needs to judge it: what the reporter asked for, what
// the union admins advise, and (on an adviser's own list) what this admin said.
type ForwardingCandidate struct {
	Problem ProblemView
	Scope   domain.AdviceScope

	Suggestions []domain.Suggestion
	// TopOfficialID is the official the advisers most agree on, and TopCount how
	// many said so. Both are empty/zero when nobody advised — AND when the advisers
	// tied, because a tie is genuine disagreement and breaking it arbitrarily would
	// invent a consensus (see domain.TopSuggestion).
	TopOfficialID string
	TopCount      int

	// MySuggestion is the calling admin's own advice, nil when they have not
	// advised. It is composed only on the advisers' list; the super admin does not
	// advise, so it is always nil on theirs.
	MySuggestion *domain.Suggestion
}

// forwardingCandidates returns every above-union report still awaiting a forward,
// decorated with the advice on it — one batch query for the whole page, never one
// per row (A.3.4).
//
// It takes no caller: both surfaces need the same set, and they differ only in
// what they then filter it to. Keeping the caller out means the shared half cannot
// grow a scope rule that one surface silently depends on.
func forwardingCandidates(ctx context.Context, problems Problems, officials Officials, repo domain.Repository, svc *domain.Service) ([]ForwardingCandidate, error) {
	assignable, err := problems.ListAssignable(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ForwardingCandidate, 0, len(assignable))
	ids := make([]string, 0, len(assignable))
	for _, p := range assignable {
		pointed, err := officials.Get(ctx, p.PointedOfficialID)
		if err != nil {
			// The pointed official has vanished from the directory. Routing cannot be
			// decided, so the report belongs to neither queue rather than defaulting
			// into one — a default here would put an unrouted report in front of
			// whichever surface guessed first.
			continue
		}
		route := svc.Route(pointed.Tier)
		if route.Kind != domain.RouteSuperAdmin {
			continue // union-level: AssignWithinUnion's queue, not this one
		}
		p.Routing = string(route.Kind)
		out = append(out, ForwardingCandidate{Problem: p, Scope: route.Scope})
		ids = append(ids, p.ID)
	}
	if len(out) == 0 {
		return nil, nil
	}

	// A failed decoration degrades to "nobody has advised" rather than failing the
	// whole queue (A.3.4 rule 3): the super admin must still be able to see the
	// reports waiting on them if the advice query breaks.
	byProblem, err := repo.SuggestionsByProblems(ctx, ids)
	if err != nil {
		byProblem = nil
	}
	for i := range out {
		s := byProblem[out[i].Problem.ID]
		out[i].Suggestions = s
		out[i].TopOfficialID, out[i].TopCount = domain.TopSuggestion(s)
	}
	return out, nil
}
