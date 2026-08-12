package application

import (
	"context"
	"strings"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// UploadEvidence attaches a before/after photo pair to a case. Allowed only while
// the case is workable; it does not change status, but it satisfies the guard
// MarkDone requires. Both photos are part of the public record.
type UploadEvidence struct {
	repo  domain.Repository
	audit auditdomain.Repository
}

// NewUploadEvidence wires the use case.
func NewUploadEvidence(r domain.Repository, audit auditdomain.Repository) *UploadEvidence {
	return &UploadEvidence{repo: r, audit: audit}
}

// Execute stores the evidence.
func (uc *UploadEvidence) Execute(ctx context.Context, caseID, officialID, beforeImageURL, afterImageURL string) (*domain.Case, error) {
	if strings.TrimSpace(beforeImageURL) == "" || strings.TrimSpace(afterImageURL) == "" {
		return nil, domain.ErrEmptyText
	}
	c, err := loadOwnedCase(ctx, uc.repo, caseID, officialID)
	if err != nil {
		return nil, err
	}
	if !domain.IsWorkable(c.Status) {
		return nil, domain.ErrIllegalTransition
	}

	c.Evidence = append(c.Evidence, domain.NewEvidence(c.ID, beforeImageURL, afterImageURL))
	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, officialID, "evidence_uploaded", ""))
	return c, nil
}
