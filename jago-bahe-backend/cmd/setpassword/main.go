// Command setpassword rotates one existing account password without changing its
// role, claims, or history. Read the password from stdin, never command arguments.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"jago-bahe-backend/config"
	"jago-bahe-backend/pkg/postgres"
	"jago-bahe-backend/pkg/security"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: setpassword ACCOUNT_ID < password-file (12-72 bytes)")
	}
	input, err := io.ReadAll(io.LimitReader(os.Stdin, 75))
	if err != nil {
		return err
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(input), "\n"), "\r")
	if len(password) < 12 || len(password) > 72 || strings.TrimSpace(password) == "" || strings.ContainsAny(password, "\r\n") {
		return fmt.Errorf("password must be a single line of 12-72 bytes")
	}
	hash, err := security.NewProductionBcryptHasher().Hash(password)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	result, err := pool.Exec(ctx, "UPDATE accounts SET password_hash = $1 WHERE id = $2", hash, os.Args[1])
	if err != nil {
		return fmt.Errorf("password update failed")
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("account not found")
	}
	fmt.Println("Password updated. Existing JWTs require JWT_SECRET rotation to revoke.")
	return nil
}
