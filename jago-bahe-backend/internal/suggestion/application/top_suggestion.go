package application

import (
	"context"

	"jago-bahe-backend/internal/suggestion/domain"
)

// TopSuggestion returns the single highest-upvoted suggestion for a problem. It
// has no HTTP route in B3; it is the authority B5's resolution plan must answer
// ("the plan must respond to the community's top suggestion").
type TopSuggestion struct {
	suggestions domain.Repository
	svc         *domain.Service
}

// NewTopSuggestion wires the use case.
func NewTopSuggestion(s domain.Repository, svc *domain.Service) *TopSuggestion {
	return &TopSuggestion{suggestions: s, svc: svc}
}

// Execute returns the top suggestion and whether one exists (false when the
// problem has no upvoted suggestion).
func (uc *TopSuggestion) Execute(ctx context.Context, problemID string) (*domain.Suggestion, bool, error) {
	sugs, err := uc.suggestions.ListByProblem(ctx, problemID)
	if err != nil {
		return nil, false, err
	}
	top, ok := uc.svc.Top(sugs)
	if !ok {
		return nil, false, nil
	}
	return &top, true, nil
}
