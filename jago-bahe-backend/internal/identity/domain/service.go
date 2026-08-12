package domain

// Hasher is the port for password hashing/verification, implemented outside the
// domain (pkg/security with bcrypt). Keeping it an interface lets the domain
// express the authentication rule without importing a crypto library.
type Hasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

// Service holds identity domain rules that span entities.
type Service struct{}

// NewService constructs the identity domain service.
func NewService() *Service { return &Service{} }

// Authenticate verifies a plaintext password against an account, returning
// ErrInvalidCredentials on any mismatch. It never distinguishes "no such
// account" from "wrong password" to callers, to avoid leaking which phones exist.
func (s *Service) Authenticate(a *Account, password string, h Hasher) error {
	if a == nil {
		return ErrInvalidCredentials
	}
	if err := h.Compare(a.PasswordHash, password); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}
