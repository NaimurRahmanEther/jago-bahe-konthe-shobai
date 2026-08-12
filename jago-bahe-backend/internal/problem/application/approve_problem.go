package application

import (
	"context"
	"time"

	"jago-bahe-backend/internal/problem/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// ApproveProblem publishes a pending problem: the union's admin has screened it
// and found it to be a genuine report. It becomes Reported — public, but still
// unvalidated, since clearing the spam gate says nothing about merit. From here
// the community decides via V.
//
// The acting admin must belong to the problem's own union (ensureUnionAdmin);
// auth.RequireAdmin is role-only and knows nothing about geography, so without
// this check any admin could approve any union's reports.
type ApproveProblem struct {
	problems domain.Repository
	areas    areadomain.Repository
	admins   Admins
	audit    auditdomain.Repository
	notify   notificationdomain.Repository
	scr      *domain.Screening
}

// NewApproveProblem wires the use case.
func NewApproveProblem(p domain.Repository, a areadomain.Repository, ad Admins, au auditdomain.Repository, n notificationdomain.Repository, scr *domain.Screening) *ApproveProblem {
	return &ApproveProblem{problems: p, areas: a, admins: ad, audit: au, notify: n, scr: scr}
}

// Execute approves the problem on the acting admin's behalf and audits it.
func (uc *ApproveProblem) Execute(ctx context.Context, problemID, adminAccountID string) (*domain.Problem, error) {
	p, err := uc.problems.GetByID(ctx, problemID)
	if err != nil {
		return nil, err
	}
	if err := ensureUnionAdmin(ctx, uc.areas, uc.admins, adminAccountID, p.Location.AreaID.String()); err != nil {
		return nil, err
	}

	next, err := uc.scr.Approve(p.Status)
	if err != nil {
		return nil, err
	}

	reviewedAt := time.Now().UTC()
	// SetScreening is a compare-and-set on the source status. It is shared with
	// reject; on a race (the problem moved between load and write) it returns
	// ErrNotRejectable, which here reads as "the report is no longer pending" — the
	// domain's Approve guard above is the primary check.
	if err := uc.problems.SetScreening(ctx, problemID, p.Status, next, "", adminAccountID, reviewedAt); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", problemID, adminAccountID, "approved", ""))

	// B21: the reporter has no other surface that tells them their report went
	// public. Until this moment it was visible only on /me, in a status which looks
	// identical to "still waiting" — and the gate is deliberately hard, so waiting
	// is the one outcome that never resolves itself (A.3.1).
	//
	// Entitlement: the recipient is the reporter reading their own report, which
	// they may do at any status; the problem is now Reported and public anyway. That
	// check is owed at every write site, because the title below is SNAPSHOTTED and
	// so bypasses TitlesByIDs's publicly-visible-only filter (A.3.9 constraint 3).
	_ = uc.notify.Append(ctx, notificationdomain.ForResident(
		p.ReporterID, notificationdomain.TypeProblemApproved, p.ID, p.Title, ""))

	p.Status = next
	p.ReviewedBy = adminAccountID
	p.ReviewedAt = &reviewedAt
	return p, nil
}
