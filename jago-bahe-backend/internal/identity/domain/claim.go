package domain

import (
	"errors"
	"time"

	"jago-bahe-backend/internal/shared/domain/valueobject"
	"jago-bahe-backend/pkg/idgen"
)

// Claim sentinel errors.
var (
	ErrClaimNotFound      = errors.New("claim not found")
	ErrClaimNotPending    = errors.New("this claim has already been decided")
	ErrInvalidClaimStatus = errors.New("invalid claim status")
	ErrOfficeTaken        = errors.New("this office is already claimed")
	ErrAlreadyClaimed     = errors.New("this account already has a claim")
	ErrNotClaimApprover   = errors.New("you cannot decide this claim")
)

// ClaimStatus is where an official's claim to a directory entry stands.
type ClaimStatus string

const (
	ClaimPending  ClaimStatus = "Pending"
	ClaimApproved ClaimStatus = "Approved"
	ClaimRejected ClaimStatus = "Rejected"
)

// Valid reports whether s is a known claim status.
func (s ClaimStatus) Valid() bool {
	switch s {
	case ClaimPending, ClaimApproved, ClaimRejected:
		return true
	}
	return false
}

// ClaimRoute names who may decide a claim.
type ClaimRoute string

const (
	ClaimRouteUnionAdmin ClaimRoute = "union_admin"
	ClaimRouteSuperAdmin ClaimRoute = "super_admin"
)

// OfficialClaim is a person's assertion that they hold a directory office.
//
// The directory itself is a record of real election results, so a claim never
// creates an office — it links an account to one that already exists. What the
// approver answers is narrow and checkable: is this phone and identity really
// that person? Until it is approved the account holds the official role but no
// OfficialID, and so can do nothing as that official.
type OfficialClaim struct {
	ID         string
	AccountID  string
	OfficialID string
	Status     ClaimStatus
	ReviewedBy string
	ReviewedAt *time.Time
	Reason     string // required on rejection; shown to the claimant
	CreatedAt  time.Time
}

// NewOfficialClaim constructs a pending claim.
func NewOfficialClaim(accountID, officialID string) *OfficialClaim {
	return &OfficialClaim{
		ID:         idgen.New("claim"),
		AccountID:  accountID,
		OfficialID: officialID,
		Status:     ClaimPending,
		CreatedAt:  time.Now().UTC(),
	}
}

// RouteClaim decides who may verify a claim to an office at this tier.
//
// It reuses Tier.IsUnionLevel() — the same split B4 routes assignment on — so the
// platform has one answer to "is this a union matter or a seat matter". A union
// admin knows their own ward members and chairman personally, which is exactly
// what makes them the right person to confirm an identity at that tier. Nobody at
// union level has standing over an MP, so those go to the super admin.
func RouteClaim(tier valueobject.Tier) ClaimRoute {
	if tier.IsUnionLevel() {
		return ClaimRouteUnionAdmin
	}
	return ClaimRouteSuperAdmin
}

// DecideClaim reports whether moving a claim from current to next is legal. A
// claim is decided exactly once: re-deciding an approved one would hand an office
// from one person to another with no trace of the transfer.
func DecideClaim(current, next ClaimStatus) error {
	if next != ClaimApproved && next != ClaimRejected {
		return ErrInvalidClaimStatus
	}
	if current != ClaimPending {
		return ErrClaimNotPending
	}
	return nil
}
