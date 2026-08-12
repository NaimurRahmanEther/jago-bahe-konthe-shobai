package application

import (
	"context"
	"errors"

	"jago-bahe-backend/internal/identity/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// RegisterOfficialInput is the official-registration request. OfficialID names an
// entry that must already exist in the directory — there is no field for a tier
// or an area, because registration cannot invent an office.
type RegisterOfficialInput struct {
	Name       string
	Phone      string
	Password   string
	NID        string
	OfficialID string
}

// RegisterOfficial creates an unverified official account and a pending claim to
// a directory office, and returns a token.
//
// The account exists immediately but can do nothing as that official: the claim
// is what sets accounts.official_id, and every official-scoped route reads the
// office off the token's OfficialID. So a self-declared MP holds an account that
// answers no cases until a human confirms them.
type RegisterOfficial struct {
	accounts  domain.AccountRepository
	officials domain.OfficialRepository
	claims    domain.ClaimRepository
	hasher    domain.Hasher
	tokens    TokenIssuer
	audit     auditdomain.Repository
}

// NewRegisterOfficial wires the use case.
func NewRegisterOfficial(
	a domain.AccountRepository,
	o domain.OfficialRepository,
	c domain.ClaimRepository,
	h domain.Hasher,
	t TokenIssuer,
	au auditdomain.Repository,
) *RegisterOfficial {
	return &RegisterOfficial{accounts: a, officials: o, claims: c, hasher: h, tokens: t, audit: au}
}

// Execute registers the account and files the claim.
func (uc *RegisterOfficial) Execute(ctx context.Context, in RegisterOfficialInput) (*domain.Account, string, error) {
	phone, err := valueobject.NewPhoneNumber(in.Phone)
	if err != nil {
		return nil, "", err
	}
	// The office must already be on the directory. Refusing an unknown id here is
	// what keeps the directory a record of real elections rather than a list of
	// whoever signed up.
	if _, err := uc.officials.GetByID(ctx, in.OfficialID); err != nil {
		return nil, "", domain.ErrOfficialNotFound
	}
	if _, err := uc.accounts.GetByPhone(ctx, phone); err == nil {
		return nil, "", domain.ErrPhoneTaken
	} else if !errors.Is(err, domain.ErrAccountNotFound) {
		return nil, "", err
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return nil, "", err
	}

	acc := domain.NewUnverifiedOfficial(in.Name, phone, hash, in.NID)
	if err := uc.accounts.Create(ctx, acc); err != nil {
		return nil, "", err
	}

	claim := domain.NewOfficialClaim(acc.ID, in.OfficialID)
	if err := uc.claims.Create(ctx, claim); err != nil {
		return nil, "", err
	}

	token, err := uc.tokens.Issue(acc.ID, string(acc.Role), acc.OfficialID)
	if err != nil {
		return nil, "", err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("official", in.OfficialID, acc.ID, "claim_filed", ""))
	return acc, token, nil
}
