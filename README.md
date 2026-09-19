# Jago Bahe

A Go/PostgreSQL backend and React/Vite frontend for community reporting and resolution.

## Repository layout

```text
 docker-compose.yml          Backend deployment stack
 jago-bahe-backend/
   cmd/api/                  HTTP server and dependency wiring
   cmd/worker/               Scheduled background work
   cmd/migrate/              Database migration command
   config/                   Environment parsing and validation
   internal/<feature>/
     domain/                 Business rules and repository contracts
     application/            Use cases and application ports
     infrastructure/postgres/ PostgreSQL implementations
     interfaces/http/        Handlers and request/response DTOs
   internal/shared/          Shared business capabilities
   pkg/                      Reusable technical helpers
   migrations/               Ordered database changes
   scripts/                  Explicit maintenance operations
 jago-bahe-frontend/
   src/pages/                Route screens grouped by audience/feature
   src/components/           Feature components and shared UI primitives
   src/hooks/                Queries, mutations, and reusable UI behavior
   src/lib/api/              HTTP endpoint wrappers
   src/lib/types/            Shared JSDoc model definitions
   src/auth/                 Context, provider, and authentication hook
   src/routes/               Lazy-loaded routes and client route guards
   src/config/               Frontend environment settings
   src/lib/i18n/             Translations and localization setup
   src/test/                 Shared test setup
```

Keep the existing feature boundaries. Domain and application packages should not
import HTTP handlers or PostgreSQL implementations. Wire dependencies in `cmd/`.
Keep SQL in repositories and transport validation in handlers. Add migrations
instead of rewriting migrations that have already been deployed.

Frontend pages compose components and hooks; endpoint URLs belong in `lib/api`.
Keep query keys and invalidation rules in the corresponding hook module. Key
private queries by user identity where possible; the auth provider also clears
cached data on session changes. Route guards control presentation, while the
backend remains responsible for authorization. Put user-facing copy in the
translation file. Keep tests next to the code they exercise.

Do not move files just to match a template. Introduce a new folder when it has a
clear owner and responsibility; avoid generic catch-all helper folders and barrel
exports that obscure dependencies. `.editorconfig` defines shared whitespace rules.

## Development and checks

See [backend hosting](jago-bahe-backend/README.md) and
[frontend setup](jago-bahe-frontend/README.md).

Backend, from `jago-bahe-backend`:

```sh
go test ./...
go vet ./...
```

Format changed Go files with `gofmt -w <files>`. Database integration tests require
`TEST_DATABASE_URL` pointing to a dedicated migrated and seeded test database.
They are skipped without it; unit-test success does not validate deployment SQL.

Frontend, from `jago-bahe-frontend`:

```sh
npm ci
npm run lint
npm test
npm run build
```

Commit `package-lock.json` and `go.sum`. Keep `.env` files, generated bundles,
node_modules, and build outputs out of version control. Use environment examples
for configuration documentation. Review configuration, authorization, loading/error
states, and relevant tests with each behavior change.

## Hosting checks still required

Exercise the full stack with a dedicated test database before public deployment.
Production rejects known seed passwords; activate intended accounts with the
backend's `setpassword` command. For Vercel/Render, configure direct API requests
and exact CORS origins as described in the backend README. The current browser session persists
a bearer token in localStorage; an HttpOnly-cookie design would require coordinated
backend authentication and CSRF changes, rather than a frontend-only substitution.
