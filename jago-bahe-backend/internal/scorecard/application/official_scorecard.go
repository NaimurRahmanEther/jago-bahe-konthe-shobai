// Package application holds the scorecard read use cases: they fetch per-case
// facts through the Query port and let the domain fairness rule turn them into
// public numbers. No writes, no audit — the scorecard only reads the trail the
// other contexts already wrote.
package application

import (
	"context"

	"jago-bahe-backend/internal/scorecard/domain"
)

// OfficialScorecard produces one official's public accountability numbers.
type OfficialScorecard struct {
	query domain.Query
}

// NewOfficialScorecard wires the use case.
func NewOfficialScorecard(q domain.Query) *OfficialScorecard {
	return &OfficialScorecard{query: q}
}

// Execute returns the official's scorecard, or ErrOfficialNotFound if the id is
// not in the directory.
func (uc *OfficialScorecard) Execute(ctx context.Context, officialID string) (domain.OfficialStats, error) {
	exists, err := uc.query.OfficialExists(ctx, officialID)
	if err != nil {
		return domain.OfficialStats{}, err
	}
	if !exists {
		return domain.OfficialStats{}, domain.ErrOfficialNotFound
	}
	facts, err := uc.query.CaseFactsByOfficial(ctx, officialID)
	if err != nil {
		return domain.OfficialStats{}, err
	}
	return domain.AggregateOfficial(officialID, facts), nil
}
