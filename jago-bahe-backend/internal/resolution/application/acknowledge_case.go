package application

import (
	"context"
	"strings"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// AcknowledgeCase records the official's response to an assignment: accept (→
// Acknowledged, solving begins) or dispute (→ Disputed, with a reason). Only the
// assigned official may act, and only from the Assigned state. The decision is a
// closed enum: anything else is refused, never defaulted to either branch.
type AcknowledgeCase struct {
	repo  domain.Repository
	audit auditdomain.Repository
}

// NewAcknowledgeCase wires the use case.
func NewAcknowledgeCase(r domain.Repository, audit auditdomain.Repository) *AcknowledgeCase {
	return &AcknowledgeCase{repo: r, audit: audit}
}

// Execute applies accept or dispute. An unrecognised decision is refused rather
// than defaulted, and a dispute must carry a reason.
func (uc *AcknowledgeCase) Execute(ctx context.Context, caseID, officialID string, decision domain.Decision, reason string) (*domain.Case, error) {
	if !decision.Valid() {
		return nil, domain.ErrInvalidDecision
	}
	// Disputing drops the case off this official's public scorecard, so it is the
	// most consequential thing they can do to it. Doing that without saying why
	// would be the silent rejection the design refuses everywhere else — an
	// official may push back, but never without the public seeing the grounds.
	reason = strings.TrimSpace(reason)
	if decision == domain.DecisionDispute && reason == "" {
		return nil, domain.ErrEmptyText
	}

	c, err := loadOwnedCase(ctx, uc.repo, caseID, officialID)
	if err != nil {
		return nil, err
	}

	event := domain.EventAcknowledge
	if decision == domain.DecisionDispute {
		event = domain.EventDispute
	}
	next, err := domain.Transition(c.Status, event, domain.Guards{})
	if err != nil {
		return nil, err
	}
	c.Status = next

	action := "acknowledged"
	if decision == domain.DecisionAccept {
		now := time.Now().UTC()
		c.AcknowledgedAt = &now
	} else {
		c.DisputeReason = reason
		action = "disputed"
	}

	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, officialID, action, c.DisputeReason))
	return c, nil
}
