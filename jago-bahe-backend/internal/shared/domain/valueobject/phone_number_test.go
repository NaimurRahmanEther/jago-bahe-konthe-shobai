package valueobject_test

import (
	"errors"
	"testing"

	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// One number has many spellings. Register and login are typed by the same person
// on different days — one of them through browser autofill, which reformats — so
// every spelling below must resolve to the one canonical form that is stored.
func TestNewPhoneNumberNormalizes(t *testing.T) {
	const canonical = "01712345678"

	tests := []struct {
		name string
		raw  string
	}{
		{"already canonical", "01712345678"},
		{"country code with plus", "+8801712345678"},
		{"country code without plus", "8801712345678"},
		{"autofill formatting", "+880 1712-345678"},
		{"dashes", "01712-345678"},
		{"spaces", "017 1234 5678"},
		{"surrounding whitespace", "  01712345678 "},
		{"parenthesised country code", "(+880) 1712 345678"},
		{"dots", "01712.345678"},
		{"bengali numerals", "০১৭১২৩৪৫৬৭৮"},
		{"bengali numerals with country code", "+৮৮০ ১৭১২-৩৪৫৬৭৮"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.NewPhoneNumber(tt.raw)
			if err != nil {
				t.Fatalf("NewPhoneNumber(%q) error = %v, want nil", tt.raw, err)
			}
			if got.String() != canonical {
				t.Fatalf("NewPhoneNumber(%q) = %q, want %q", tt.raw, got.String(), canonical)
			}
		})
	}
}

func TestNewPhoneNumberRejects(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"too short", "0171234567"},
		{"too long", "017123456789"},
		{"wrong prefix", "02712345678"},
		{"letters", "abcdefghijk"},
		{"separators only", "+- ()."},
		// Not a Bangladeshi mobile number: stripping "880" is scoped to the exact
		// 13-char country-code form, so this must not be salvaged into one.
		{"foreign country code", "+14155552671"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := valueobject.NewPhoneNumber(tt.raw); !errors.Is(err, valueobject.ErrInvalidPhone) {
				t.Fatalf("NewPhoneNumber(%q) error = %v, want ErrInvalidPhone", tt.raw, err)
			}
		})
	}
}
