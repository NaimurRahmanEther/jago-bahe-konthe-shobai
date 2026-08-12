package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// ObstacleCategory names the kind of obstacle; the category decides which higher
// authority the blocker is forwarded to (Concept §8). Backend-owned routing.
type ObstacleCategory string

const (
	CategoryBudget          ObstacleCategory = "budget"
	CategoryLegalAuthority  ObstacleCategory = "legal_authority"
	CategoryHigherTier      ObstacleCategory = "higher_tier"
	CategoryLandDispute     ObstacleCategory = "land_dispute"
	CategoryInterDepartment ObstacleCategory = "inter_department"
	CategoryTechnical       ObstacleCategory = "technical"
)

// Valid reports whether c is a known category.
func (c ObstacleCategory) Valid() bool {
	switch c {
	case CategoryBudget, CategoryLegalAuthority, CategoryHigherTier,
		CategoryLandDispute, CategoryInterDepartment, CategoryTechnical:
		return true
	}
	return false
}

// Adjudication is the named authority's (or moderator's) verdict on a blocker —
// the decision that sets the scorecard consequence (NOT the advisory public
// vote). Confirmed → responsibility sits with the authority, the official is
// protected; Denied → it bounces back, the clock resumes, and it may count
// against the official (Concept §8).
type Adjudication string

const (
	AdjudicationPending   Adjudication = "pending"
	AdjudicationConfirmed Adjudication = "confirmed"
	AdjudicationDenied    Adjudication = "denied"
)

// Valid reports whether a is a known verdict.
func (a Adjudication) Valid() bool {
	return a == AdjudicationPending || a == AdjudicationConfirmed || a == AdjudicationDenied
}

// Obstacle is a typed obstacle on a case: it keeps the case open, notifies the
// named higher authority, and is judged in public (an advisory "is it real?" vote
// and community unblocking plans). The advisory tallies are DISPLAY-ONLY — they
// never change the case status or the scorecard verdict (that is set by
// adjudication in B6). ResolvedAt marks when work resumed past the blocker.
type Obstacle struct {
	ID                string
	CaseID            string
	Category          ObstacleCategory
	WhatBlocks        string
	WhoUnblocks       string
	ProofTried        string
	RealCount         int // advisory "obstacle is real" tally (derived on read)
	NotConvincedCount int // advisory "not convinced" tally (derived on read)
	UnblockingPlans   []UnblockingPlan
	Adjudication      Adjudication // the authority's verdict (pending until B6 adjudication)
	AdjudicatedAt     *time.Time
	CreatedAt         time.Time
	ResolvedAt        *time.Time
}

// NewObstacle constructs a typed obstacle (tallies start at zero; plans empty;
// adjudication pending until a higher authority rules).
func NewObstacle(caseID string, category ObstacleCategory, whatBlocks, whoUnblocks, proofTried string) Obstacle {
	return Obstacle{
		ID:           idgen.New("blk"),
		CaseID:       caseID,
		Category:     category,
		WhatBlocks:   whatBlocks,
		WhoUnblocks:  whoUnblocks,
		ProofTried:   proofTried,
		Adjudication: AdjudicationPending,
		CreatedAt:    time.Now().UTC(),
	}
}
