package domain

import (
	"context"
	"errors"
)

// Sentinel errors mapped to HTTP status codes in one place by the http layer.
var (
	ErrProblemNotFound  = errors.New("problem not found")
	ErrOfficialNotFound = errors.New("official not found")
	// ErrNotAssignable covers every status outside the forwarding window: an
	// unscreened report (PendingApproval), a terminal one (Rejected/Withdrawn), and
	// anything already in a case. It is deliberately NOT "not validated" — since
	// B17 a Reported problem with zero votes is assignable, so the old name named a
	// rule that no longer exists.
	ErrNotAssignable         = errors.New("problem cannot be assigned in its current state")
	ErrAlreadyAssigned       = errors.New("problem is already assigned")
	ErrMissingOverrideReason = errors.New("overriding the public's choice requires a public reason")
	ErrWrongRoute            = errors.New("this tier is decided by a different route")
	ErrNotUnionAdmin         = errors.New("only an admin of this problem's union may decide it")
	ErrNotEligibleAdviser    = errors.New("admin is not entitled to advise on this problem")
	ErrInvalidScope          = errors.New("invalid advice scope")
	ErrAssignmentNotFound    = errors.New("assignment not found")
	// ErrReasonRequired guards the B20 forwarding rule: a super admin forwarding an
	// above-union report to an official who is neither the advisers' top suggestion
	// nor the one the reporter pointed it at owes the public a written reason.
	ErrReasonRequired = errors.New("forwarding against the advice or the reporter requires a public reason")
)

// Repository is the port for assignment persistence: assignment records and the
// union admins' advisory forwarding suggestions that inform the above-union ones.
// One aggregate is loaded and saved whole (Backend golden rule A.4.3).
type Repository interface {
	// CreateAssignment persists a new assignment; returns ErrAlreadyAssigned if
	// the problem already has one (the distinctness invariant).
	CreateAssignment(ctx context.Context, a *Assignment) error
	GetAssignmentByProblem(ctx context.Context, problemID string) (*Assignment, error)
	// AssignmentsByProblems returns the assignment for each of the given problems,
	// keyed by problem id and omitting the ones that have none. It exists so a
	// whole page of the public feed can name its officials in ONE query — the
	// per-row alternative is an N+1, which is why the field was detail-only until
	// now. Mirrors ProblemRepository.VotesByViewer, the same shape for myVote.
	AssignmentsByProblems(ctx context.Context, problemIDs []string) (map[string]Assignment, error)
	// ListByOfficial returns every assignment held by an official (the resolution
	// context reads these to build the official's case inbox).
	ListByOfficial(ctx context.Context, officialID string) ([]Assignment, error)

	// --- B20: the union admins' advice on above-union forwarding ---

	// UpsertSuggestion records one admin's advice, replacing their previous advice
	// on the same problem. Advice settles nothing, so an admin who is persuaded by
	// the others may change it — the deliberate difference from the ballot this
	// replaced, which refused a second cast because it WOULD have changed a result.
	UpsertSuggestion(ctx context.Context, s *Suggestion) error
	// SuggestionsByProblem returns every admin's advice on one problem, oldest first.
	SuggestionsByProblem(ctx context.Context, problemID string) ([]Suggestion, error)
	// SuggestionsByProblems returns the advice on many problems in ONE query, keyed
	// by problem id — the page-decoration shape (A.3.4). Never call the singular
	// form per row.
	SuggestionsByProblems(ctx context.Context, problemIDs []string) (map[string][]Suggestion, error)
}
