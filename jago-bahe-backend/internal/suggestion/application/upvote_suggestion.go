package application

import (
	"context"

	areadomain "jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/internal/suggestion/domain"
	"jago-bahe-backend/pkg/idgen"
)

// UpvoteSuggestion toggles a resident's upvote on a suggestion: adding it if
// absent, withdrawing it if present. Only a verified resident of the problem's
// area may upvote (the B3 guardrail). It returns the suggestion with a fresh
// tally and its ranking (isTop) recomputed. No audit entry (upvotes are
// high-volume and reversible).
type UpvoteSuggestion struct {
	suggestions domain.Repository
	areas       areadomain.Repository
	problems    Problems
	identity    Identity
	svc         *domain.Service
}

// NewUpvoteSuggestion wires the use case.
func NewUpvoteSuggestion(s domain.Repository, a areadomain.Repository, pr Problems, id Identity, svc *domain.Service) *UpvoteSuggestion {
	return &UpvoteSuggestion{suggestions: s, areas: a, problems: pr, identity: id, svc: svc}
}

// Execute toggles the upvote and returns the target suggestion with its ranking
// recomputed across the problem (so isTop reflects the new tally), plus whether
// the caller now holds an upvote on it.
//
// That second value exists because this is a TOGGLE: the same request both casts
// and withdraws, so a response carrying only the new count leaves the caller to
// infer which happened. The UI renders a button whose pressed state depends on it,
// and inferring it from a count that other people are also changing is exactly the
// kind of guess that goes wrong under concurrency.
func (uc *UpvoteSuggestion) Execute(ctx context.Context, suggestionID, voterID string) (*domain.Suggestion, bool, error) {
	s, err := uc.suggestions.GetByID(ctx, suggestionID)
	if err != nil {
		return nil, false, err // domain.ErrSuggestionNotFound
	}
	if err := ensureAreaResident(ctx, uc.identity, uc.areas, uc.problems, s.ProblemID, voterID); err != nil {
		return nil, false, err
	}

	has, err := uc.suggestions.HasUpvoted(ctx, suggestionID, voterID)
	if err != nil {
		return nil, false, err
	}
	if has {
		if err := uc.suggestions.RemoveUpvote(ctx, suggestionID, voterID); err != nil {
			return nil, false, err
		}
	} else {
		vote := domain.SuggestionVote{ID: idgen.New("supv"), SuggestionID: suggestionID, VoterID: voterID}
		if err := uc.suggestions.AddUpvote(ctx, vote); err != nil {
			return nil, false, err
		}
	}
	upvoted := !has

	// Reload and rank the whole problem so the returned suggestion carries a
	// fresh count and a correct isTop (ranking is backend-owned).
	sugs, err := uc.suggestions.ListByProblem(ctx, s.ProblemID)
	if err != nil {
		return nil, false, err
	}
	for _, ranked := range uc.svc.Rank(sugs) {
		if ranked.ID == suggestionID {
			return &ranked, upvoted, nil
		}
	}
	return nil, false, domain.ErrSuggestionNotFound
}
