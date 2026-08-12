package application

import (
	"context"

	"jago-bahe-backend/internal/scorecard/domain"
)

// SeatOverview rolls every case in the seat up under the fairness rule and adds
// the seat-wide problem count.
//
// Built routeless in B7; B10 added GET /api/seat/overview to the contract and
// wired it. The problem count it carries is publicly visible problems only —
// counting reports still awaiting screening would publish, to anyone, the very
// number the screening gate withholds.
type SeatOverview struct {
	query domain.Query
}

// NewSeatOverview wires the use case.
func NewSeatOverview(q domain.Query) *SeatOverview {
	return &SeatOverview{query: q}
}

// Execute returns the seat overview.
func (uc *SeatOverview) Execute(ctx context.Context) (domain.SeatOverview, error) {
	facts, err := uc.query.SeatCaseFacts(ctx)
	if err != nil {
		return domain.SeatOverview{}, err
	}
	problems, err := uc.query.SeatProblemCount(ctx)
	if err != nil {
		return domain.SeatOverview{}, err
	}
	overview := domain.AggregateSeat(facts)
	overview.Problems = problems
	return overview, nil
}
