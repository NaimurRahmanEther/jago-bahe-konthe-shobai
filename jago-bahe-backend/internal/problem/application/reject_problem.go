package application

import (
	"context"
	"time"

	"jago-bahe-backend/internal/problem/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// RejectProblem takes a published problem down as spam, abuse, a duplicate, or
// wrong-area. The ground is a closed enum (the domain enforces it) so an admin
// cannot reject on merit; the problem stays publicly visible as Rejected with its
// reason attached, so a wrongly-buried report keeps a public witness.
//
// This is a remedy, not a gate: the problem was readable and votable from the
// moment it was filed, and the domain refuses a takedown once it has been assigned
// (Status.Rejectable).
type RejectProblem struct {
	problems domain.Repository
	areas    areadomain.Repository
	admins   Admins
	audit    auditdomain.Repository
	notify   notificationdomain.Repository
	scr      *domain.Screening
}

// NewRejectProblem wires the use case.
func NewRejectProblem(p domain.Repository, a areadomain.Repository, ad Admins, au auditdomain.Repository, n notificationdomain.Repository, scr *domain.Screening) *RejectProblem {
	return &RejectProblem{problems: p, areas: a, admins: ad, audit: au, notify: n, scr: scr}
}

// Execute rejects the problem on the given ground and audits it. note is an
// optional free-text elaboration; it never substitutes for the enum ground, and
// it is recorded on the public audit entry, not hidden.
func (uc *RejectProblem) Execute(ctx context.Context, problemID, adminAccountID string, reason domain.RejectionReason, note string) (*domain.Problem, error) {
	p, err := uc.problems.GetByID(ctx, problemID)
	if err != nil {
		return nil, err
	}
	if err := ensureUnionAdmin(ctx, uc.areas, uc.admins, adminAccountID, p.Location.AreaID.String()); err != nil {
		return nil, err
	}

	next, err := uc.scr.Reject(p.Status, reason)
	if err != nil {
		return nil, err
	}

	reviewedAt := time.Now().UTC()
	if err := uc.problems.SetScreening(ctx, problemID, p.Status, next, reason, adminAccountID, reviewedAt); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", problemID, adminAccountID, "rejected", auditReason(reason, note)))

	// B21: the GROUND travels, never the admin's free-text note. The enum is what
	// the admin was permitted to reject on, and the frontend already renders the
	// four grounds in Bangla (problem.rejectionReason.*); the note is prose, and it
	// is on the public audit entry the reporter reaches from the report itself.
	//
	// Entitlement: their own report, and a Rejected report stays publicly visible
	// with its reason precisely so a wrongly-buried one keeps a witness.
	_ = uc.notify.Append(ctx, notificationdomain.ForResident(
		p.ReporterID, notificationdomain.TypeProblemRejected, p.ID, p.Title, string(reason)))

	p.Status = next
	p.RejectionReason = reason
	p.ReviewedBy = adminAccountID
	p.ReviewedAt = &reviewedAt
	return p, nil
}

// auditReason renders the ground and any note into the trail's reason field. The
// ground always leads, so the public record shows what an admin was permitted to
// reject on rather than only their prose.
func auditReason(reason domain.RejectionReason, note string) string {
	if note == "" {
		return string(reason)
	}
	return string(reason) + ": " + note
}
