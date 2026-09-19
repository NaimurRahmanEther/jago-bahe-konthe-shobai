package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	h := CORS([]string{"https://app.vercel.app"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for _, origin := range []string{"https://app.vercel.app", "https://evil.example", "https://app.vercel.app.evil.example"} {
		for _, method := range []string{"OPTIONS", "GET"} {
			r := httptest.NewRequest(method, "/api/problems", nil)
			r.Header.Set("Origin", origin)
			if method == "OPTIONS" {
				r.Header.Set("Access-Control-Request-Method", "POST")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			allowed := origin == "https://app.vercel.app"
			if (w.Header().Get("Access-Control-Allow-Origin") == origin) != allowed {
				t.Fatal("incorrect origin permission")
			}
			if method == "OPTIONS" {
				want := 403
				if allowed {
					want = 204
				}
				if w.Code != want {
					t.Fatalf("preflight got %d want %d", w.Code, want)
				}
			}
			if w.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatal("cookies must not be enabled")
			}
		}
	}
}
