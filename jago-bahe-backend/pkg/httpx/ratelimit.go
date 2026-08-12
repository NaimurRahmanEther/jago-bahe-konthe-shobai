package httpx

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// limiter is a per-key token bucket: each key (a client IP) gets `capacity`
// tokens that refill continuously at capacity/window and are spent one per
// request. It is safe for concurrent use and self-prunes idle full buckets so
// memory stays bounded without a background goroutine.
type limiter struct {
	capacity float64
	window   time.Duration
	mu       sync.Mutex
	buckets  map[string]*bucket
	nextSwp  time.Time
}

type bucket struct {
	tokens float64
	seen   time.Time
}

func newLimiter(capacity int, window time.Duration) *limiter {
	return &limiter{
		capacity: float64(capacity),
		window:   window,
		buckets:  make(map[string]*bucket),
	}
}

// allow spends a token for key, refilling first based on elapsed time. A
// non-positive capacity disables the limiter (always allows).
func (l *limiter) allow(key string, now time.Time) bool {
	if l.capacity <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)

	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.capacity}
		l.buckets[key] = b
	}
	refill := (now.Sub(b.seen).Seconds() / l.window.Seconds()) * l.capacity
	b.tokens = min(l.capacity, b.tokens+refill)
	b.seen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweep drops buckets that have sat idle (fully refilled) for two windows, so the
// map does not grow without bound. Runs at most once per window.
func (l *limiter) sweep(now time.Time) {
	if now.Before(l.nextSwp) {
		return
	}
	l.nextSwp = now.Add(l.window)
	for k, b := range l.buckets {
		if now.Sub(b.seen) > 2*l.window {
			delete(l.buckets, k)
		}
	}
}

// RateLimit caps requests per client IP: auth routes (/api/auth/*) use the
// stricter authLimit to blunt credential brute-forcing, and every write (non-GET)
// route uses writeLimit; reads are never limited. A zero limit disables that
// budget. Over budget → 429 with a Retry-After hint. IP is taken from
// RemoteAddr, not spoofable client headers.
func RateLimit(authLimit, writeLimit int, window time.Duration) Middleware {
	auth := newLimiter(authLimit, window)
	write := newLimiter(writeLimit, window)
	retryAfter := strconv.Itoa(int(window.Seconds()))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var lim *limiter
			switch {
			case isAuthPath(r.URL.Path):
				lim = auth
			case r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions:
				lim = write
			}
			if lim != nil && !lim.allow(clientIP(r), time.Now()) {
				w.Header().Set("Retry-After", retryAfter)
				Error(w, http.StatusTooManyRequests, "rate_limited", "Too many requests. Please slow down and try again.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isAuthPath(path string) bool {
	const prefix = "/api/auth/"
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

// clientIP returns the request's source IP (host part of RemoteAddr), the
// identity the rate limiter keys on. RemoteAddr is set by the server from the
// socket, so it cannot be forged by the client the way a header could.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
