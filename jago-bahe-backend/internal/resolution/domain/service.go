package domain

import "sort"

// Service holds the pure rules that span a case's obstacle judgment: ranking the
// community's unblocking plans. Ranking is backend-owned (as with suggestions);
// the http/UI layers only display the order. The advisory obstacle vote is a
// tally, never a status change — that authority stays out of this service.
type Service struct{}

// NewService constructs the resolution domain service.
func NewService() *Service { return &Service{} }

// RankUnblocking orders unblocking plans by upvotes (highest first, ties broken
// by oldest first for stability) — the same rule the suggestion context uses,
// re-pointed at a blocker.
func (Service) RankUnblocking(plans []UnblockingPlan) []UnblockingPlan {
	ranked := make([]UnblockingPlan, len(plans))
	copy(ranked, plans)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].UpvoteCount != ranked[j].UpvoteCount {
			return ranked[i].UpvoteCount > ranked[j].UpvoteCount
		}
		return ranked[i].CreatedAt.Before(ranked[j].CreatedAt)
	})
	return ranked
}
