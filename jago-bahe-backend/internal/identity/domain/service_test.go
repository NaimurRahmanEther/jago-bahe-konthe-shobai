package domain_test

import (
	"errors"
	"testing"

	"jago-bahe-backend/internal/identity/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// fakeHasher treats "hash-"+plain as the matching hash.
type fakeHasher struct{}

func (fakeHasher) Hash(p string) (string, error) { return "hash-" + p, nil }
func (fakeHasher) Compare(hash, plain string) error {
	if hash == "hash-"+plain {
		return nil
	}
	return errors.New("mismatch")
}

func TestAuthenticate(t *testing.T) {
	phone, _ := valueobject.NewPhoneNumber("01710000001")
	acc := domain.NewResident("রহিমা", phone, "hash-secret", "nid", "union-1")
	svc := domain.NewService()

	tests := []struct {
		name    string
		acc     *domain.Account
		pw      string
		wantErr error
	}{
		{"correct password", acc, "secret", nil},
		{"wrong password", acc, "nope", domain.ErrInvalidCredentials},
		{"nil account", nil, "secret", domain.ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Authenticate(tt.acc, tt.pw, fakeHasher{})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Authenticate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
