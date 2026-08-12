package domain

import "context"

// Repository is the port for the audit log. It is append-only with exactly one
// exception: DeleteByTarget, below. Implementations must never update an entry,
// and must never delete one except through that method.
type Repository interface {
	Append(ctx context.Context, entry AuditEntry) error
	ListByTarget(ctx context.Context, targetType, targetID string) ([]AuditEntry, error)

	// DeleteByTarget erases every entry for one target. It is the single hole in
	// the append-only rule and exists for exactly one caller: the reporter's hard
	// delete of their own problem (problem/application/delete_problem.go), which
	// removes the problem row and is required to leave no trace of it behind.
	//
	// Understand what this costs before calling it from anywhere else. The audit
	// log is the platform's proof that a state change happened and who made it;
	// entries erased here make the deletion itself unprovable — afterwards nothing
	// records that the target ever existed. Every other takedown path (an admin's
	// rejection, a reporter's withdrawal) deliberately keeps its trail and marks a
	// status instead. Prefer those. See CLAUDE.md A.3.3.
	DeleteByTarget(ctx context.Context, targetType, targetID string) error
	// ListByActions returns recent entries for any of the given actions, newest
	// first, capped at limit. It serves the super admin's oversight view: the audit
	// log is already the record of every state change, so oversight reads it rather
	// than keeping a second store that could disagree with it.
	ListByActions(ctx context.Context, actions []string, limit int) ([]AuditEntry, error)
}
