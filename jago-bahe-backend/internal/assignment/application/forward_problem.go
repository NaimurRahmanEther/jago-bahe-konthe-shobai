package application

import (
	"context"
	"strings"
	"time"

	"jago-bahe-backend/internal/assignment/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// ListForwardingQueue is the super admin's decision surface: every above-union
// report waiting to be forwarded, seat-wide, with the union admins' advice on it.
//
// It is seat-wide and takes no scope, because the super admin is the seat's — they
// are not an admin of any union, and the four above-union offices are the seat's
// offices. Reads are never audited (A.4.4): an entry per view would publish who is
// looking at what.
type ListForwardingQueue struct {
	assignments domain.Repository
	problems    Problems
	officials   Officials
	svc         *domain.Service
}

// NewListForwardingQueue wires the use case.
func NewListForwardingQueue(r domain.Repository, p Problems, o Officials, svc *domain.Service) *ListForwardingQueue {
	return &ListForwardingQueue{assignments: r, problems: p, officials: o, svc: svc}
}

// Execute returns the above-union reports awaiting a forward, most-advised first.
func (uc *ListForwardingQueue) Execute(ctx context.Context) ([]ForwardingCandidate, error) {
	candidates, err := forwardingCandidates(ctx, uc.problems, uc.officials, uc.assignments, uc.svc)
	if err != nil {
		return nil, err
	}
	sortForwarding(candidates)
	return candidates, nil
}

// ForwardProblem is the super admin forwarding an above-union report to an
// official (B20). It replaced the admin vote's settlement.
//
// Two rules are load-bearing:
//
//  1. The advice is a SIGNAL, NOT A GATE. There is no minimum number of
//     suggestions and no window to wait out — the super admin may forward a report
//     nobody has advised on. This is deliberately the same rule B17 set for V
//     (A.3.1.1), so the platform stays consistent with itself, and the suggestion
//     count is public either way, which is what makes an early forward judgeable.
//  2. Departing from the advisers or from the reporter requires a PUBLIC REASON,
//     and the decision is then audited as `assignment_overridden` rather than
//     `assigned`. The rule itself lives in domain.RequiresReason — never restate it
//     here, or the audit trail and the rule can disagree.
//
// It reuses the same `assigner` the union path uses, so the monitor, the deadline
// D and the priority are set identically and B19's observation ladder works on
// these cases without knowing they arrived by a different route.
type ForwardProblem struct {
	assigner
	admins domain.Repository // suggestions live on the assignment repository
	svc    *domain.Service
}

// NewForwardProblem wires the use case.
func NewForwardProblem(r domain.Repository, p Problems, o Officials, svc *domain.Service, audit auditdomain.Repository, notify notificationdomain.Repository, deadline time.Duration) *ForwardProblem {
	return &ForwardProblem{
		assigner: assigner{assignments: r, problems: p, officials: o, audit: audit, notify: notify, defaultDeadline: deadline},
		admins:   r,
		svc:      svc,
	}
}

// Execute forwards the report. reason may be empty only when the chosen official
// is both the advisers' top suggestion and the one the reporter pointed it at.
func (uc *ForwardProblem) Execute(ctx context.Context, problemID, superAdminID, officialID, priority string, deadline *time.Time, reason string) (*domain.Assignment, error) {
	p, err := uc.problems.Get(ctx, problemID)
	if err != nil {
		return nil, err // domain.ErrProblemNotFound
	}
	if !p.EligibleForAssignment {
		return nil, domain.ErrNotAssignable
	}

	pointed, err := uc.officials.Get(ctx, p.PointedOfficialID)
	if err != nil {
		return nil, err
	}
	if uc.svc.Route(pointed.Tier).Kind != domain.RouteSuperAdmin {
		return nil, domain.ErrWrongRoute // union-level reports are not the super admin's
	}

	// The target must exist and be above-union: the super admin forwards to the
	// seat's offices, not into a union, whose own admin decides alone.
	target, err := uc.officials.Get(ctx, officialID)
	if err != nil {
		return nil, err // domain.ErrOfficialNotFound
	}
	if uc.svc.Route(target.Tier).Kind != domain.RouteSuperAdmin {
		return nil, domain.ErrWrongRoute
	}

	suggestions, err := uc.admins.SuggestionsByProblem(ctx, problemID)
	if err != nil {
		return nil, err
	}
	top, _ := domain.TopSuggestion(suggestions)

	reason = strings.TrimSpace(reason)
	action := "assigned"
	if domain.RequiresReason(officialID, top, p.PointedOfficialID) {
		if reason == "" {
			return nil, domain.ErrReasonRequired
		}
		action = "assignment_overridden"
	} else {
		// A reason offered where none was required is kept — it is still public
		// rationale — but the action stays `assigned`, because nothing was overridden
		// and the public feed reads these literals to mean exactly that (A.3.5).
		_ = reason
	}

	return uc.create(ctx, superAdminID, p, target, priority, deadline, reason, action)
}
