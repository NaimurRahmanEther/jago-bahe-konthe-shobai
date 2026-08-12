// Package domain holds the scorecard read model and the fairness rule that turns
// a set of per-case facts into public accountability numbers. The rule — not the
// SQL and never the advisory public vote — is the authority here, so it lives in
// pure, table-driven-testable Go: silence and missed deadlines count against an
// official; a case blocked on a higher authority (a blocker adjudicated *real*)
// does not (Concept §9).
package domain

import "math"

// Case status values the fairness rule keys on. Kept as local literals rather
// than importing the resolution context, so the read model couples to no other
// domain package (they are stable contract strings).
const (
	statusResolved = "Resolved"
	statusBlocked  = "Blocked"
	statusDisputed = "Disputed"
)

// Adjudication is the verdict on a case's active blocker, projected for the
// fairness rule. Only Confirmed (the named authority judged the obstacle real)
// protects the official; the advisory public vote is never consulted here.
type Adjudication string

const (
	AdjNone      Adjudication = ""          // no active (unresolved) blocker
	AdjPending   Adjudication = "pending"   // declared, not yet adjudicated
	AdjConfirmed Adjudication = "confirmed" // judged real → responsibility moved up
	AdjDenied    Adjudication = "denied"    // judged an excuse → bounced back
)

// CaseFact is the minimal per-case projection the fairness rule needs: the
// current status, the adjudication of the active blocker (if any), and the
// first-response latency (acknowledgedAt − createdAt, in days) when acknowledged.
type CaseFact struct {
	Status        string
	ActiveBlocker Adjudication
	Acknowledged  bool
	ResponseDays  float64
}

// OfficialStats is the public per-official scorecard (mirrors the frontend
// OfficialStats read model). Blocked is the *fair* bucket — cases waiting on a
// higher authority, never counted as this official's failure.
type OfficialStats struct {
	OfficialID      string
	Resolved        int
	Pending         int
	Blocked         int
	AvgResponseDays float64
}

// SeatOverview rolls the same fairness buckets up across the whole seat. Problems
// is the seat-wide reported-problem count, set by the use case.
type SeatOverview struct {
	Problems        int
	Cases           int
	Resolved        int
	Pending         int
	Blocked         int
	AvgResponseDays float64
}

// classify buckets one case under the fairness rule.
//   - Resolved                          → resolved (credit)
//   - Blocked *and* adjudicated confirmed → blocked (fair — waiting on a higher
//     authority, not a failure)
//   - Disputed                          → neither (not this official's case)
//   - everything else, including a blocker still pending or judged an excuse
//     (the case has bounced back to open work), silence, and missed deadlines
//     → pending (counts against)
//
// It returns which counter to bump: 0 resolved, 1 blocked, 2 pending, -1 skip.
func classify(f CaseFact) int {
	switch {
	case f.Status == statusResolved:
		return 0
	case f.Status == statusBlocked && f.ActiveBlocker == AdjConfirmed:
		return 1
	case f.Status == statusDisputed:
		return -1
	default:
		return 2
	}
}

// aggregate applies the fairness rule to a set of case facts and averages the
// first-response latency over the cases that were acknowledged (0 when none).
func aggregate(facts []CaseFact) (resolved, pending, blocked int, avgResponseDays float64) {
	var sum float64
	var n int
	for _, f := range facts {
		switch classify(f) {
		case 0:
			resolved++
		case 1:
			blocked++
		case 2:
			pending++
		}
		if f.Acknowledged {
			sum += f.ResponseDays
			n++
		}
	}
	if n > 0 {
		avgResponseDays = roundDays(sum / float64(n))
	}
	return resolved, pending, blocked, avgResponseDays
}

// AggregateOfficial produces one official's public scorecard from their cases.
func AggregateOfficial(officialID string, facts []CaseFact) OfficialStats {
	resolved, pending, blocked, avg := aggregate(facts)
	return OfficialStats{
		OfficialID:      officialID,
		Resolved:        resolved,
		Pending:         pending,
		Blocked:         blocked,
		AvgResponseDays: avg,
	}
}

// AggregateSeat rolls every case in the seat up under the same fairness rule.
// Problems is filled in by the use case from the seat-wide problem count.
func AggregateSeat(facts []CaseFact) SeatOverview {
	resolved, pending, blocked, avg := aggregate(facts)
	return SeatOverview{
		Cases:           len(facts),
		Resolved:        resolved,
		Pending:         pending,
		Blocked:         blocked,
		AvgResponseDays: avg,
	}
}

// roundDays rounds to one decimal place, matching the frontend's presentation.
func roundDays(d float64) float64 {
	return math.Round(d*10) / 10
}
