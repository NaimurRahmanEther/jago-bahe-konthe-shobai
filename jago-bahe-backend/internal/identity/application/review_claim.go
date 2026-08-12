package application

import (
	"context"
	"strings"
	"time"

	"jago-bahe-backend/internal/identity/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// PendingClaim is one entry in a reviewer's queue: the claim plus the office and
// person it concerns, so the UI need not fetch each separately.
type PendingClaim struct {
	Claim        domain.OfficialClaim
	OfficialName string
	OfficialTier string
	AreaID       string
	ClaimantName string
	ClaimantNID  string
}

// ReviewClaims serves the claim queue and the approve/reject decisions for both
// admin kinds. One use case rather than two, because who may decide a claim is a
// property of the office's tier — not of which URL the caller reached for. Union
// admins get their own union's union-level claims; the super admin gets every
// above-union claim.
type ReviewClaims struct {
	accounts  domain.AccountRepository
	officials domain.OfficialRepository
	claims    domain.ClaimRepository
	areas     areadomain.Repository
	audit     auditdomain.Repository
}

// NewReviewClaims wires the use case.
func NewReviewClaims(
	a domain.AccountRepository,
	o domain.OfficialRepository,
	c domain.ClaimRepository,
	ar areadomain.Repository,
	au auditdomain.Repository,
) *ReviewClaims {
	return &ReviewClaims{accounts: a, officials: o, claims: c, areas: ar, audit: au}
}

// ListPending returns the pending claims this caller may decide.
func (uc *ReviewClaims) ListPending(ctx context.Context, reviewerID string, reviewerRole domain.Role) ([]PendingClaim, error) {
	pending, err := uc.claims.ListPending(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]PendingClaim, 0, len(pending))
	for i := range pending {
		c := pending[i]
		off, err := uc.officials.GetByID(ctx, c.OfficialID)
		if err != nil {
			continue // an office that vanished cannot be claimed; leave it out
		}
		if err := uc.mayDecide(ctx, reviewerID, reviewerRole, off); err != nil {
			continue
		}
		claimant, err := uc.accounts.GetByID(ctx, c.AccountID)
		if err != nil {
			continue
		}
		out = append(out, PendingClaim{
			Claim:        c,
			OfficialName: off.Name,
			OfficialTier: string(off.Tier),
			AreaID:       off.AreaID.String(),
			ClaimantName: claimant.Name,
			ClaimantNID:  claimant.NID,
		})
	}
	return out, nil
}

// Approve confirms the claim and binds the account to the office. This is the
// only runtime path that sets accounts.official_id — until it runs, the claimant
// can act as nobody.
func (uc *ReviewClaims) Approve(ctx context.Context, claimID, reviewerID string, reviewerRole domain.Role) (*domain.OfficialClaim, error) {
	c, _, err := uc.load(ctx, claimID, reviewerID, reviewerRole)
	if err != nil {
		return nil, err
	}
	if err := domain.DecideClaim(c.Status, domain.ClaimApproved); err != nil {
		return nil, err
	}

	at := time.Now().UTC()
	// Decide carries the one-approved-claim-per-office rule: the database refuses
	// a second holder, so two reviewers cannot both hand out the same seat.
	//
	// The order matters, and these three writes are not in one transaction. Recording
	// the decision first means a failure below leaves a claim marked approved whose
	// account is not yet bound — the person has less power than the record shows, and
	// an operator can see the gap. Binding first would fail the other way: an account
	// holding an office with no approved claim explaining why, which is power without
	// a record. Fail towards too little authority, never too much.
	// TODO(contract): fold these into one transaction once the repositories expose one.
	if err := uc.claims.Decide(ctx, claimID, domain.ClaimApproved, reviewerID, "", at); err != nil {
		return nil, err
	}
	if err := uc.accounts.SetOfficialID(ctx, c.AccountID, c.OfficialID); err != nil {
		return nil, err
	}
	if err := uc.accounts.SetVerified(ctx, c.AccountID, true); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("official", c.OfficialID, reviewerID, "claim_approved", ""))

	c.Status = domain.ClaimApproved
	c.ReviewedBy = reviewerID
	c.ReviewedAt = &at
	return c, nil
}

// Reject refuses the claim. The reason is required and shown to the claimant —
// a person told they are not who they say they are is owed the grounds.
func (uc *ReviewClaims) Reject(ctx context.Context, claimID, reviewerID string, reviewerRole domain.Role, reason string) (*domain.OfficialClaim, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, domain.ErrEmptyReason
	}
	c, _, err := uc.load(ctx, claimID, reviewerID, reviewerRole)
	if err != nil {
		return nil, err
	}
	if err := domain.DecideClaim(c.Status, domain.ClaimRejected); err != nil {
		return nil, err
	}

	at := time.Now().UTC()
	if err := uc.claims.Decide(ctx, claimID, domain.ClaimRejected, reviewerID, reason, at); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("official", c.OfficialID, reviewerID, "claim_rejected", reason))

	c.Status = domain.ClaimRejected
	c.ReviewedBy = reviewerID
	c.ReviewedAt = &at
	c.Reason = reason
	return c, nil
}

// load fetches the claim and its office, and checks the caller may decide it.
func (uc *ReviewClaims) load(ctx context.Context, claimID, reviewerID string, reviewerRole domain.Role) (*domain.OfficialClaim, *domain.Official, error) {
	c, err := uc.claims.GetByID(ctx, claimID)
	if err != nil {
		return nil, nil, err
	}
	off, err := uc.officials.GetByID(ctx, c.OfficialID)
	if err != nil {
		return nil, nil, domain.ErrOfficialNotFound
	}
	if err := uc.mayDecide(ctx, reviewerID, reviewerRole, off); err != nil {
		return nil, nil, err
	}
	return c, off, nil
}

// mayDecide applies the tier routing: a union-level office is confirmed by that
// union's own admin (who knows these people), and anything above the union by the
// super admin (no union admin has standing over a seat-wide office).
//
// The route is derived from the office, never from the caller's role — so an
// admin cannot reach an above-union claim, and the super admin cannot reach a
// union-level one. Their powers are disjoint, not nested.
func (uc *ReviewClaims) mayDecide(ctx context.Context, reviewerID string, reviewerRole domain.Role, off *domain.Official) error {
	switch domain.RouteClaim(off.Tier) {
	case domain.ClaimRouteSuperAdmin:
		if reviewerRole != domain.RoleSuperAdmin {
			return domain.ErrNotClaimApprover
		}
		return nil

	case domain.ClaimRouteUnionAdmin:
		if reviewerRole != domain.RoleAdmin {
			return domain.ErrNotClaimApprover
		}
		officeUnion, err := areadomain.ResolveUnion(ctx, uc.areas, off.AreaID.String())
		if err != nil {
			return domain.ErrNotClaimApprover
		}
		reviewerUnion, err := AdminUnion(ctx, uc.accounts, uc.officials, uc.areas, reviewerID)
		if err != nil {
			return err
		}
		if reviewerUnion == "" || reviewerUnion != officeUnion {
			return domain.ErrNotClaimApprover
		}
		return nil
	}
	return domain.ErrNotClaimApprover
}
