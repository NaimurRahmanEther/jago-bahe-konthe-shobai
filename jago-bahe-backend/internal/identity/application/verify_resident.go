package application

import (
	"context"

	"jago-bahe-backend/internal/identity/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// VerifyResident marks a resident account as verified, on the authority of their
// own union's admin.
//
// This is the gate the whole community filter rests on. Only verified residents
// may validate, suggest, or upvote, so verification is what makes the threshold V
// a count of real neighbours rather than of accounts — without it, anyone could
// manufacture agreement by registering five times.
//
// An admin may verify only residents of their own union: the point is that
// someone local vouches for a local person, which an admin from another union
// cannot do.
type VerifyResident struct {
	accounts  domain.AccountRepository
	officials domain.OfficialRepository
	areas     areadomain.Repository
	audit     auditdomain.Repository
}

// NewVerifyResident wires the use case.
func NewVerifyResident(a domain.AccountRepository, o domain.OfficialRepository, ar areadomain.Repository, au auditdomain.Repository) *VerifyResident {
	return &VerifyResident{accounts: a, officials: o, areas: ar, audit: au}
}

// ListPending returns the unverified residents of the admin's own union.
func (uc *VerifyResident) ListPending(ctx context.Context, adminAccountID string) ([]domain.Account, error) {
	union, err := uc.adminUnion(ctx, adminAccountID)
	if err != nil {
		return nil, err
	}
	return uc.accounts.ListUnverifiedResidents(ctx, valueobject.AreaID(union))
}

// Execute verifies the account, writing an audit entry. Verifying an already
// verified resident is a no-op rather than an error, so a double-click cannot
// fail; the guard that matters is the union check.
func (uc *VerifyResident) Execute(ctx context.Context, accountID, adminAccountID string) error {
	adminUnion, err := uc.adminUnion(ctx, adminAccountID)
	if err != nil {
		return err
	}

	acc, err := uc.accounts.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	// Only residents are verified this way. An official's standing comes from an
	// approved claim, and letting a union admin flip an official's verified flag
	// directly would route around the tier check that keeps above-union claims
	// with the super admin.
	if acc.Role != domain.RoleResident {
		return domain.ErrNotAResident
	}
	if acc.UnionID.String() != adminUnion {
		return domain.ErrNotUnionAdmin
	}
	if acc.Verified {
		return nil
	}

	if err := uc.accounts.SetVerified(ctx, accountID, true); err != nil {
		return err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("resident", accountID, adminAccountID, "verified", ""))
	return nil
}

// adminUnion resolves the acting admin's union, failing closed when they have
// none on record.
func (uc *VerifyResident) adminUnion(ctx context.Context, adminAccountID string) (string, error) {
	union, err := AdminUnion(ctx, uc.accounts, uc.officials, uc.areas, adminAccountID)
	if err != nil {
		return "", err
	}
	if union == "" {
		return "", domain.ErrNotUnionAdmin
	}
	return union, nil
}
