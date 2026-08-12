package domain

import (
	"context"
	"errors"
	"time"

	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// Sentinel errors mapped to HTTP status codes in one place by the http layer.
var (
	ErrAccountNotFound    = errors.New("account not found")
	ErrPhoneTaken         = errors.New("phone already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrOfficialNotFound   = errors.New("official not found")
	ErrInvalidUnion       = errors.New("union does not exist")
	ErrNotUnionAdmin      = errors.New("you can only act within your own union")
	ErrNotAResident       = errors.New("this account is not a resident")
	ErrEmptyReason        = errors.New("a reason is required")
)

// AccountRepository is the port for account persistence.
type AccountRepository interface {
	Create(ctx context.Context, a *Account) error
	GetByPhone(ctx context.Context, phone valueobject.PhoneNumber) (*Account, error)
	GetByID(ctx context.Context, id string) (*Account, error)
	SetVerified(ctx context.Context, id string, verified bool) error
	// ListByRole returns every account holding the given role (e.g. the admin
	// voter pool for an above-union assignment vote).
	ListByRole(ctx context.Context, role Role) ([]Account, error)
	// ListUnverifiedResidents returns the unverified residents of a union — the
	// queue their union's admin works through.
	ListUnverifiedResidents(ctx context.Context, unionID valueobject.AreaID) ([]Account, error)
	// SetOfficialID binds an account to a directory office. This is what an
	// approved claim does, and it is the only runtime path to that link.
	SetOfficialID(ctx context.Context, accountID, officialID string) error

	// NamesByIDs resolves account ids to display names, one query for a whole
	// page (A.3.4). It serves the public activity feed, which would otherwise
	// print raw ids like `acct-admin-aranagar` at a reader.
	//
	// Ids it does not recognise are simply absent from the map rather than an
	// error: the feed's actors include directory offices and the literal
	// "system", neither of which is an account.
	NamesByIDs(ctx context.Context, ids []string) (map[string]string, error)
}

// OfficialRepository is the port for the read-only officials directory. It stays
// read-only by design: the directory records who won a real election, and an
// official joins by claiming an entry rather than creating one, so nothing in the
// application may write an office into existence.
type OfficialRepository interface {
	List(ctx context.Context) ([]Official, error)
	GetByID(ctx context.Context, id string) (*Official, error)

	// NamesByIDs resolves directory office ids to office-holder names, the
	// companion to AccountRepository.NamesByIDs. Both are needed because an audit
	// entry's actor is an ACCOUNT id for an admin's action but an OFFICE id for an
	// official's — the two id spaces meet in that one column.
	NamesByIDs(ctx context.Context, ids []string) (map[string]string, error)
}

// ClaimRepository is the port for official claims.
type ClaimRepository interface {
	Create(ctx context.Context, c *OfficialClaim) error
	GetByID(ctx context.Context, id string) (*OfficialClaim, error)
	// ListPending returns every claim awaiting a decision, newest first. Callers
	// filter it to the claims they may act on (the tier routing lives in the use
	// case, not in SQL).
	ListPending(ctx context.Context) ([]OfficialClaim, error)
	// Decide records an approval or rejection. The `status = 'Pending'` guard is
	// part of the statement, so two reviewers racing on one claim cannot both win.
	// Returns ErrClaimNotPending if it has already been decided, and ErrOfficeTaken
	// if approving would give an office a second holder.
	Decide(ctx context.Context, claimID string, to ClaimStatus, reviewedBy, reason string, at time.Time) error
}
