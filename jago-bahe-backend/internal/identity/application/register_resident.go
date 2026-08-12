package application

import (
	"context"
	"errors"

	"jago-bahe-backend/internal/identity/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// RegisterResidentInput is the register use-case request.
type RegisterResidentInput struct {
	Name     string
	Phone    string
	Password string
	NID      string
	UnionID  string
}

// RegisterResident creates a new, unverified resident account and returns a token.
type RegisterResident struct {
	accounts domain.AccountRepository
	hasher   domain.Hasher
	tokens   TokenIssuer
	audit    auditdomain.Repository
}

// NewRegisterResident wires the use case.
func NewRegisterResident(a domain.AccountRepository, h domain.Hasher, t TokenIssuer, au auditdomain.Repository) *RegisterResident {
	return &RegisterResident{accounts: a, hasher: h, tokens: t, audit: au}
}

// Execute validates input, ensures the phone is free, hashes the password,
// persists the resident, audits, and issues a token.
func (uc *RegisterResident) Execute(ctx context.Context, in RegisterResidentInput) (string, error) {
	phone, err := valueobject.NewPhoneNumber(in.Phone)
	if err != nil {
		return "", err
	}

	switch _, err := uc.accounts.GetByPhone(ctx, phone); {
	case err == nil:
		return "", domain.ErrPhoneTaken
	case !errors.Is(err, domain.ErrAccountNotFound):
		return "", err
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return "", err
	}

	acc := domain.NewResident(in.Name, phone, hash, in.NID, valueobject.AreaID(in.UnionID))
	if err := uc.accounts.Create(ctx, acc); err != nil {
		return "", err
	}

	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("resident", acc.ID, acc.ID, "registered", ""))

	return uc.tokens.Issue(acc.ID, string(acc.Role), acc.OfficialID)
}
