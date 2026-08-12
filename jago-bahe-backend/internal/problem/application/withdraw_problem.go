package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// WithdrawProblem lets a reporter take their own problem down before it is
// assigned. It is the reporter's counterpart to an admin rejection, and it obeys
// the same principle: the problem does not vanish. Its status becomes Withdrawn,
// it stays on the public record, and the retraction's reason is written to the
// public audit trail.
//
// This is the accountable retraction, and it is the one to prefer. A reporter may
// ALSO erase a report outright — see DeleteProblem, which removes the row, the
// cascade, and the audit entries, at any status. That path exists by explicit
// decision (CLAUDE.md A.3.3); this one is what the platform's design argues for.
type WithdrawProblem struct {
	problems domain.Repository
	audit    auditdomain.Repository
}

// NewWithdrawProblem wires the use case.
func NewWithdrawProblem(p domain.Repository, au auditdomain.Repository) *WithdrawProblem {
	return &WithdrawProblem{problems: p, audit: au}
}

// Execute withdraws the problem and audits it. note is an optional free-text
// reason, recorded on the audit entry (there is no enum here — a reporter owes no
// fixed ground for retracting their own report). Returns ErrNotReporter if the
// caller does not own the problem and ErrNotWithdrawable once it has been assigned.
func (uc *WithdrawProblem) Execute(ctx context.Context, problemID, callerID, note string) (*domain.Problem, error) {
	p, err := uc.problems.GetByID(ctx, problemID)
	if err != nil {
		return nil, err
	}
	if callerID == "" || p.ReporterID != callerID {
		return nil, domain.ErrNotReporter
	}
	if !p.Status.Withdrawable() {
		return nil, domain.ErrNotWithdrawable
	}

	if err := uc.problems.SetWithdrawn(ctx, problemID, p.Status); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", problemID, callerID, "withdrawn", note))

	p.Status = domain.StatusWithdrawn
	return p, nil
}
