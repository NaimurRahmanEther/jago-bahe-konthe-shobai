// Package auth issues and validates the JWTs that carry a caller's identity and
// role, and provides the HTTP middleware that enforces role gates. It is generic
// (pkg-level): it defines its own role string constants rather than importing an
// internal context.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role values carried in a token. They match the identity domain's Role strings.
const (
	RoleResident = "resident"
	RoleOfficial = "official"
	RoleAdmin    = "admin"
	// RoleSuperAdmin is a peer of RoleAdmin, not a rank above it: RequireAdmin
	// must reject it. See identity/domain.RoleSuperAdmin for why.
	RoleSuperAdmin = "super_admin"
)

// ErrInvalidToken is returned when a token is missing, malformed, or expired.
var ErrInvalidToken = errors.New("invalid token")

// Claims is the decoded, trusted identity of a caller.
type Claims struct {
	UserID     string
	Role       string
	OfficialID string
}

// Manager issues and parses HS256 tokens.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// NewManager constructs a token Manager.
func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

type jwtClaims struct {
	Role       string `json:"role"`
	OfficialID string `json:"officialId,omitempty"`
	jwt.RegisteredClaims
}

// Issue signs a token for the given account.
func (m *Manager) Issue(userID, role, officialID string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		Role:       role,
		OfficialID: officialID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse validates a token string and returns its claims.
func (m *Manager) Parse(token string) (Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	c, ok := parsed.Claims.(*jwtClaims)
	if !ok || !parsed.Valid {
		return Claims{}, ErrInvalidToken
	}
	return Claims{UserID: c.Subject, Role: c.Role, OfficialID: c.OfficialID}, nil
}
