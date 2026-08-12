package domain

import "sort"

// Service holds the ranking rule that spans a problem's suggestions. It is the
// authority for ordering and "top" selection (the B3 guardrail: ranking is
// backend-owned); the http/UI layers only display the outcome.
type Service struct{}

// NewService constructs the suggestion domain service.
func NewService() *Service { return &Service{} }

// Rank returns the suggestions ordered by upvotes (highest first, ties broken by
// oldest first for stability) and marks every suggestion holding the maximum
// upvote count as IsTop. A suggestion with zero upvotes is never top, so a
// problem with no upvotes has no top suggestion at all.
func (Service) Rank(suggestions []Suggestion) []Suggestion {
	ranked := make([]Suggestion, len(suggestions))
	copy(ranked, suggestions)

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].UpvoteCount != ranked[j].UpvoteCount {
			return ranked[i].UpvoteCount > ranked[j].UpvoteCount
		}
		return ranked[i].CreatedAt.Before(ranked[j].CreatedAt)
	})

	max := 0
	for _, s := range ranked {
		if s.UpvoteCount > max {
			max = s.UpvoteCount
		}
	}
	for i := range ranked {
		ranked[i].IsTop = max > 0 && ranked[i].UpvoteCount == max
	}
	return ranked
}

// Top returns the single highest-upvoted suggestion for a problem and whether
// one exists. Ties resolve to the oldest (via Rank's stable order); a set with no
// upvotes yields ok=false. B5's plan must answer this suggestion.
func (s Service) Top(suggestions []Suggestion) (Suggestion, bool) {
	ranked := s.Rank(suggestions)
	if len(ranked) == 0 || !ranked[0].IsTop {
		return Suggestion{}, false
	}
	return ranked[0], true
}
