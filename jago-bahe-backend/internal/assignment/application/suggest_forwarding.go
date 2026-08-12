package application

import (
	"context"
	"sort"
	"strings"

	"jago-bahe-backend/internal/assignment/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// SuggestForwarding records one union admin's advice on where an above-union
// report should be forwarded (B20).
//
// It is advice and nothing more. There is no quorum to reach, no window to beat,
// and no outcome it can produce: the super admin decides, and may forward at any
// count including none (A.3.8). What this buys is that the decision is made in
// the open — the tally is public, so a forward that goes against the seat's admins
// is visible as such.
//
// It replaced OpenAdminVote and CastAdminVote, whose one worthwhile guard it
// keeps: the caller must be an admin entitled to speak on THIS problem. RequireAdmin
// is role-only and knows nothing about geography, so without this check any admin
// on the platform could advise on any upazila's report — and B11 had to add exactly
// this guard to OpenAdminVote after it shipped without one.
type SuggestForwarding struct {
	assignments domain.Repository
	problems    Problems
	officials   Officials
	admins      Admins
	svc         *domain.Service
	audit       auditdomain.Repository
}

// NewSuggestForwarding wires the use case.
func NewSuggestForwarding(r domain.Repository, p Problems, o Officials, ad Admins, svc *domain.Service, audit auditdomain.Repository) *SuggestForwarding {
	return &SuggestForwarding{assignments: r, problems: p, officials: o, admins: ad, svc: svc, audit: audit}
}

// Execute records or replaces the admin's advice on a problem.
func (uc *SuggestForwarding) Execute(ctx context.Context, problemID, adminAccountID, suggestedOfficialID, reason string) (*domain.Suggestion, error) {
	p, err := uc.problems.Get(ctx, problemID)
	if err != nil {
		return nil, err // domain.ErrProblemNotFound
	}
	if !p.EligibleForAssignment {
		return nil, domain.ErrNotAssignable
	}

	pointed, err := uc.officials.Get(ctx, p.PointedOfficialID)
	if err != nil {
		return nil, err
	}
	route := uc.svc.Route(pointed.Tier)
	if route.Kind != domain.RouteSuperAdmin {
		return nil, domain.ErrWrongRoute // union-level: that union's admin decides alone
	}

	// The advised official must exist AND be above-union. Advising a union-level
	// official would be advice the super admin could not act on: forwarding there
	// would take a report the union's own admin should have decided and route it
	// through the seat's appointed watcher instead.
	advised, err := uc.officials.Get(ctx, suggestedOfficialID)
	if err != nil {
		return nil, err // domain.ErrOfficialNotFound
	}
	if uc.svc.Route(advised.Tier).Kind != domain.RouteSuperAdmin {
		return nil, domain.ErrWrongRoute
	}

	eligible, err := uc.admins.EligibleAdvisers(ctx, route.Scope, p.AreaID)
	if err != nil {
		return nil, err
	}
	if !containsID(eligible, adminAccountID) {
		return nil, domain.ErrNotEligibleAdviser
	}

	s := domain.NewSuggestion(problemID, adminAccountID, suggestedOfficialID, strings.TrimSpace(reason))
	if err := uc.assignments.UpsertSuggestion(ctx, s); err != nil {
		return nil, err
	}
	// Audited per change, not per admin: an admin who moves from one official to
	// another has done something the record should show, and the upsert leaves only
	// the latest row behind.
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry(
		"problem", problemID, adminAccountID, "forwarding_suggested", s.Reason))
	return s, nil
}

// ListMyForwarding is the union admin's advisory list: the above-union reports
// this admin may speak on, with the public tally and their own advice.
//
// It is scoped by ADVICE SCOPE, not by the admin's union — an upazila-tier report
// is advised by every admin in that upazila, and a seat-tier one by every admin in
// the seat. That is the same scope rule the vote used, kept because who has
// standing to speak on a problem did not change when the decision moved.
type ListMyForwarding struct {
	assignments domain.Repository
	problems    Problems
	officials   Officials
	admins      Admins
	svc         *domain.Service
}

// NewListMyForwarding wires the use case.
func NewListMyForwarding(r domain.Repository, p Problems, o Officials, ad Admins, svc *domain.Service) *ListMyForwarding {
	return &ListMyForwarding{assignments: r, problems: p, officials: o, admins: ad, svc: svc}
}

// Execute returns the reports this admin may advise on. It is a read and is never
// audited (A.4.4).
func (uc *ListMyForwarding) Execute(ctx context.Context, adminAccountID string) ([]ForwardingCandidate, error) {
	if adminAccountID == "" {
		return nil, domain.ErrNotEligibleAdviser
	}
	candidates, err := forwardingCandidates(ctx, uc.problems, uc.officials, uc.assignments, uc.svc)
	if err != nil {
		return nil, err
	}

	out := make([]ForwardingCandidate, 0, len(candidates))
	for _, c := range candidates {
		eligible, err := uc.admins.EligibleAdvisers(ctx, c.Scope, c.Problem.AreaID)
		if err != nil {
			continue // cannot establish standing → leave it out, never let it in
		}
		if !containsID(eligible, adminAccountID) {
			continue
		}
		for i := range c.Suggestions {
			if c.Suggestions[i].AdminAccountID == adminAccountID {
				mine := c.Suggestions[i]
				c.MySuggestion = &mine
				break
			}
		}
		out = append(out, c)
	}
	sortForwarding(out)
	return out, nil
}

// sortForwarding puts the reports with the most advice first, then the oldest
// waiting. Ordering is the backend's, so no surface has to decide what is urgent
// (A.5 rule 7).
func sortForwarding(rows []ForwardingCandidate) {
	sort.SliceStable(rows, func(i, j int) bool {
		if len(rows[i].Suggestions) != len(rows[j].Suggestions) {
			return len(rows[i].Suggestions) > len(rows[j].Suggestions)
		}
		return rows[i].Problem.ID < rows[j].Problem.ID
	})
}
