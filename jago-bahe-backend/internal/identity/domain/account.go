package domain

import (
	"time"

	"jago-bahe-backend/internal/shared/domain/valueobject"
	"jago-bahe-backend/pkg/idgen"
)

// Account is the authentication identity for a person. Residents self-register
// (role resident, initially unverified, carrying nid + union). Officials and
// admins are seeded accounts that link to a directory Official via OfficialID.
type Account struct {
	ID           string
	Name         string
	Phone        valueobject.PhoneNumber
	PasswordHash string
	Role         Role
	NID          string             // residents only
	UnionID      valueobject.AreaID // residents only; their home union
	Verified     bool               // residents only; gates validation/suggestion later
	OfficialID   string             // officials/admins only; links to the directory
	CreatedAt    time.Time
}

// NewResident builds an unverified resident account from an already-hashed
// password. Verification is a separate step (an admin/backend confirms them).
func NewResident(name string, phone valueobject.PhoneNumber, passwordHash, nid string, unionID valueobject.AreaID) *Account {
	return &Account{
		ID:           idgen.New("acct"),
		Name:         name,
		Phone:        phone,
		PasswordHash: passwordHash,
		Role:         RoleResident,
		NID:          nid,
		UnionID:      unionID,
		Verified:     false,
		CreatedAt:    time.Now().UTC(),
	}
}

// NewUnverifiedOfficial builds an official's account from an already-hashed
// password. It carries no OfficialID: the account is not bound to any office
// until their claim is approved, and every official-scoped route reads the office
// off that binding. So a person who registers claiming to be the MP holds an
// account that can act as nobody until a human confirms them.
func NewUnverifiedOfficial(name string, phone valueobject.PhoneNumber, passwordHash, nid string) *Account {
	return &Account{
		ID:           idgen.New("acct"),
		Name:         name,
		Phone:        phone,
		PasswordHash: passwordHash,
		Role:         RoleOfficial,
		NID:          nid,
		Verified:     false,
		CreatedAt:    time.Now().UTC(),
	}
}

// Verify marks a resident account as verified.
func (a *Account) Verify() { a.Verified = true }
