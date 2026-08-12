package auth

import (
	"context"
	"net/http"
	"strings"

	"jago-bahe-backend/pkg/httpx"
)

type ctxKey string

const claimsKey ctxKey = "auth_claims"

// Authenticate is best-effort: if a valid Bearer token is present it puts the
// caller's Claims on the context; otherwise the request proceeds anonymously.
// The RequireX gates enforce presence and role.
func Authenticate(m *Manager) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := bearer(r); token != "" {
				if claims, err := m.Parse(token); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), claimsKey, claims))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFrom returns the authenticated caller's claims, and whether present.
func ClaimsFrom(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}

// RequireRole builds a gate that admits only the given roles.
func RequireRole(roles ...string) httpx.Middleware {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFrom(r.Context())
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication required.")
				return
			}
			if !allowed[claims.Role] {
				httpx.Error(w, http.StatusForbidden, "forbidden", "You do not have access to this resource.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePublic admits residents (the public). RequireOfficial, RequireAdmin and
// RequireSuperAdmin admit their respective roles.
//
// These gates are flat and exact: no role implies another. In particular
// RequireAdmin does NOT admit a super admin — the super admin oversees union
// admins rather than outranking them, so they must not be able to screen, assign,
// or vote. RequireClaimReviewer is the one place both admin kinds meet, and even
// there the use case decides which of them may act on a given claim.
func RequirePublic() httpx.Middleware     { return RequireRole(RoleResident) }
func RequireOfficial() httpx.Middleware   { return RequireRole(RoleOfficial) }
func RequireAdmin() httpx.Middleware      { return RequireRole(RoleAdmin) }
func RequireSuperAdmin() httpx.Middleware { return RequireRole(RoleSuperAdmin) }

// RequireClaimReviewer admits both admin kinds; which of them may decide a
// particular claim is routed by tier in the use case, not by the URL.
func RequireClaimReviewer() httpx.Middleware { return RequireRole(RoleAdmin, RoleSuperAdmin) }

// RequireAnyRole admits every signed-in role and refuses anonymous callers. It is
// the gate for surfaces that are about the CALLER rather than about their office;
// /api/me/notifications is the first, since a resident, an official, an admin and
// the super admin each have their own.
//
// Named for the family it belongs to (RequireRole -> RequireAnyRole) rather than
// for its effect. The alternative, RequireSignedIn, reads better in isolation and
// is worse in practice: it invites an implementation that merely checks claims are
// present, which would admit any role added later, silently. This one ENUMERATES,
// exactly as the flat gates above it do, so a fifth role has to be added here on
// purpose. RequireClaimReviewer is the precedent for a multi-role gate.
func RequireAnyRole() httpx.Middleware {
	return RequireRole(RoleResident, RoleOfficial, RoleAdmin, RoleSuperAdmin)
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
