// Package config loads the typed application configuration from the environment.
// Every tunable rule value (V, Q, W, D, R, A) and every secret lives here —
// nothing in the domain or services may hardcode a rule value (CLAUDE.md A.4
// rule 7).
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"jago-bahe-backend/pkg/dotenv"
)

// Config is the fully-parsed, typed configuration for both the api and worker.
type Config struct {
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

	// Rate limiting (B8) — per-client-IP request budgets, refilled each window.
	// Reads are unlimited; auth and write routes are capped to blunt brute-force
	// and abuse. Zero disables a limiter.
	AuthRateLimit   int           // requests per window allowed on /api/auth/*
	WriteRateLimit  int           // requests per window allowed on write (non-GET) routes
	RateLimitWindow time.Duration // the refill window for both budgets
}

// insecureJWTSecret is the dev placeholder; refusing it in production keeps an
// unset secret from silently shipping.
const insecureJWTSecret = "dev-insecure-secret-change-me"

// Load reads the environment into a Config, applying dev defaults for optional
// values. It returns an error only for values that cannot sensibly default.
func Load() (*Config, error) {
	// Pick up a local .env for no-Docker runs; real env vars still win.
	dotenv.Load("")

	cfg := &Config{
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           getEnv("JWT_SECRET", insecureJWTSecret),
		JWTTTL:              getEnvDuration("JWT_TTL", 24*time.Hour),
		ValidityThreshold:   getEnvInt("VALIDITY_THRESHOLD", 5),
		ResponseDeadline:    getEnvDuration("RESPONSE_DEADLINE", 7*24*time.Hour),
		BlockerReviewWindow: getEnvDuration("BLOCKER_REVIEW_WINDOW", 72*time.Hour),
		AuthRateLimit:       getEnvInt("AUTH_RATE_LIMIT", 20),
		WriteRateLimit:      getEnvInt("WRITE_RATE_LIMIT", 60),
		RateLimitWindow:     getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
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
	case c.RateLimitWindow <= 0:
		return fmt.Errorf("config: RATE_LIMIT_WINDOW must be > 0, got %s", c.RateLimitWindow)
	case os.Getenv("APP_ENV") == "production" && c.JWTSecret == insecureJWTSecret:
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

func getEnvFloat(key string, fallback float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
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
