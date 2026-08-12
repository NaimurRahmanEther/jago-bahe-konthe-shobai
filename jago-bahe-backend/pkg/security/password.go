// Package security provides password hashing. It implements the identity
// domain's Hasher port with bcrypt, keeping the crypto dependency out of the
// domain layer.
package security

import "golang.org/x/crypto/bcrypt"

// BcryptHasher hashes and verifies passwords with bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a hasher at the default cost.
func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

// Hash returns the bcrypt hash of the plaintext password.
func (h *BcryptHasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare returns nil when plain matches hash, or an error otherwise.
func (h *BcryptHasher) Compare(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
