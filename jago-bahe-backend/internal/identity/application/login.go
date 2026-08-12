package application

import (
	"context"
	"errors"

	"jago-bahe-backend/internal/identity/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// LoginInput is the login request.
type LoginInput struct {
	Phone    string
	Password string
}

// LoginResult carries the issued token and the authenticated account (for the
// http layer to build the user DTO).
type LoginResult struct {
	Token   string
	Account *domain.Account
}

// Login authenticates a phone + password and issues a token.
type Login struct {
	accounts domain.AccountRepository
	svc      *domain.Service
	hasher   domain.Hasher
	tokens   TokenIssuer
}

// NewLogin wires the use case.
func NewLogin(a domain.AccountRepository, s *domain.Service, h domain.Hasher, t TokenIssuer) *Login {
	return &Login{accounts: a, svc: s, hasher: h, tokens: t}
}

// Execute verifies credentials and returns a token plus the account. An unknown
// account and a wrong password both surface as ErrInvalidCredentials, so neither
// reveals which phones exist. A malformed phone surfaces as ErrInvalidPhone: it
// cannot be anyone's account, so saying so leaks nothing — and calling it a
// credential failure tells the user to doubt a password that was correct.
func (uc *Login) Execute(ctx context.Context, in LoginInput) (*LoginResult, error) {
	phone, err := valueobject.NewPhoneNumber(in.Phone)
	if err != nil {
		return nil, err
	}

	acc, err := uc.accounts.GetByPhone(ctx, phone)
	if errors.Is(err, domain.ErrAccountNotFound) {
		return nil, domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := uc.svc.Authenticate(acc, in.Password, uc.hasher); err != nil {
		return nil, err
	}

	token, err := uc.tokens.Issue(acc.ID, string(acc.Role), acc.OfficialID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, Account: acc}, nil
}
