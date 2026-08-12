package application

import (
	"context"
	"errors"

	"jago-bahe-backend/internal/resolution/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/pkg/idgen"
)

// ensureCase returns the case for a problem, materializing it from the problem's
// assignment (status Assigned) the first time an official interacts with it. One
// case per problem (repository enforces the uniqueness).
func ensureCase(ctx context.Context, repo domain.Repository, assignments Assignments, problemID string) (*domain.Case, error) {
	c, err := repo.GetByProblem(ctx, problemID)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, domain.ErrCaseNotFound) {
		return nil, err
	}
	a, err := assignments.Get(ctx, problemID)
	if err != nil {
		return nil, err // domain.ErrAssignmentMissing
	}
	c = domain.NewCase(idgen.New("case"), problemID, a.OfficialID, a.MonitorOfficialID, a.Deadline)
	if err := repo.Save(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// loadOwnedCase loads a case by id and verifies the caller owns it (an official
// may only act on cases assigned to them — the ownership guard).
func loadOwnedCase(ctx context.Context, repo domain.Repository, caseID, officialID string) (*domain.Case, error) {
	c, err := repo.GetByID(ctx, caseID)
	if err != nil {
		return nil, err // domain.ErrCaseNotFound
	}
	if c.OfficialID != officialID {
		return nil, domain.ErrNotCaseOwner
	}
	return c, nil
}

// ensureAreaResident enforces the obstacle-judgment guardrail (shared by vote,
// propose, and upvote): only a verified resident of the problem's area (its
// union) may act — mirroring the suggestion context's rule.
func ensureAreaResident(ctx context.Context, identity Identity, areas areadomain.Repository, problems Problems, problemID, actorID string) error {
	areaID, err := problems.AreaID(ctx, problemID)
	if err != nil {
		return err // domain.ErrProblemNotFound
	}
	voter, err := identity.Voter(ctx, actorID)
	if err != nil {
		return err
	}
	if !voter.IsResident || !voter.Verified {
		return domain.ErrNotVerified
	}
	union, err := resolveUnion(ctx, areas, areaID)
	if err != nil {
		return err
	}
	if voter.UnionID == "" || voter.UnionID != union {
		return domain.ErrNotAreaResident
	}
	return nil
}

// notificationTitle resolves ONE problem title for a notification, reusing the
// existing batch method rather than adding a port method for a single id.
//
// It degrades to "" on any failure, which is correct rather than lax: a
// notification with no title is one the UI renders as "শিরোনামহীন প্রতিবেদন", while
// a failure that propagated would abort a lifecycle action because a cosmetic
// lookup had a bad day (A.3.4 rule 3).
//
// TitlesByIDs resolves PUBLICLY VISIBLE problems only. Every status that can carry
// a case is public, so this always resolves in practice — and that it would blank
// otherwise is a free safety net under the snapshotting rule (A.3.9 constraint 3).
func notificationTitle(ctx context.Context, problems Problems, problemID string) string {
	titles, err := problems.TitlesByIDs(ctx, []string{problemID})
	if err != nil {
		return ""
	}
	return titles[problemID]
}

// resolveUnion returns the union an area belongs to, mapping an unknown area onto
// this context's sentinel.
func resolveUnion(ctx context.Context, areas areadomain.Repository, areaID string) (string, error) {
	union, err := areadomain.ResolveUnion(ctx, areas, areaID)
	if err != nil {
		return "", domain.ErrAreaNotFound
	}
	return union, nil
}
