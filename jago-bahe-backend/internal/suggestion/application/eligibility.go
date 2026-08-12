package application

import (
	"context"

	areadomain "jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/internal/suggestion/domain"
)

// ensureAreaResident enforces the B3 guardrail shared by propose and upvote: only
// a verified resident of the problem's area (its union) may act. It returns the
// resolved area so callers need not fetch it twice.
func ensureAreaResident(ctx context.Context, id Identity, areas areadomain.Repository, problems Problems, problemID, actorID string) error {
	areaID, err := problems.AreaID(ctx, problemID)
	if err != nil {
		return err // domain.ErrProblemNotFound
	}
	voter, err := id.Voter(ctx, actorID)
	if err != nil {
		return err
	}
	if !voter.IsResident || !voter.Verified {
		return domain.ErrNotVerified
	}
	union, err := unionOf(ctx, areas, areaID.String())
	if err != nil {
		return err
	}
	if voter.UnionID == "" || voter.UnionID != union {
		return domain.ErrNotAreaResident
	}
	return nil
}

// unionOf resolves the union an area belongs to, mapping an unknown area onto
// this context's sentinel.
func unionOf(ctx context.Context, areas areadomain.Repository, areaID string) (string, error) {
	union, err := areadomain.ResolveUnion(ctx, areas, areaID)
	if err != nil {
		return "", domain.ErrAreaNotFound
	}
	return union, nil
}
