// Package idgen produces opaque, collision-resistant string identifiers used as
// aggregate ids across contexts. Not RFC-4122, but unique enough for this scale
// and dependency-free.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a 32-character hex identifier, optionally prefixed (e.g. "audit").
func New(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	if prefix == "" {
		return id
	}
	return prefix + "-" + id
}
