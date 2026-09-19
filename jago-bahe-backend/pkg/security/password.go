// Package security provides password hashing. It implements the identity
// domain's Hasher port with bcrypt, keeping the crypto dependency out of the
// domain layer.
package security

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher hashes and verifies passwords with bcrypt.
type BcryptHasher struct {
	cost       int
	production bool
}

// NewBcryptHasher returns a hasher at the default cost.
func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

func NewProductionBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost, production: true}
}

// These publicly documented migration passwords must never authenticate in
// production, even if somebody re-hashes one with a different salt.
func IsDevelopmentPassword(plain string) bool {
	return plain == "admin123" || plain == "resident123" || plain == "official123"
}

// Hash returns the bcrypt hash of the plaintext password.
func (h *BcryptHasher) Hash(plain string) (string, error) {
	if h.production && IsDevelopmentPassword(plain) {
		return "", errors.New("development passwords are disabled in production")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare returns nil when plain matches hash, or an error otherwise.
func (h *BcryptHasher) Compare(hash, plain string) error {
	if h.production && IsDevelopmentPassword(plain) {
		return bcrypt.ErrMismatchedHashAndPassword
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
