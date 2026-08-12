package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// DeleteProblem permanently erases a reporter's own problem: the row, everything
// that cascades from it, and its audit trail.
//
// It is deliberately unlike every other takedown on this platform. An admin's
// rejection and the reporter's own withdrawal both keep the report on the public
// record under a terminal status, with the reason in the public audit log, because
// a takedown here is meant to be accountable rather than a disappearance. This
// erases instead. Two consequences follow and neither is a bug:
//
//   - Everything attached to the problem goes with it, by ON DELETE CASCADE — the
//     validation votes other residents cast, the suggestions they wrote, and, if the
//     problem ever reached an official, the case: their plan, their progress updates,
//     their evidence photos. That also silently changes the official's scorecard,
//     which is computed from those case rows.
//   - Because the audit entries go too, afterwards nothing records that the problem
//     existed at all. The deletion is not itself provable.
//
// There is no status window: a reporter may delete at any point in the lifecycle,
// including a Resolved case. That was decided knowingly — see CLAUDE.md A.3.3.
// Prefer WithdrawProblem wherever a retraction will do.
type DeleteProblem struct {
	problems domain.Repository
	audit    auditdomain.Repository
}

// NewDeleteProblem wires the use case.
func NewDeleteProblem(p domain.Repository, au auditdomain.Repository) *DeleteProblem {
	return &DeleteProblem{problems: p, audit: au}
}

// Execute deletes the problem and its audit trail. Returns ErrProblemNotFound if
// no such problem exists and ErrNotReporter if the caller does not own it.
//
// Ownership is checked before anything is written, so a stranger's attempt never
// reaches storage.
func (uc *DeleteProblem) Execute(ctx context.Context, problemID, callerID string) error {
	p, err := uc.problems.GetByID(ctx, problemID)
	if err != nil {
		return err
	}
	if callerID == "" || p.ReporterID != callerID {
		return domain.ErrNotReporter
	}

	// The two writes are not atomic, and cannot be with the repository layer as it
	// stands: there is no transaction helper anywhere in this codebase, and these
	// two tables belong to two repositories holding their own pools.
	//
	// So the ORDER is the safety property, and it is this way round on purpose. If
	// the audit erasure below fails, some entries survive pointing at a problem that
	// is gone — untidy, harmless, and cleanable by re-running the delete.
	// Audit-first would mean a failure here destroys the trail of a problem that
	// still exists, which is unrecoverable corruption of a live public record.
	// Do not "tidy" these into the other order.
	if err := uc.problems.Delete(ctx, problemID); err != nil {
		return err
	}
	return uc.audit.DeleteByTarget(ctx, "problem", problemID)
}
