package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// Evidence is a before/after photo pair attached to a case. At least one piece is
// required before the case can be marked Done (the lifecycle guard). Both photos
// are part of the public record.
type Evidence struct {
	ID             string
	CaseID         string
	BeforeImageURL string
	AfterImageURL  string
	CreatedAt      time.Time
}

// NewEvidence constructs an evidence entry.
func NewEvidence(caseID, beforeImageURL, afterImageURL string) Evidence {
	return Evidence{
		ID:             idgen.New("evd"),
		CaseID:         caseID,
		BeforeImageURL: beforeImageURL,
		AfterImageURL:  afterImageURL,
		CreatedAt:      time.Now().UTC(),
	}
}
