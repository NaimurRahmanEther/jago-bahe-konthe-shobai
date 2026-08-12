package application

import (
	"context"

	"jago-bahe-backend/internal/suggestion/domain"
)

// ListSuggestions returns a problem's suggestions ranked by upvotes with the top
// suggestion(s) marked — ranking is backend-owned (the B3 guardrail).
type ListSuggestions struct {
	suggestions domain.Repository
	svc         *domain.Service
}

// NewListSuggestions wires the use case.
func NewListSuggestions(s domain.Repository, svc *domain.Service) *ListSuggestions {
	return &ListSuggestions{suggestions: s, svc: svc}
}

// Execute lists and ranks the problem's suggestions.
func (uc *ListSuggestions) Execute(ctx context.Context, problemID string) ([]domain.Suggestion, error) {
	sugs, err := uc.suggestions.ListByProblem(ctx, problemID)
	if err != nil {
		return nil, err
	}
	return uc.svc.Rank(sugs), nil
}
