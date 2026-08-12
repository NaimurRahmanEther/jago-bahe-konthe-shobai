// Package dotenv is a tiny, dependency-free loader for a local .env file so that
// `go run ./cmd/api` picks up configuration without Docker injecting env vars.
// Values already present in the real environment always win.
package dotenv

import (
	"bufio"
	"os"
	"strings"
)

// Load reads KEY=VALUE lines from the given file (default ".env" when empty) and
// sets any that are not already in the environment. A missing file is not an error.
func Load(path string) {
	if path == "" {
		path = ".env"
	}
	f, err := os.Open(path)
	if err != nil {
		return // no .env is fine
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
