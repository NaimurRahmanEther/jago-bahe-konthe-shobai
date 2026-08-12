package application

import (
	"context"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// MarkDone moves a case from InProgress to Done. Evidence is required (the
// lifecycle guard), and the official can only reach Done — never Resolved; the
// reporting residents confirm the fix (B6).
type MarkDone struct {
	repo     domain.Repository
	problems Problems
	audit    auditdomain.Repository
	notify   notificationdomain.Repository
}

// NewMarkDone wires the use case.
func NewMarkDone(r domain.Repository, p Problems, audit auditdomain.Repository, n notificationdomain.Repository) *MarkDone {
	return &MarkDone{repo: r, problems: p, audit: audit, notify: n}
}

// Execute marks the case done if evidence is attached.
func (uc *MarkDone) Execute(ctx context.Context, caseID, officialID string) (*domain.Case, error) {
	c, err := loadOwnedCase(ctx, uc.repo, caseID, officialID)
	if err != nil {
		return nil, err
	}

	next, err := domain.Transition(c.Status, domain.EventMarkDone, domain.Guards{HasEvidence: c.HasEvidence()})
	if err != nil {
		return nil, err // domain.ErrEvidenceRequired or domain.ErrIllegalTransition
	}
	c.Status = next

	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = uc.problems.SetStatus(ctx, c.ProblemID, string(domain.StatusDone))
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, officialID, "marked_done", ""))

	// B21: the highest-value notification in the platform. The reporting resident is
	// the ONLY party who can move this case to Resolved — an official may never
	// close their own (Scaffold §2) — so a loop which ends here ends because nobody
	// told them they were being asked.
	//
	// It fires AGAIN on a second Done after a reopen, and that is correct: the
	// second is a new claim of completion about work done since the first was
	// rejected. Suppressing it would leave the reporter holding a notification they
	// may already have marked read, with no signal that the official had answered
	// them. Same reasoning idx_case_observations_open is partial for; see the
	// migration, which declines a unique index in those words.
	if reporterID, err := uc.problems.ReporterID(ctx, c.ProblemID); err == nil {
		_ = uc.notify.Append(ctx, notificationdomain.ForResident(
			reporterID, notificationdomain.TypeConfirmationRequested, c.ProblemID,
			notificationTitle(ctx, uc.problems, c.ProblemID), ""))
	}
	return c, nil
}
