package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// Caller is who is asking for a problem. It is zero for an anonymous request —
// GET /api/problems/{id} is unauthenticated, and the router's best-effort
// Authenticate middleware fills these in only when a token happens to be present.
type Caller struct {
	AccountID string
	Role      string
}

// RoleAdmin is the claims role that may screen. It mirrors pkg/auth's role
// vocabulary without importing it — the application layer must not depend on the
// HTTP/JWT plumbing.
const RoleAdmin = "admin"

// GetProblem returns a single problem together with its public audit trail.
// Suggestions (B3) and evidence (B5) are composed by the http layer as empty
// until those contexts exist.
type GetProblem struct {
	problems    domain.Repository
	areas       areadomain.Repository
	admins      Admins
	assignments Assignments
	audit       auditdomain.Repository
}

// NewGetProblem wires the use case.
func NewGetProblem(p domain.Repository, a areadomain.Repository, ad Admins, asgn Assignments, au auditdomain.Repository) *GetProblem {
	return &GetProblem{problems: p, areas: a, admins: ad, assignments: asgn, audit: au}
}

// Execute returns the problem and its audit entries (oldest first), enforcing the
// screening visibility rule: a PendingApproval problem is readable only by its
// reporter (who needs to see what they just filed) and by an admin of its union
// (who has to screen it). Everyone else — including other admins and anonymous
// callers — is told it does not exist.
//
// The refusal is ErrProblemNotFound rather than a forbidden error on purpose. A
// 403 would confirm that a hidden report exists at that id, which is exactly the
// fact the gate is meant to withhold; it would let anyone enumerate what is
// awaiting screening.
// It also returns the assignment view: the official the problem was actually
// handed to, and the reason for an override. Pointed and assigned are two
// different facts — pointed is who the public asked for, assigned is who the
// admin decided — and the gap between them is an override the public record is
// meant to show rather than merely store, which is why the reason travels with it.
// The view is zero when the problem has not reached an official.
//
// The assignment read comes AFTER the visibility gate on purpose. A pending
// report must cost nothing on its way to a 404 — the gate is the first thing
// that runs, and nothing about a hidden problem is fetched behind it.
func (uc *GetProblem) Execute(ctx context.Context, id string, caller Caller) (*domain.Problem, []auditdomain.AuditEntry, AssignmentView, error) {
	p, err := uc.problems.GetByID(ctx, id)
	if err != nil {
		return nil, nil, AssignmentView{}, err
	}
	if !p.Status.PubliclyVisible() && !uc.mayViewPending(ctx, p, caller) {
		return nil, nil, AssignmentView{}, domain.ErrProblemNotFound
	}

	audit, err := uc.audit.ListByTarget(ctx, "problem", id)
	if err != nil {
		return nil, nil, AssignmentView{}, err
	}

	assignment, err := uc.assignments.For(ctx, id)
	if err != nil {
		return nil, nil, AssignmentView{}, err
	}
	return p, audit, assignment, nil
}

// mayViewPending reports whether the caller is allowed to see a problem that has
// not yet cleared screening.
func (uc *GetProblem) mayViewPending(ctx context.Context, p *domain.Problem, caller Caller) bool {
	if caller.AccountID == "" {
		return false
	}
	if caller.AccountID == p.ReporterID {
		return true
	}
	if caller.Role != RoleAdmin {
		return false
	}
	// An admin may only see their own union's pending reports — the same scoping
	// that governs screening itself.
	return ensureUnionAdmin(ctx, uc.areas, uc.admins, caller.AccountID, p.Location.AreaID.String()) == nil
}
