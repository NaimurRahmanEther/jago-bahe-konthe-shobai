// Command migrate applies SQL migrations without any external tool or Docker.
// It reads migrations/NNNNNN_name.up.sql (or .down.sql) in order, tracks applied
// versions in a schema_migrations table, and runs each file atomically.
//
//	go run ./cmd/migrate up      # apply all pending (default)
//	go run ./cmd/migrate down    # roll back the most recent
//
// The file naming stays golang-migrate-compatible, so the Docker migrate service
// still works for anyone who prefers it.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"jago-bahe-backend/config"
	"jago-bahe-backend/pkg/postgres"
)

func main() {
	dir := flag.String("path", "migrations", "migrations directory")
	flag.Parse()

	direction := "up"
	if args := flag.Args(); len(args) > 0 {
		direction = args[0]
	}

	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		fail(err)
	}
	defer pool.Close()

	if err := ensureTable(ctx, pool); err != nil {
		fail(err)
	}

	switch direction {
	case "up":
		err = up(ctx, pool, *dir)
	case "down":
		err = down(ctx, pool, *dir)
	default:
		fail(fmt.Errorf("unknown direction %q (use up|down)", direction))
	}
	if err != nil {
		fail(err)
	}
	fmt.Println("migrate: done")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "migrate: "+err.Error())
	os.Exit(1)
}

func ensureTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	return err
}

func up(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}
	files, err := migrationFiles(dir, ".up.sql")
	if err != nil {
		return err
	}
	pending := 0
	for _, f := range files {
		if applied[f.version] {
			continue
		}
		sqlBytes, err := os.ReadFile(f.path)
		if err != nil {
			return err
		}
		script := fmt.Sprintf("BEGIN;\n%s\nINSERT INTO schema_migrations(version) VALUES ('%s');\nCOMMIT;",
			string(sqlBytes), f.version)
		if err := execScript(ctx, pool, script); err != nil {
			return fmt.Errorf("apply %s: %w", f.name, err)
		}
		fmt.Printf("migrate: applied %s\n", f.name)
		pending++
	}
	if pending == 0 {
		fmt.Println("migrate: nothing to apply")
	}
	return nil
}

func down(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}
	files, err := migrationFiles(dir, ".down.sql")
	if err != nil {
		return err
	}
	// Roll back the highest applied version.
	for i := len(files) - 1; i >= 0; i-- {
		f := files[i]
		if !applied[f.version] {
			continue
		}
		sqlBytes, err := os.ReadFile(f.path)
		if err != nil {
			return err
		}
		script := fmt.Sprintf("BEGIN;\n%s\nDELETE FROM schema_migrations WHERE version = '%s';\nCOMMIT;",
			string(sqlBytes), f.version)
		if err := execScript(ctx, pool, script); err != nil {
			return fmt.Errorf("rollback %s: %w", f.name, err)
		}
		fmt.Printf("migrate: rolled back %s\n", f.name)
		return nil
	}
	fmt.Println("migrate: nothing to roll back")
	return nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

type migration struct {
	version string
	name    string
	path    string
}

func migrationFiles(dir, suffix string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		version, _, _ := strings.Cut(e.Name(), "_")
		out = append(out, migration{
			version: version,
			name:    e.Name(),
			path:    filepath.Join(dir, e.Name()),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// execScript runs a full multi-statement SQL script via the simple protocol,
// which the extended (parameterized) path does not allow.
func execScript(ctx context.Context, pool *pgxpool.Pool, script string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	return conn.Conn().PgConn().Exec(ctx, script).Close()
}
