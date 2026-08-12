package domain_test

import (
	"testing"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
	scorecarddomain "jago-bahe-backend/internal/scorecard/domain"
)

func blockedCase(status domain.Status, adj domain.Adjudication, resolved bool) *domain.Case {
	b := domain.Obstacle{ID: "blk-1", Adjudication: adj}
	if resolved {
		t := time.Now().UTC()
		b.ResolvedAt = &t
	}
	return &domain.Case{ID: "case-1", Status: status, Obstacles: []domain.Obstacle{b}}
}

// TestBlockedOnHigherAuthority pins the fairness nuance a public view owes an
// official: only an obstacle the named authority judged *real* protects them.
func TestBlockedOnHigherAuthority(t *testing.T) {
	tests := []struct {
		name   string
		status domain.Status
		adj    domain.Adjudication
		want   bool
	}{
		{"blocked and confirmed real is protected", domain.StatusBlocked, domain.AdjudicationConfirmed, true},
		// A blocker nobody has ruled on yet earns no protection — otherwise
		// declaring one would be enough to stop the clock on your own say-so.
		{"blocked but not yet adjudicated is not protected", domain.StatusBlocked, domain.AdjudicationPending, false},
		{"blocked and judged an excuse is not protected", domain.StatusBlocked, domain.AdjudicationDenied, false},
		{"in progress is not protected", domain.StatusInProgress, domain.AdjudicationConfirmed, false},
		{"resolved is not protected", domain.StatusResolved, domain.AdjudicationConfirmed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := blockedCase(tt.status, tt.adj, false).BlockedOnHigherAuthority(); got != tt.want {
				t.Errorf("BlockedOnHigherAuthority() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNoActiveObstacleIsNotProtected covers the two ways a case can be Blocked
// with nothing active: no blocker at all, or every blocker already resolved.
func TestNoActiveObstacleIsNotProtected(t *testing.T) {
	bare := &domain.Case{ID: "case-1", Status: domain.StatusBlocked}
	if bare.BlockedOnHigherAuthority() {
		t.Error("a case with no blocker must not read as blocked on a higher authority")
	}
	if blockedCase(domain.StatusBlocked, domain.AdjudicationConfirmed, true).BlockedOnHigherAuthority() {
		t.Error("a resolved blocker is not active and must not protect the official")
	}
}

// TestFairnessRuleAgreesWithScorecard is the guard against the one real risk of
// stating this rule in two places: that the two drift and the platform starts
// contradicting itself — a problem page saying an official is fairly blocked
// while their scorecard counts the same case against them.
//
// resolution owns Status and Adjudication; the scorecard read model deliberately
// restates them as local literals so it couples to no other domain package (see
// scorecard/domain/fairness.go). Neither should import the other, so instead of a
// shared function this test asserts they return the same answer for every
// combination. It is a test-only dependency and pins behaviour, not identity.
func TestFairnessRuleAgreesWithScorecard(t *testing.T) {
	statuses := []domain.Status{
		domain.StatusAssigned, domain.StatusAcknowledged, domain.StatusPlanned,
		domain.StatusInProgress, domain.StatusBlocked, domain.StatusDone,
		domain.StatusResolved, domain.StatusReopened, domain.StatusDisputed,
	}
	adjudications := []domain.Adjudication{
		domain.AdjudicationPending, domain.AdjudicationConfirmed, domain.AdjudicationDenied,
	}

	for _, status := range statuses {
		for _, adj := range adjudications {
			mine := blockedCase(status, adj, false).BlockedOnHigherAuthority()
			theirs := scorecarddomain.FairnessFlag(string(status), scorecarddomain.Adjudication(adj))
			if mine != theirs {
				t.Errorf("status=%s adj=%s: resolution says %v, scorecard says %v — the two fairness rules have drifted",
					status, adj, mine, theirs)
			}
		}
	}
}
