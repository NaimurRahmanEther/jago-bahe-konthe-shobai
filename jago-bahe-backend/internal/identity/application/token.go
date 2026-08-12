// Package application holds the identity use cases: they orchestrate the domain,
// repositories, password hashing, token issuance, and audit.
package application

// TokenIssuer is the port for minting a session token for an account. It is
// implemented by pkg/auth (JWT) and injected by the composition root.
type TokenIssuer interface {
	Issue(userID, role, officialID string) (string, error)
}
