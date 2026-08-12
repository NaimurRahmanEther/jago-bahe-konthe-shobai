package application

import (
	"context"
	"time"

	"jago-bahe-backend/internal/assignment/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

const defaultPriority = "normal"

// AssignWithinUnion is the union-level assignment path: the union's own admin
// confirms the public's pointed official or overrides it (with a required public
// reason). It sets the monitor, priority, and first-response deadline D, flips the
// problem to Assigned, and audits the decision. Above-union tiers are rejected
// here (ErrWrongRoute) and go through OpenAdminVote instead.
type AssignWithinUnion struct {
	assigner
	admins Admins
	areas  areadomain.Repository
	svc    *domain.Service
}

// NewAssignWithinUnion wires the use case.
func NewAssignWithinUnion(r domain.Repository, p Problems, o Officials, ad Admins, areas areadomain.Repository, svc *domain.Service, audit auditdomain.Repository, notify notificationdomain.Repository, deadline time.Duration) *AssignWithinUnion {
	return &AssignWithinUnion{
		assigner: assigner{assignments: r, problems: p, officials: o, audit: audit, notify: notify, defaultDeadline: deadline},
		admins:   ad,
		areas:    areas,
		svc:      svc,
	}
}

// Execute assigns an assignable, union-level problem. "Assignable" is no longer
// "validated": since B17 the admin may forward a Reported problem with any number
// of validations, including none — the count is evidence, not a gate (A.3.1).
// chosenOfficialID defaults to
// the pointed official (a confirm); a different official is an override that
// requires overrideReason. deadline is optional — nil defaults to now + D.
func (uc *AssignWithinUnion) Execute(ctx context.Context, problemID, adminAccountID, chosenOfficialID, priority string, deadline *time.Time, overrideReason string) (*domain.Assignment, error) {
	p, err := uc.problems.Get(ctx, problemID)
	if err != nil {
		return nil, err // domain.ErrProblemNotFound
	}
	if !p.EligibleForAssignment {
		return nil, domain.ErrNotAssignable
	}

	pointed, err := uc.officials.Get(ctx, p.PointedOfficialID)
	if err != nil {
		return nil, err // domain.ErrOfficialNotFound
	}
	if route := uc.svc.Route(pointed.Tier); route.Kind != domain.RouteUnion {
		return nil, domain.ErrWrongRoute // above-union: must open an admin vote
	}

	if err := uc.ensureUnionAdmin(ctx, adminAccountID, p.AreaID); err != nil {
		return nil, err
	}

	if chosenOfficialID == "" {
		chosenOfficialID = p.PointedOfficialID
	}
	// The view was already being fetched to validate the id; it is now kept, because
	// create needs the official's NAME to tell the reporter who their report went to
	// (B21). No extra query.
	chosen, err := uc.officials.Get(ctx, chosenOfficialID)
	if err != nil {
		return nil, err // domain.ErrOfficialNotFound
	}

	reason, err := resolveOverride(p.PointedOfficialID, chosenOfficialID, overrideReason)
	if err != nil {
		return nil, err // domain.ErrMissingOverrideReason
	}

	action := "assigned"
	if reason != "" {
		action = "assignment_overridden"
	}
	return uc.create(ctx, adminAccountID, p, chosen, priority, deadline, reason, action)
}

// ensureUnionAdmin enforces that the acting admin belongs to the problem's union.
func (uc *AssignWithinUnion) ensureUnionAdmin(ctx context.Context, adminAccountID, problemAreaID string) error {
	problemUnion, err := resolveUnion(ctx, uc.areas, problemAreaID)
	if err != nil {
		return err
	}
	adminUnion, err := uc.admins.UnionOf(ctx, adminAccountID)
	if err != nil {
		return err
	}
	if adminUnion == "" || adminUnion != problemUnion {
		return domain.ErrNotUnionAdmin
	}
	return nil
}
