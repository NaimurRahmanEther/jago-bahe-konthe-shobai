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
		AuthRateLimit:       20,
		WriteRateLimit:      60,
		RateLimitWindow:     time.Minute,
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
		{"rate-limit window must be positive", func(c *Config) { c.RateLimitWindow = 0 }, true},
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
