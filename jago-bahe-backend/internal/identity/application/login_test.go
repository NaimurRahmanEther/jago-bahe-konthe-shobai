package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/identity/application"
	"jago-bahe-backend/internal/identity/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// fakeAccounts serves one account by phone and records the phone it was asked
// for, so a test can assert on the LOOKUP KEY and not merely on the rows: a
// result-only assertion still passes when login hands the repository a phone the
// user never typed.
type fakeAccounts struct {
	account *domain.Account
	asked   valueobject.PhoneNumber
}

func (f *fakeAccounts) GetByPhone(_ context.Context, phone valueobject.PhoneNumber) (*domain.Account, error) {
	f.asked = phone
	if f.account == nil || f.account.Phone != phone {
		return nil, domain.ErrAccountNotFound
	}
	return f.account, nil
}

// Unused by Login; present to satisfy the port.
func (f *fakeAccounts) Create(context.Context, *domain.Account) error { return nil }
func (f *fakeAccounts) GetByID(context.Context, string) (*domain.Account, error) {
	return nil, domain.ErrAccountNotFound
}
func (f *fakeAccounts) SetVerified(context.Context, string, bool) error { return nil }
func (f *fakeAccounts) ListByRole(context.Context, domain.Role) ([]domain.Account, error) {
	return nil, nil
}
func (f *fakeAccounts) ListUnverifiedResidents(context.Context, valueobject.AreaID) ([]domain.Account, error) {
	return nil, nil
}
func (f *fakeAccounts) SetOfficialID(context.Context, string, string) error { return nil }
func (f *fakeAccounts) NamesByIDs(context.Context, []string) (map[string]string, error) {
	return nil, nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(p string) (string, error) { return "hash-" + p, nil }
func (fakeHasher) Compare(hash, plain string) error {
	if hash == "hash-"+plain {
		return nil
	}
	return errors.New("mismatch")
}

type fakeTokens struct{}

func (fakeTokens) Issue(userID, role, officialID string) (string, error) {
	return "token-" + userID, nil
}

func newAccount(t *testing.T) *domain.Account {
	t.Helper()
	phone, err := valueobject.NewPhoneNumber("01710000001")
	if err != nil {
		t.Fatalf("seed phone: %v", err)
	}
	return domain.NewResident("রহিমা", phone, "hash-secret", "nid", "union-1")
}

func newLogin(acc *domain.Account) (*application.Login, *fakeAccounts) {
	repo := &fakeAccounts{account: acc}
	return application.NewLogin(repo, domain.NewService(), fakeHasher{}, fakeTokens{}), repo
}

// The bug this fixes: the number was typed plainly at registration and comes
// back reformatted by browser autofill at the next login. Both must reach the
// same row.
func TestLoginAcceptsEverySpellingOfTheSameNumber(t *testing.T) {
	for _, raw := range []string{
		"01710000001",
		"+8801710000001",
		"+880 1710-000001",
		"  01710000001 ",
		"০১৭১০০০০০০১",
	} {
		t.Run(raw, func(t *testing.T) {
			uc, repo := newLogin(newAccount(t))

			res, err := uc.Execute(context.Background(), application.LoginInput{Phone: raw, Password: "secret"})
			if err != nil {
				t.Fatalf("Execute(%q) error = %v, want nil", raw, err)
			}
			if res.Token == "" {
				t.Fatal("Execute() issued no token")
			}
			if repo.asked.String() != "01710000001" {
				t.Fatalf("looked up %q, want the canonical 01710000001", repo.asked.String())
			}
		})
	}
}

func TestLoginErrors(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		password string
		wantErr  error
	}{
		// A malformed phone is not a credential failure. It cannot be anyone's
		// account, so naming it leaks nothing — and calling it "wrong phone or
		// password" tells the user to doubt a password that was correct.
		{"malformed phone", "not-a-phone", "secret", valueobject.ErrInvalidPhone},
		// Unknown account and wrong password stay indistinguishable: that pairing
		// is what stops login being an oracle over which phones are registered.
		{"unknown account", "01799999999", "secret", domain.ErrInvalidCredentials},
		{"wrong password", "01710000001", "nope", domain.ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, _ := newLogin(newAccount(t))

			_, err := uc.Execute(context.Background(), application.LoginInput{Phone: tt.phone, Password: tt.password})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
