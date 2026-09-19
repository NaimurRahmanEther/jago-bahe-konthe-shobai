.PHONY: up down migrate migrate-up migrate-down reset-problems reset-demo backdate-demo test run build tidy fmt

# Bring up the full stack: postgres + migrate + api + worker.
up:
	docker compose --env-file .env -f ../docker-compose.yml up --build

# Stop the stack while preserving database data.
down:
	docker compose --env-file .env -f ../docker-compose.yml down

# Apply migrations against the running database (Docker path).
migrate:
	docker compose --env-file .env -f ../docker-compose.yml run --rm migrate

# No-Docker path: apply / roll back migrations with the built-in Go runner,
# using DATABASE_URL from .env (against a locally-installed Postgres).
migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

# Wipe every problem and everything cascading from it, for a clean manual test
# round. Accounts, geography and the officials directory are untouched.
# Run `go run ./cmd/reset` without the target for a dry count first.
reset-problems:
	go run ./cmd/reset confirm

# Clean slate before re-seeding the demo board: wipes every problem AND every
# account created at runtime, leaving only what the migrations seeded (geography,
# the officials directory, the nine union admins, the super admin, and the two
# logins information.md documents). Step 1 of the demo pipeline:
#
#   make reset-demo
#   node ../report/scripts/seed_demo.mjs
#   make backdate-demo
#   go run ./cmd/worker      # one tick, then Ctrl-C, to open the observations
#
# Note the dry run (`go run ./cmd/reset -file scripts/reset_demo.sql`) counts
# PROBLEMS only, so it under-reports what this removes -- it says nothing about the
# accounts. Check with psql if you want the full picture before running.
reset-demo:
	go run ./cmd/reset -file scripts/reset_demo.sql confirm

# Step 3 of the demo pipeline: shift the freshly seeded rows into the past so the
# board reads as lived-in rather than all-filed-today. Deletes nothing, despite
# running through cmd/reset (which is just the multi-statement SQL runner).
backdate-demo:
	go run ./cmd/reset -file scripts/backdate_demo.sql confirm

# Run the Go unit tests.
test:
	go test ./...

# Run the api locally (expects DATABASE_URL in the environment).
run:
	go run ./cmd/api

# Compile both binaries locally.
build:
	go build ./...

# Sync module dependencies.
tidy:
	go mod tidy

# Format all Go files.
fmt:
	gofmt -w .
