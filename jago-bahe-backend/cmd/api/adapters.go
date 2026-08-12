package main

import (
	"context"
	"errors"

	assignmentapp "jago-bahe-backend/internal/assignment/application"
	assignmentdomain "jago-bahe-backend/internal/assignment/domain"
	identityapp "jago-bahe-backend/internal/identity/application"
	identitydomain "jago-bahe-backend/internal/identity/domain"
	problemapp "jago-bahe-backend/internal/problem/application"
	problemdomain "jago-bahe-backend/internal/problem/domain"
	resolutionapp "jago-bahe-backend/internal/resolution/application"
	resolutiondomain "jago-bahe-backend/internal/resolution/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditapp "jago-bahe-backend/internal/shared/audit/application"
	"jago-bahe-backend/internal/shared/domain/valueobject"
	suggestionapp "jago-bahe-backend/internal/suggestion/application"
	suggestiondomain "jago-bahe-backend/internal/suggestion/domain"
)

// identityAdapter bridges the problem context's Identity port to the identity
// repositories, so the problem context stays decoupled from identity internals.
type identityAdapter struct {
	accounts  identitydomain.AccountRepository
	officials identitydomain.OfficialRepository
}

var _ problemapp.Identity = identityAdapter{}

func (a identityAdapter) Voter(ctx context.Context, accountID string) (problemapp.Voter, error) {
	acc, err := a.accounts.GetByID(ctx, accountID)
	if err != nil {
		return problemapp.Voter{}, err
	}
	return problemapp.Voter{
		IsResident: acc.Role == identitydomain.RoleResident,
		Verified:   acc.Verified,
		UnionID:    acc.UnionID.String(),
	}, nil
}

func (a identityAdapter) OfficialExists(ctx context.Context, officialID string) (bool, error) {
	_, err := a.officials.GetByID(ctx, officialID)
	if errors.Is(err, identitydomain.ErrOfficialNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// suggestionIdentityAdapter bridges the suggestion context's Identity port to the
// identity accounts (a distinct port type from the problem context's).
type suggestionIdentityAdapter struct {
	accounts identitydomain.AccountRepository
}

var _ suggestionapp.Identity = suggestionIdentityAdapter{}

func (a suggestionIdentityAdapter) Voter(ctx context.Context, accountID string) (suggestionapp.Voter, error) {
	acc, err := a.accounts.GetByID(ctx, accountID)
	if err != nil {
		return suggestionapp.Voter{}, err
	}
	return suggestionapp.Voter{
		IsResident: acc.Role == identitydomain.RoleResident,
		Verified:   acc.Verified,
		UnionID:    acc.UnionID.String(),
	}, nil
}

// problemAreaAdapter bridges the suggestion context's Problems port to the
// problem repository, exposing only the problem's area (and its existence).
type problemAreaAdapter struct {
	problems problemdomain.Repository
}

var _ suggestionapp.Problems = problemAreaAdapter{}

func (a problemAreaAdapter) AreaID(ctx context.Context, problemID string) (valueobject.AreaID, error) {
	p, err := a.problems.GetByID(ctx, problemID)
	if errors.Is(err, problemdomain.ErrProblemNotFound) {
		return "", suggestiondomain.ErrProblemNotFound
	}
	if err != nil {
		return "", err
	}
	return p.Location.AreaID, nil
}

// --- assignment context adapters (B4) ---

// assignmentProblemAdapter bridges the assignment context's Problems port to the
// problem repository: reading a problem's routing facts, listing the validated
// queue, and flipping a problem to Assigned.
type assignmentProblemAdapter struct {
	problems problemdomain.Repository
}

var _ assignmentapp.Problems = assignmentProblemAdapter{}

func (a assignmentProblemAdapter) Get(ctx context.Context, problemID string) (assignmentapp.ProblemView, error) {
	p, err := a.problems.GetByID(ctx, problemID)
	if errors.Is(err, problemdomain.ErrProblemNotFound) {
		return assignmentapp.ProblemView{}, assignmentdomain.ErrProblemNotFound
	}
	if err != nil {
		return assignmentapp.ProblemView{}, err
	}
	return toProblemView(p), nil
}

// ListAssignable returns the problems still forwardable to an official. Reported
// is in the set alongside Validated: since B17 an approved report is assignable at
// once, and the admin judges its validation count rather than waiting on V
// (A.3.1). The two statuses here must stay in step with
// problemdomain.Status.EligibleForAssignment, which is what actually gates the
// assign call — this filter only decides what the queue shows.
func (a assignmentProblemAdapter) ListAssignable(ctx context.Context) ([]assignmentapp.ProblemView, error) {
	ps, err := a.problems.List(ctx, problemdomain.Filter{Statuses: []problemdomain.Status{
		problemdomain.StatusReported,
		problemdomain.StatusValidated,
	}})
	if err != nil {
		return nil, err
	}
	out := make([]assignmentapp.ProblemView, 0, len(ps))
	for i := range ps {
		out = append(out, toProblemView(&ps[i]))
	}
	return out, nil
}

func (a assignmentProblemAdapter) MarkAssigned(ctx context.Context, problemID string) error {
	err := a.problems.SetStatus(ctx, problemID, problemdomain.StatusAssigned)
	if errors.Is(err, problemdomain.ErrProblemNotFound) {
		return assignmentdomain.ErrProblemNotFound
	}
	return err
}

// toProblemView translates a problem into the assignment context's view of it.
// This is where the status vocabulary is interpreted — EligibleForAssignment is
// computed here, in the composition root, so the assignment context can gate on a
// bool without importing the problem package (A.4.1). It previously duplicated the
// rule as a local "Validated" string const instead, which is precisely how one
// context's copy of another's vocabulary goes stale.
func toProblemView(p *problemdomain.Problem) assignmentapp.ProblemView {
	return assignmentapp.ProblemView{
		ID:                    p.ID,
		Title:                 p.Title,
		Status:                string(p.Status),
		Address:               p.Location.Address,
		PointedOfficialID:     p.PointedOfficialID,
		AreaID:                p.Location.AreaID.String(),
		ReporterID:            p.ReporterID,
		ValidCount:            p.ValidCount,
		EligibleForAssignment: p.Status.EligibleForAssignment(),
		// Routing is filled by ListQueue, not here — it is the assignment context's
		// own authority, not a fact about the problem.
	}
}

// assignmentOfficialAdapter bridges the assignment context's Officials port to
// the identity officials directory + the area tree (for monitor and fallback
// resolution).
type assignmentOfficialAdapter struct {
	officials identitydomain.OfficialRepository
	areas     areadomain.Repository
}

var _ assignmentapp.Officials = assignmentOfficialAdapter{}

func (a assignmentOfficialAdapter) Get(ctx context.Context, officialID string) (assignmentapp.OfficialView, error) {
	o, err := a.officials.GetByID(ctx, officialID)
	if errors.Is(err, identitydomain.ErrOfficialNotFound) {
		return assignmentapp.OfficialView{}, assignmentdomain.ErrOfficialNotFound
	}
	if err != nil {
		return assignmentapp.OfficialView{}, err
	}
	return assignmentapp.OfficialView{ID: o.ID, Tier: o.Tier, AreaID: o.AreaID.String(), Name: o.Name}, nil
}

// MonitorFor returns the official one area-level up the ladder: a union/ward
// official is monitored by the enclosing upazila's chairman; an upazila official
// by the seat's MP; the MP/minister has no monitor.
//
// The walk itself lives in identity, which owns the officials directory and so is
// the context entitled to answer — the same delegation adminUnion makes. B19 needs
// the ladder more than one rung deep (an unanswered case climbs to the tier above,
// then the one above that), and a monitor resolved two different ways is how the
// assignment's monitor and the escalation's observers would come to disagree about
// who supervises whom. This is that ladder asked for its first rung only.
func (a assignmentOfficialAdapter) MonitorFor(ctx context.Context, officialID string) (string, error) {
	if _, err := a.officials.GetByID(ctx, officialID); errors.Is(err, identitydomain.ErrOfficialNotFound) {
		return "", assignmentdomain.ErrOfficialNotFound
	} else if err != nil {
		return "", err
	}
	chain, err := identityapp.MonitorChain(ctx, a.officials, a.areas, officialID, 1)
	if err != nil || len(chain) == 0 {
		return "", err
	}
	return chain[0], nil
}

func (a assignmentOfficialAdapter) officialByTierArea(ctx context.Context, tier valueobject.Tier, areaID string) (string, error) {
	officials, err := a.officials.List(ctx)
	if err != nil {
		return "", err
	}
	for _, o := range officials {
		if o.Tier == tier && o.AreaID.String() == areaID {
			return o.ID, nil
		}
	}
	return "", nil
}

func (a assignmentOfficialAdapter) officialByTier(ctx context.Context, tier valueobject.Tier) (string, error) {
	officials, err := a.officials.List(ctx)
	if err != nil {
		return "", err
	}
	for _, o := range officials {
		if o.Tier == tier {
			return o.ID, nil
		}
	}
	return "", nil
}

// assignmentAdminAdapter bridges the assignment context's Admins port: it resolves
// which union an admin belongs to (via their linked official) and the eligible
// voter set for a scope.
type assignmentAdminAdapter struct {
	accounts  identitydomain.AccountRepository
	officials identitydomain.OfficialRepository
	areas     areadomain.Repository
}

var _ assignmentapp.Admins = assignmentAdminAdapter{}

func (a assignmentAdminAdapter) UnionOf(ctx context.Context, adminAccountID string) (string, error) {
	return adminUnion(ctx, a.accounts, a.officials, a.areas, adminAccountID)
}

// adminUnion resolves the union an admin belongs to, on behalf of the problem and
// assignment contexts' Admins ports.
//
// The walk itself lives in identity, which owns accounts and officials and so is
// the context entitled to answer "what union is this admin scoped to". This is a
// delegation rather than a copy: an admin's scope decides who may screen, assign,
// and verify, and two implementations that drifted apart would disagree about who
// has power over whom.
func adminUnion(
	ctx context.Context,
	accounts identitydomain.AccountRepository,
	officials identitydomain.OfficialRepository,
	areas areadomain.Repository,
	adminAccountID string,
) (string, error) {
	return identityapp.AdminUnion(ctx, accounts, officials, areas, adminAccountID)
}

// problemAdminAdapter bridges the problem context's Admins port (B9 screening) to
// the identity repositories. It answers the same question as its assignment
// counterpart — which union is this admin scoped to — because screening and
// assignment enforce the same union boundary.
type problemAdminAdapter struct {
	accounts  identitydomain.AccountRepository
	officials identitydomain.OfficialRepository
	areas     areadomain.Repository
}

var _ problemapp.Admins = problemAdminAdapter{}

func (a problemAdminAdapter) UnionOf(ctx context.Context, adminAccountID string) (string, error) {
	return adminUnion(ctx, a.accounts, a.officials, a.areas, adminAccountID)
}

// problemAssignmentAdapter bridges the problem context's Assignments port to the
// assignment repository, so the problem reads can publish the official the problem
// was actually handed to alongside the one the public pointed at — and, when those
// two differ, the reason the admin gave for the override.
type problemAssignmentAdapter struct {
	assignments assignmentdomain.Repository
}

var _ problemapp.Assignments = problemAssignmentAdapter{}

// For swallows ErrAssignmentNotFound into a zero view, mirroring how the evidence
// and progress reads treat a missing case: most problems have never been assigned,
// and "nobody yet" is the normal answer, not a failure.
func (a problemAssignmentAdapter) For(ctx context.Context, problemID string) (problemapp.AssignmentView, error) {
	asgn, err := a.assignments.GetAssignmentByProblem(ctx, problemID)
	if errors.Is(err, assignmentdomain.ErrAssignmentNotFound) {
		return problemapp.AssignmentView{}, nil
	}
	if err != nil {
		return problemapp.AssignmentView{}, err
	}
	return problemapp.AssignmentView{OfficialID: asgn.OfficialID, OverrideReason: asgn.OverrideReason}, nil
}

// ForMany is the batch form, one query for a whole page. Unassigned problems are
// absent from the map rather than present-and-zero, so the caller's lookup miss
// and "not assigned" are the same answer.
func (a problemAssignmentAdapter) ForMany(ctx context.Context, problemIDs []string) (map[string]problemapp.AssignmentView, error) {
	asgns, err := a.assignments.AssignmentsByProblems(ctx, problemIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]problemapp.AssignmentView, len(asgns))
	for problemID, asgn := range asgns {
		out[problemID] = problemapp.AssignmentView{OfficialID: asgn.OfficialID, OverrideReason: asgn.OverrideReason}
	}
	return out, nil
}

// EligibleAdvisers returns the admin account IDs entitled to advise on an
// above-union report in the given scope: every admin in the seat, or the admins
// whose union sits in the problem's upazila.
//
// It was EligibleVoters until B20. The set it computes is unchanged — who has
// standing to speak on a problem did not change when the decision moved to the
// super admin, only what their input does (A.3.8).
func (a assignmentAdminAdapter) EligibleAdvisers(ctx context.Context, scope assignmentdomain.AdviceScope, problemAreaID string) ([]string, error) {
	admins, err := a.accounts.ListByRole(ctx, identitydomain.RoleAdmin)
	if err != nil {
		return nil, err
	}
	if scope == assignmentdomain.ScopeSeat {
		ids := make([]string, 0, len(admins))
		for i := range admins {
			ids = append(ids, admins[i].ID)
		}
		return ids, nil
	}

	problemUpazila, err := a.upazilaOfArea(ctx, problemAreaID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for i := range admins {
		adminUpazila, err := a.upazilaOfAdmin(ctx, admins[i])
		if err != nil {
			return nil, err
		}
		if adminUpazila != "" && adminUpazila == problemUpazila {
			ids = append(ids, admins[i].ID)
		}
	}
	return ids, nil
}

func (a assignmentAdminAdapter) upazilaOfAdmin(ctx context.Context, acc identitydomain.Account) (string, error) {
	if acc.OfficialID == "" {
		return "", nil
	}
	o, err := a.officials.GetByID(ctx, acc.OfficialID)
	if err != nil {
		return "", nil
	}
	return a.upazilaOfArea(ctx, o.AreaID.String())
}

func (a assignmentAdminAdapter) upazilaOfArea(ctx context.Context, areaID string) (string, error) {
	area, err := a.areas.GetByID(ctx, valueobject.AreaID(areaID))
	if err != nil {
		return "", nil
	}
	return upazilaOf(ctx, a.areas, area)
}

// unionOfArea returns the union an area belongs to.
func unionOfArea(area *areadomain.Area) string {
	return areadomain.UnionOfArea(area)
}

// --- resolution context adapters (B5) ---

// resolutionAssignmentAdapter bridges the resolution context's Assignments port
// to the assignment repository (to materialize cases from assignments).
type resolutionAssignmentAdapter struct {
	assignments assignmentdomain.Repository
}

var _ resolutionapp.Assignments = resolutionAssignmentAdapter{}

func (a resolutionAssignmentAdapter) Get(ctx context.Context, problemID string) (resolutionapp.AssignmentView, error) {
	asgn, err := a.assignments.GetAssignmentByProblem(ctx, problemID)
	if errors.Is(err, assignmentdomain.ErrAssignmentNotFound) {
		return resolutionapp.AssignmentView{}, resolutiondomain.ErrAssignmentMissing
	}
	if err != nil {
		return resolutionapp.AssignmentView{}, err
	}
	return toAssignmentView(asgn), nil
}

func (a resolutionAssignmentAdapter) ListByOfficial(ctx context.Context, officialID string) ([]resolutionapp.AssignmentView, error) {
	asgns, err := a.assignments.ListByOfficial(ctx, officialID)
	if err != nil {
		return nil, err
	}
	out := make([]resolutionapp.AssignmentView, 0, len(asgns))
	for i := range asgns {
		out = append(out, toAssignmentView(&asgns[i]))
	}
	return out, nil
}

func toAssignmentView(a *assignmentdomain.Assignment) resolutionapp.AssignmentView {
	return resolutionapp.AssignmentView{
		ProblemID:         a.ProblemID,
		OfficialID:        a.OfficialID,
		MonitorOfficialID: a.MonitorOfficialID,
		Deadline:          a.Deadline,
	}
}

// resolutionProblemAdapter bridges the resolution context's Problems port to the
// problem repository: the problem's area (for residency) and status mirroring.
type resolutionProblemAdapter struct {
	problems problemdomain.Repository
}

var _ resolutionapp.Problems = resolutionProblemAdapter{}

func (a resolutionProblemAdapter) AreaID(ctx context.Context, problemID string) (string, error) {
	p, err := a.problems.GetByID(ctx, problemID)
	if errors.Is(err, problemdomain.ErrProblemNotFound) {
		return "", resolutiondomain.ErrProblemNotFound
	}
	if err != nil {
		return "", err
	}
	return p.Location.AreaID.String(), nil
}

func (a resolutionProblemAdapter) ReporterID(ctx context.Context, problemID string) (string, error) {
	p, err := a.problems.GetByID(ctx, problemID)
	if errors.Is(err, problemdomain.ErrProblemNotFound) {
		return "", resolutiondomain.ErrProblemNotFound
	}
	if err != nil {
		return "", err
	}
	return p.ReporterID, nil
}

func (a resolutionProblemAdapter) SetStatus(ctx context.Context, problemID, status string) error {
	return a.problems.SetStatus(ctx, problemID, problemdomain.Status(status))
}

// TitlesByIDs names the problems on a monitor's observation list (B19), reusing
// the same batch query the public activity feed does — publicly visible problems
// only, filtered in the SQL.
func (a resolutionProblemAdapter) TitlesByIDs(ctx context.Context, problemIDs []string) (map[string]string, error) {
	return a.problems.TitlesByIDs(ctx, problemIDs)
}

// resolutionOfficialAdapter bridges the resolution context's Officials and Ladder
// ports to the identity directory and the area tree.
//
// The ladder walk is delegated to identity rather than restated here, for the
// reason assignmentOfficialAdapter.MonitorFor gives: the assignment's monitor and
// the escalation's observers must never be able to disagree about who supervises
// whom.
type resolutionOfficialAdapter struct {
	officials identitydomain.OfficialRepository
	areas     areadomain.Repository
}

var (
	_ resolutionapp.Officials = resolutionOfficialAdapter{}
	_ resolutionapp.Ladder    = resolutionOfficialAdapter{}
)

func (a resolutionOfficialAdapter) NamesByIDs(ctx context.Context, officialIDs []string) (map[string]string, error) {
	return a.officials.NamesByIDs(ctx, officialIDs)
}

func (a resolutionOfficialAdapter) Chain(ctx context.Context, officialID string, maxRungs int) ([]string, error) {
	return identityapp.MonitorChain(ctx, a.officials, a.areas, officialID, maxRungs)
}

// resolutionSuggestionAdapter bridges the resolution context's Suggestions port
// to the suggestion repository and its ranking service, so a plan can snapshot
// the suggestion it answers (B10).
//
// It delegates "top" to suggestiondomain.Service.Top rather than deciding it
// here. That service is the single authority for ranking (B3), and a second
// implementation — in SQL or in this adapter — could disagree with it, so the
// official's public record would name a different top suggestion than the problem
// page the community actually voted on.
type resolutionSuggestionAdapter struct {
	suggestions suggestiondomain.Repository
	svc         *suggestiondomain.Service
}

var _ resolutionapp.Suggestions = resolutionSuggestionAdapter{}

func (a resolutionSuggestionAdapter) Top(ctx context.Context, problemID string) (resolutionapp.TopSuggestion, error) {
	all, err := a.suggestions.ListByProblem(ctx, problemID)
	if err != nil {
		return resolutionapp.TopSuggestion{}, err
	}
	top, ok := a.svc.Top(all)
	if !ok {
		// No suggestion, or none with any upvotes: there is nothing to answer, and
		// that is a normal state rather than an error.
		return resolutionapp.TopSuggestion{}, nil
	}
	return resolutionapp.TopSuggestion{ID: top.ID, Text: top.Text, Found: true}, nil
}

// resolutionIdentityAdapter bridges the resolution context's Identity port to the
// identity accounts (for obstacle-judgment residency).
type resolutionIdentityAdapter struct {
	accounts identitydomain.AccountRepository
}

var _ resolutionapp.Identity = resolutionIdentityAdapter{}

func (a resolutionIdentityAdapter) Voter(ctx context.Context, accountID string) (resolutionapp.Voter, error) {
	acc, err := a.accounts.GetByID(ctx, accountID)
	if err != nil {
		return resolutionapp.Voter{}, err
	}
	return resolutionapp.Voter{
		IsResident: acc.Role == identitydomain.RoleResident,
		Verified:   acc.Verified,
		UnionID:    acc.UnionID.String(),
	}, nil
}

// auditActorAdapter resolves the audit log's actor ids to display names for the
// public activity feed.
//
// It reads BOTH tables because that one column holds two id spaces: an admin's
// action is recorded against their ACCOUNT id, an official's against their
// DIRECTORY OFFICE id. Accounts are asked first and offices fill the gaps, so a
// hypothetical id present in both resolves to the account — the actor of an
// account-authored action.
//
// This is why the audit context declares a port instead of importing identity:
// `shared/audit` sits under every context and must not reach sideways into one
// (A.4.1).
type auditActorAdapter struct {
	accounts  identitydomain.AccountRepository
	officials identitydomain.OfficialRepository
}

var _ auditapp.Actors = auditActorAdapter{}

func (a auditActorAdapter) NamesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	names, err := a.accounts.NamesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	missing := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := names[id]; !ok {
			missing = append(missing, id)
		}
	}
	// Every actor was an account: no second query.
	if len(missing) == 0 {
		return names, nil
	}
	offices, err := a.officials.NamesByIDs(ctx, missing)
	if err != nil {
		// The accounts half already resolved; returning it beats failing the whole
		// page over the offices half (the handler degrades either way).
		return names, nil
	}
	for id, name := range offices {
		names[id] = name
	}
	return names, nil
}

// auditProblemAdapter resolves problem ids to titles for the public activity
// feed. The repository filters to publicly visible problems, so a report still
// awaiting screening can never surface a title here.
type auditProblemAdapter struct {
	problems problemdomain.Repository
}

var _ auditapp.Problems = auditProblemAdapter{}

func (a auditProblemAdapter) TitlesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	return a.problems.TitlesByIDs(ctx, ids)
}

// upazilaOf resolves the upazila enclosing an area (ward → union → upazila, union
// → upazila, upazila → itself, seat → "").
func upazilaOf(ctx context.Context, areas areadomain.Repository, area *areadomain.Area) (string, error) {
	switch area.Level {
	case areadomain.LevelUpazila:
		return area.ID.String(), nil
	case areadomain.LevelUnion:
		return area.ParentID.String(), nil
	case areadomain.LevelWard:
		union, err := areas.GetByID(ctx, area.ParentID)
		if err != nil {
			return "", nil
		}
		return union.ParentID.String(), nil
	default: // seat
		return "", nil
	}
}
