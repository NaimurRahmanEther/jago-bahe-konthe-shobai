package domain

import "context"

// Query is the read-model port: the scorecard context depends only on this
// interface, with a read-optimized Postgres implementation in infrastructure.
// Being a read model, it may draw facts from whatever tables it needs (cases,
// blockers, officials, problems) — it owns no aggregate.
type Query interface {
	// OfficialExists reports whether the official id is in the directory.
	OfficialExists(ctx context.Context, officialID string) (bool, error)
	// CaseFactsByOfficial returns the fairness facts for one official's cases.
	CaseFactsByOfficial(ctx context.Context, officialID string) ([]CaseFact, error)
	// SeatCaseFacts returns the fairness facts for every case in the seat.
	SeatCaseFacts(ctx context.Context) ([]CaseFact, error)
	// SeatProblemCount returns the number of publicly visible problems in the seat.
	// It must never count problems awaiting screening: that number is exactly what
	// the screening gate withholds.
	SeatProblemCount(ctx context.Context) (int, error)
	// RecordByOfficial returns the official's public case record, newest first,
	// restricted to publicly visible problems.
	RecordByOfficial(ctx context.Context, officialID string) ([]CaseRecord, error)
}
