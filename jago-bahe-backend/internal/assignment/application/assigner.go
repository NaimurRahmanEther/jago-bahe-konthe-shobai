package application

import (
	"context"
	"time"

	"jago-bahe-backend/internal/assignment/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// assigner holds the deps shared by every path that produces an assignment (the
// union admin's decision and the super admin's forward): it builds the assignment,
// derives the monitor, sets the deadline, flips the problem to Assigned, writes the
// audit entry, and tells the two parties who need to know. Keeping it here avoids
// duplicating that sequence — and means a party told on one path can never be
// silently missed on the other.
type assigner struct {
	assignments     domain.Repository
	problems        Problems
	officials       Officials
	audit           auditdomain.Repository
	notify          notificationdomain.Repository
	defaultDeadline time.Duration // D
}

// create persists an assignment for the chosen official and records action on the
// audit trail. deadline is optional (nil → now + D); priority defaults to normal;
// overrideReason is empty unless the public's choice was overridden.
//
// It takes the whole ProblemView and OfficialView rather than two ids because it
// needs the reporter and the official's name for B21's notifications, and BOTH
// callers already hold both — so this costs no extra query.
func (a *assigner) create(ctx context.Context, actorID string, p ProblemView, chosen OfficialView, priority string, deadline *time.Time, overrideReason, action string) (*domain.Assignment, error) {
	if priority == "" {
		priority = defaultPriority
	}
	due := time.Now().UTC().Add(a.defaultDeadline)
	if deadline != nil {
		due = *deadline
	}
	monitor, err := a.officials.MonitorFor(ctx, chosen.ID)
	if err != nil {
		return nil, err
	}

	record := domain.NewAssignment(p.ID, chosen.ID, monitor, priority, due, overrideReason)
	if err := a.assignments.CreateAssignment(ctx, record); err != nil {
		return nil, err // domain.ErrAlreadyAssigned
	}
	if err := a.problems.MarkAssigned(ctx, p.ID); err != nil {
		return nil, err
	}
	_ = a.audit.Append(ctx, auditdomain.NewAuditEntry("problem", p.ID, actorID, action, overrideReason))

	// B21: two notifications, two id spaces, one place. The reporter is an ACCOUNT
	// and the official is a DIRECTORY OFFICE, and the constructor names say which,
	// so the two cannot be transposed here.
	//
	// Entitlement: the reporter reads their own report (now Assigned, and public);
	// the official is the assignee. Both are told the fact, never the admin's
	// override reason — that is on the public audit trail, where accountability for
	// passing over the public's choice belongs.
	_ = a.notify.Append(ctx, notificationdomain.ForResident(
		p.ReporterID, notificationdomain.TypeProblemAssigned, p.ID, p.Title, chosen.Name))
	_ = a.notify.Append(ctx, notificationdomain.ForOfficial(
		chosen.ID, notificationdomain.TypeCaseAssigned, p.ID, p.Title, due.Format(time.RFC3339)))

	return record, nil
}
