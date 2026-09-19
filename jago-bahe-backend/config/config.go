// Package config loads the typed application configuration from the environment.
// Every tunable rule value (V, Q, W, D, R, A) and every secret lives here —
// nothing in the domain or services may hardcode a rule value (CLAUDE.md A.4
// rule 7).
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"jago-bahe-backend/pkg/dotenv"
)

// Config is the fully-parsed, typed configuration for both the api and worker.
type Config struct {
	Production bool

	AllowedOrigins []string
	// HTTP
	HTTPAddr string

	// Database
	DatabaseURL string

	// Auth (used from B1 onward; parsed here so the shape is stable)
	JWTSecret string
	JWTTTL    time.Duration

	// Tunable rule values — the pilot dials (CLAUDE.md A.3). Placeholders in dev;
	// real values come from .env during the one-union pilot.
	//
	// There were five. Q (ADMIN_QUORUM) and W (VOTE_WINDOW) went with the binding
	// admin vote in B20: above-union forwarding is now the super admin's decision
	// advised by the union admins, and advice that settles nothing has no quorum to
	// meet and no window to close. Do not reintroduce them without reinstating the
	// vote itself — a dial that turns nothing is worse than no dial (A.3.8).
	ValidityThreshold   int           // V — distinct verified area residents to validate
	ResponseDeadline    time.Duration // D — first-response deadline; drives escalation
	BlockerReviewWindow time.Duration // R — capped clock-pause while a blocker is judged

}

// insecureJWTSecret is the dev placeholder; refusing it in production keeps an
// unset secret from silently shipping.
const insecureJWTSecret = "dev-insecure-secret-change-me"

// Load reads the environment into a Config, applying dev defaults for optional
// values. It returns an error only for values that cannot sensibly default.
func Load() (*Config, error) {
	// Pick up a local .env for no-Docker runs; real env vars still win.
	dotenv.Load("")
	if err := validateEnvironment(); err != nil {
		return nil, err
	}

	cfg := &Config{
		Production:          os.Getenv("APP_ENV") == "production",
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           getEnv("JWT_SECRET", insecureJWTSecret),
		JWTTTL:              getEnvDuration("JWT_TTL", 24*time.Hour),
		ValidityThreshold:   getEnvInt("VALIDITY_THRESHOLD", 5),
		ResponseDeadline:    getEnvDuration("RESPONSE_DEADLINE", 7*24*time.Hour),
		BlockerReviewWindow: getEnvDuration("BLOCKER_REVIEW_WINDOW", 72*time.Hour),
	}
	for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, origin)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate rejects a configuration that cannot run correctly, so a misconfigured
// deploy fails fast on boot rather than misbehaving under load. The tunables V,
// D, R must be in-range, and a production run (APP_ENV=production) must set a
// real JWT secret rather than ship the dev placeholder.
func (c *Config) Validate() error {
	for _, origin := range c.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") || strings.Contains(origin, "*") {
			return fmt.Errorf("config: ALLOWED_ORIGINS must contain exact HTTP(S) origins without paths or wildcards")
		}
	}
	switch {
	case c.DatabaseURL == "":
		return fmt.Errorf("config: DATABASE_URL is required")
	case c.ValidityThreshold < 1:
		return fmt.Errorf("config: VALIDITY_THRESHOLD (V) must be >= 1, got %d", c.ValidityThreshold)
	case c.ResponseDeadline <= 0:
		return fmt.Errorf("config: RESPONSE_DEADLINE (D) must be > 0, got %s", c.ResponseDeadline)
	case c.BlockerReviewWindow <= 0:
		return fmt.Errorf("config: BLOCKER_REVIEW_WINDOW (R) must be > 0, got %s", c.BlockerReviewWindow)
	case c.JWTTTL <= 0:
		return fmt.Errorf("config: JWT_TTL must be > 0, got %s", c.JWTTTL)
	case os.Getenv("APP_ENV") == "production" && (strings.TrimSpace(c.JWTSecret) == "" || c.JWTSecret == insecureJWTSecret):
		return fmt.Errorf("config: JWT_SECRET must be set to a real secret in production")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

// Optional settings may be absent, but explicitly malformed values must not
// silently change deployment behavior by falling back to development defaults.
func validateEnvironment() error {
	for _, key := range []string{"VALIDITY_THRESHOLD"} {
		if value := os.Getenv(key); value != "" {
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("config: %s must be an integer", key)
			}
		}
	}
	for _, key := range []string{"JWT_TTL", "RESPONSE_DEADLINE", "BLOCKER_REVIEW_WINDOW"} {
		if value := os.Getenv(key); value != "" {
			if _, err := time.ParseDuration(value); err != nil {
				return fmt.Errorf("config: %s must be a duration such as 1m or 24h", key)
			}
		}
	}
	return nil
}
