package config

import (
	"testing"
	"time"
)

// valid returns a baseline config that passes Validate; tests mutate one field.
func valid() *Config {
	return &Config{
		DatabaseURL:         "postgres://localhost/jago",
		JWTSecret:           "a-real-secret",
		JWTTTL:              time.Hour,
		ValidityThreshold:   5,
		ResponseDeadline:    7 * 24 * time.Hour,
		BlockerReviewWindow: 72 * time.Hour,
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"baseline is valid", func(*Config) {}, false},
		{"missing database url", func(c *Config) { c.DatabaseURL = "" }, true},
		{"V must be >= 1", func(c *Config) { c.ValidityThreshold = 0 }, true},
		{"D must be positive", func(c *Config) { c.ResponseDeadline = -time.Hour }, true},
		{"R must be positive", func(c *Config) { c.BlockerReviewWindow = 0 }, true},
		{"JWT TTL must be positive", func(c *Config) { c.JWTTTL = 0 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := valid()
			tt.mutate(c)
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMalformedEnvironment(t *testing.T) {
	keys := []string{"VALIDITY_THRESHOLD", "JWT_TTL", "RESPONSE_DEADLINE", "BLOCKER_REVIEW_WINDOW"}
	for _, key := range keys {
		t.Setenv(key, "")
	}
	if err := validateEnvironment(); err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "not-a-number-or-duration")
			if err := validateEnvironment(); err == nil {
				t.Fatalf("expected malformed %s to be rejected", key)
			}
		})
	}
}

func TestAllowedOrigins(t *testing.T) {
	c := valid()
	c.AllowedOrigins = []string{"https://app.vercel.app"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"*", "https://*.vercel.app", "https://app.vercel.app/path", "https://user:pass@app.vercel.app"} {
		c.AllowedOrigins = []string{origin}
		if c.Validate() == nil {
			t.Fatalf("accepted unsafe origin %s", origin)
		}
	}
}

// The dev JWT placeholder is fine locally but must be rejected in production.
func TestConfigValidate_ProductionSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	c := valid()
	c.JWTSecret = insecureJWTSecret
	if err := c.Validate(); err == nil {
		t.Fatal("expected production run with the dev JWT secret to be rejected")
	}
	c.JWTSecret = "a-real-production-secret"
	if err := c.Validate(); err != nil {
		t.Fatalf("a real secret in production should pass, got %v", err)
	}
}
