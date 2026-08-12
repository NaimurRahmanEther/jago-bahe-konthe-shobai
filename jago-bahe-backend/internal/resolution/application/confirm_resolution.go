package application

import (
	"context"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// ConfirmResolution closes the loop: the reporting resident judges a Done case —
// solved → Resolved (feeds the scorecard), not_solved → Reopened → back to
// InProgress. Only the reporting resident may confirm (an official can never
// close their own case).
type ConfirmResolution struct {
	repo     domain.Repository
	problems Problems
	audit    auditdomain.Repository
	notify   notificationdomain.Repository
}

// NewConfirmResolution wires the use case.
func NewConfirmResolution(r domain.Repository, p Problems, audit auditdomain.Repository, n notificationdomain.Repository) *ConfirmResolution {
	return &ConfirmResolution{repo: r, problems: p, audit: audit, notify: n}
}

// Execute records the resident's verdict on the problem's Done case.
func (uc *ConfirmResolution) Execute(ctx context.Context, problemID, residentID string, outcome domain.ConfirmationOutcome) (*domain.Case, error) {
	if !outcome.Valid() {
		return nil, domain.ErrInvalidOutcome
	}
	reporterID, err := uc.problems.ReporterID(ctx, problemID)
	if err != nil {
		return nil, err // domain.ErrProblemNotFound
	}
	if reporterID != residentID {
		return nil, domain.ErrNotReporter
	}
	c, err := uc.repo.GetByProblem(ctx, problemID)
	if err != nil {
		return nil, err // domain.ErrCaseNotFound
	}

	event := domain.EventConfirmResolved
	if outcome == domain.ConfirmationNotSolved {
		event = domain.EventConfirmReopened
	}
	next, err := domain.Transition(c.Status, event, domain.Guards{})
	if err != nil {
		return nil, err // domain.ErrIllegalTransition (not Done)
	}
	c.Status = next

	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	if err := uc.repo.AddConfirmation(ctx, domain.NewConfirmation(problemID, residentID, outcome)); err != nil {
		return nil, err
	}

	action := "confirmed_resolved"
	if outcome == domain.ConfirmationNotSolved {
		action = "confirmed_reopened"
	}
	_ = uc.problems.SetStatus(ctx, problemID, string(c.Status))
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", problemID, residentID, action, ""))

	// B21: ONLY the reopen is notified, and the asymmetry is decided rather than
	// overlooked. A `solved` confirmation is good news the official reads on their
	// own dashboard; a reopen is a demand for more work on a case they believed was
	// finished, and it is the one the loop stalls without.
	//
	// The recipient is the ASSIGNEE — a DIRECTORY OFFICE id, not an account — which
	// is why the constructor is ForOfficial. Entitlement: the case is theirs.
	if outcome == domain.ConfirmationNotSolved {
		_ = uc.notify.Append(ctx, notificationdomain.ForOfficial(
			c.OfficialID, notificationdomain.TypeCaseReopened, problemID,
			notificationTitle(ctx, uc.problems, problemID), ""))
	}
	return c, nil
}
