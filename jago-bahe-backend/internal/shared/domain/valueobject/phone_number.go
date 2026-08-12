package valueobject

import (
	"errors"
	"regexp"
	"strings"
)

// ErrInvalidPhone is returned when a phone number is not a valid Bangladeshi
// 11-digit mobile number (01XXXXXXXXX).
var ErrInvalidPhone = errors.New("invalid phone number")

var phonePattern = regexp.MustCompile(`^01[0-9]{9}$`)

// bengaliZero is U+09E6 ('০'), the first of the ten Bengali digits.
const bengaliZero = '০'

// PhoneNumber is a validated Bangladeshi mobile number, always in the canonical
// 01XXXXXXXXX form regardless of how it was written.
type PhoneNumber string

// normalize reduces the many ways one number gets written down to the single
// form that is stored and compared. It is deliberately narrow: it only removes
// what is decoration (separators, the country code, Bengali digits) and never
// repairs a number that is genuinely wrong.
//
// Without this, "+880 1712-345678" and "01712345678" are different accounts to
// the database and the second is unreachable — which is exactly what browser
// autofill produces on a login form after the number was typed plainly at
// registration.
func normalize(raw string) string {
	s := strings.Map(func(r rune) rune {
		switch {
		case r >= bengaliZero && r <= bengaliZero+9:
			return '0' + (r - bengaliZero)
		case r == ' ' || r == '\t' || r == '-' || r == '(' || r == ')' || r == '.':
			return -1 // drop
		default:
			return r
		}
	}, strings.TrimSpace(raw))

	s = strings.TrimPrefix(s, "+")

	// Scoped to the exact country-code length so a foreign number is never
	// salvaged into a local one: a canonical number is 11 characters, so this
	// can only ever match a +880/880-prefixed one.
	if len(s) == 13 && strings.HasPrefix(s, "880") {
		s = "0" + s[3:]
	}
	return s
}

// NewPhoneNumber normalizes, validates and constructs a PhoneNumber.
func NewPhoneNumber(raw string) (PhoneNumber, error) {
	s := normalize(raw)
	if !phonePattern.MatchString(s) {
		return "", ErrInvalidPhone
	}
	return PhoneNumber(s), nil
}

func (p PhoneNumber) String() string { return string(p) }
