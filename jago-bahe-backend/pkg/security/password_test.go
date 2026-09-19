package security

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestProductionRejectsEverySeedPassword(t *testing.T) {
	files, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]bool{}
	re := regexp.MustCompile(`\$2[aby]\$[0-9]{2}\$[./A-Za-z0-9]{53}`)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, hash := range re.FindAllString(string(data), -1) {
			hashes[hash] = true
		}
	}
	if len(hashes) == 0 {
		t.Fatal("no seed hashes found")
	}
	dev, prod := NewBcryptHasher(), NewProductionBcryptHasher()
	for hash := range hashes {
		matched := false
		for _, password := range []string{"admin123", "resident123", "official123"} {
			if dev.Compare(hash, password) == nil {
				matched = true
				if prod.Compare(hash, password) == nil {
					t.Fatal("production accepted a seed password")
				}
			}
		}
		if !matched {
			t.Fatal("new seed hash needs a production credential policy review")
		}
	}
	hash, err := prod.Hash("unique-replacement-password-123!")
	if err != nil {
		t.Fatal(err)
	}
	if prod.Compare(hash, "unique-replacement-password-123!") != nil {
		t.Fatal("replacement password rejected")
	}
}
