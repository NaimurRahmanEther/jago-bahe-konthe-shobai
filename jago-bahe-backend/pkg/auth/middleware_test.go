package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

func TestRoleGates(t *testing.T) {
	m := auth.NewManager("test-secret", time.Hour)
	residentTok, _ := m.Issue("u1", auth.RoleResident, "")
	officialTok, _ := m.Issue("u2", auth.RoleOfficial, "off-1")
	adminTok, _ := m.Issue("u3", auth.RoleAdmin, "off-2")
	superTok, _ := m.Issue("u4", auth.RoleSuperAdmin, "")

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	tests := []struct {
		name       string
		gate       httpx.Middleware
		authHeader string
		wantStatus int
	}{
		{"official route, no token", auth.RequireOfficial(), "", http.StatusUnauthorized},
		{"official route, wrong role", auth.RequireOfficial(), "Bearer " + residentTok, http.StatusForbidden},
		{"official route, right role", auth.RequireOfficial(), "Bearer " + officialTok, http.StatusOK},
		{"public route, resident ok", auth.RequirePublic(), "Bearer " + residentTok, http.StatusOK},
		{"public route, garbage token", auth.RequirePublic(), "Bearer not-a-jwt", http.StatusUnauthorized},

		// RequireAnyRole (B21) admits all four and nobody else. The anonymous case is
		// the load-bearing one: /api/me/notifications is derived from the token, so an
		// unauthenticated caller must get a 401 rather than a silently empty list that
		// reads as "you have no messages".
		{"any-role route, resident", auth.RequireAnyRole(), "Bearer " + residentTok, http.StatusOK},
		{"any-role route, official", auth.RequireAnyRole(), "Bearer " + officialTok, http.StatusOK},
		{"any-role route, admin", auth.RequireAnyRole(), "Bearer " + adminTok, http.StatusOK},
		{"any-role route, super admin", auth.RequireAnyRole(), "Bearer " + superTok, http.StatusOK},
		{"any-role route, anonymous", auth.RequireAnyRole(), "", http.StatusUnauthorized},

		// Roles stay FLAT: the super admin is a peer of the admin, not a rank above.
		{"admin route rejects super admin", auth.RequireAdmin(), "Bearer " + superTok, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := httpx.Chain(ok, auth.Authenticate(m), tt.gate)
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
