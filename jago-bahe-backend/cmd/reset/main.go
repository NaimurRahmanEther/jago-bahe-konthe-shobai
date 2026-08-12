// Command reset clears every problem from the database so a manual test round
// starts from an empty board, without touching accounts, geography or the
// officials directory.
//
//	go run ./cmd/reset            # report what would be deleted, change nothing
//	go run ./cmd/reset confirm    # delete
//
// It runs scripts/reset_problems.sql, which documents what cascades and why the
// audit entries need deleting by hand. This mirrors cmd/migrate: a small Go
// runner so the no-Docker path needs no psql on PATH.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"jago-bahe-backend/config"
	"jago-bahe-backend/pkg/postgres"
)

func main() {
	file := flag.String("file", "scripts/reset_problems.sql", "SQL script to run")
	flag.Parse()

	confirmed := false
	if args := flag.Args(); len(args) > 0 {
		if args[0] != "confirm" {
			fail(fmt.Errorf("unknown argument %q (use confirm)", args[0]))
		}
		confirmed = true
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

	problems, entries, err := counts(ctx, pool)
	if err != nil {
		fail(err)
	}

	if !confirmed {
		fmt.Printf("reset: %d problems and %d problem audit entries would be deleted.\n", problems, entries)
		fmt.Println("reset: nothing changed. Re-run with `confirm` to delete.")
		return
	}

	script, err := os.ReadFile(*file)
	if err != nil {
		fail(err)
	}
	if err := execScript(ctx, pool, string(script)); err != nil {
		fail(fmt.Errorf("run %s: %w", *file, err))
	}

	// Recount rather than reporting the before-figures as if they were deletions.
	// Not every script here deletes: scripts/backdate_demo.sql only shifts
	// timestamps, and printing "deleted 23 problems" after it ran was alarming and
	// false. The delta is the honest number for both kinds.
	after, afterEntries, err := counts(ctx, pool)
	if err != nil {
		fail(err)
	}
	if problems == after && entries == afterEntries {
		fmt.Printf("reset: ran %s; no rows removed (%d problems, %d problem audit entries remain).\n",
			*file, after, afterEntries)
		return
	}
	fmt.Printf("reset: deleted %d problems and %d problem audit entries.\n", problems-after, entries-afterEntries)
	fmt.Println("reset: accounts, areas and the officials directory are untouched.")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "reset: "+err.Error())
	os.Exit(1)
}

func counts(ctx context.Context, pool *pgxpool.Pool) (problems, entries int, err error) {
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM problems`).Scan(&problems); err != nil {
		return 0, 0, err
	}
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_entries WHERE target_type = 'problem'`).Scan(&entries)
	return problems, entries, err
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
