package application

import (
	"context"

	"jago-bahe-backend/internal/scorecard/domain"
)

// OfficialRecord returns an official's public record: what they were assigned,
// what they published about it, and when.
//
// It exists because the scorecard alone is a thin form of accountability — four
// numbers say an official is slow without ever showing what they said. The
// suggestionResponse, where an official answers the community's most-supported
// idea, is required at submission and was until now unreachable from their own
// page. This makes their words attributable to them.
//
// It reports; it does not judge. There is no ranking, no grading, and no
// comparison against other officials here, and none should be added (Concept §10:
// stay strictly factual and neutral — report what happened and the dates).
type OfficialRecord struct {
	query domain.Query
}

// NewOfficialRecord wires the use case.
func NewOfficialRecord(q domain.Query) *OfficialRecord {
	return &OfficialRecord{query: q}
}

// Execute returns the record, or ErrOfficialNotFound for an unknown official.
func (uc *OfficialRecord) Execute(ctx context.Context, officialID string) (domain.OfficialRecord, error) {
	exists, err := uc.query.OfficialExists(ctx, officialID)
	if err != nil {
		return domain.OfficialRecord{}, err
	}
	if !exists {
		return domain.OfficialRecord{}, domain.ErrOfficialNotFound
	}

	cases, err := uc.query.RecordByOfficial(ctx, officialID)
	if err != nil {
		return domain.OfficialRecord{}, err
	}
	return domain.BuildRecord(officialID, cases), nil
}
