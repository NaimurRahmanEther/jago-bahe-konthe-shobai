// Package domain (identity) models residents, officials, and the accounts that
// authenticate all three roles. Pure logic — no database, HTTP, JWT, or bcrypt.
package domain

// Role is one of the three user types on the platform.
type Role string

const (
	RoleResident Role = "resident"
	RoleOfficial Role = "official"
	RoleAdmin    Role = "admin"

	// RoleSuperAdmin is the seat-level moderator: they verify above-union official
	// claims and can see every admin decision across the seat.
	//
	// It is deliberately NOT a superset of RoleAdmin — a super admin must fail
	// RequireAdmin and cannot screen a report, assign a problem, or vote. Their
	// power is sight, not override. An account able to overrule every union admin
	// would sit above every check this design relies on with nothing above it,
	// which is the concentration Concept §10 refuses ("no single moderator
	// deciding alone"). When a union admin abuses their position the remedy is
	// removing them in public, not silently reversing them.
	RoleSuperAdmin Role = "super_admin"
)

// Valid reports whether r is a known role.
func (r Role) Valid() bool {
	switch r {
	case RoleResident, RoleOfficial, RoleAdmin, RoleSuperAdmin:
		return true
	}
	return false
}
