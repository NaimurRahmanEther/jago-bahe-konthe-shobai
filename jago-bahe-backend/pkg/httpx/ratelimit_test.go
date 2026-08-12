package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func do(h http.Handler, method, path, ip string) int {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = ip + ":12345"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestRateLimit_AuthPathIsCapped(t *testing.T) {
	h := RateLimit(3, 100, time.Minute)(okHandler())
	for i := 0; i < 3; i++ {
		if code := do(h, http.MethodPost, "/api/auth/login", "1.1.1.1"); code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i, code)
		}
	}
	if code := do(h, http.MethodPost, "/api/auth/login", "1.1.1.1"); code != http.StatusTooManyRequests {
		t.Fatalf("4th auth request: want 429, got %d", code)
	}
	// A different IP has its own budget.
	if code := do(h, http.MethodPost, "/api/auth/login", "2.2.2.2"); code != http.StatusOK {
		t.Fatalf("other IP: want 200, got %d", code)
	}
}

func TestRateLimit_WritesCappedReadsFree(t *testing.T) {
	h := RateLimit(100, 2, time.Minute)(okHandler())
	// Writes share the write budget.
	do(h, http.MethodPost, "/api/problems", "3.3.3.3")
	do(h, http.MethodPost, "/api/problems", "3.3.3.3")
	if code := do(h, http.MethodPost, "/api/problems", "3.3.3.3"); code != http.StatusTooManyRequests {
		t.Fatalf("3rd write: want 429, got %d", code)
	}
	// Reads are never limited, even past the write budget.
	for i := 0; i < 10; i++ {
		if code := do(h, http.MethodGet, "/api/problems", "3.3.3.3"); code != http.StatusOK {
			t.Fatalf("GET %d: want 200, got %d", i, code)
		}
	}
}

func TestRateLimit_Refill(t *testing.T) {
	l := newLimiter(2, time.Minute)
	base := time.Now()
	if !l.allow("k", base) || !l.allow("k", base) {
		t.Fatal("first two should pass")
	}
	if l.allow("k", base) {
		t.Fatal("third at the same instant should be denied")
	}
	// Half a window later, one token (capacity*0.5 = 1) has refilled.
	if !l.allow("k", base.Add(30*time.Second)) {
		t.Fatal("a token should have refilled after half the window")
	}
}

func TestRateLimit_ZeroDisables(t *testing.T) {
	l := newLimiter(0, time.Minute)
	now := time.Now()
	for i := 0; i < 1000; i++ {
		if !l.allow("k", now) {
			t.Fatal("a zero-capacity limiter should never deny")
		}
	}
}
