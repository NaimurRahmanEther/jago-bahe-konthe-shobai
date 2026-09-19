<<<<<<< HEAD
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
=======
# জাগো বাহে কণ্ঠে সবাই (Jago Bahe Konthe Shobai)

[![Go](https://img.shields.io/badge/backend-Go%201.22-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/frontend-React%2019-61DAFB?logo=react&logoColor=black)](https://react.dev)
[![PostgreSQL](https://img.shields.io/badge/database-PostgreSQL%2016-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org)
[![Vite](https://img.shields.io/badge/bundler-Vite-646CFF?logo=vite&logoColor=white)](https://vitejs.dev)
[![License](https://img.shields.io/badge/license-unspecified-lightgrey)](#license)

A civic accountability platform for reporting and tracking local infrastructure and public-service problems in Bangladesh's local government areas (unions and upazilas). Residents report problems, the community validates them, the responsible official is assigned automatically by administrative tier, and progress is tracked publicly through to resolution — with a scorecard showing how responsive each official actually is.

## Table of contents

- [How it works](#how-it-works)
- [Tech stack](#tech-stack)
- [Project structure](#project-structure)
- [Getting started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Backend setup](#backend-setup)
  - [Frontend setup](#frontend-setup)
- [Configuration](#configuration)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## How it works

- **Residents** report a problem (e.g. a broken road, a water supply issue) tied to their union.
- The report is public and votable immediately — there is no admin screening gate before publication. The community is the first filter (`VALIDITY_THRESHOLD` verified residents).
- Once a problem gathers enough verified-resident support, it is **assigned** to the responsible official based on a tier hierarchy (ward member → union chairman → upazila chairman → MP, etc.).
- Officials record **observations** and **resolutions**; a genuine blocker gets a capped review window instead of an indefinite clock-stop.
- A missed **response deadline** triggers escalation.
- **Union admins** moderate and assign within their own union. A **super admin** has cross-union oversight and decides above-union escalations, but cannot override a union admin's decisions directly — no single moderator holds concentrated power.
- A public **scorecard** and **official directory** make each representative's track record visible.
- Officials can **claim** their seat in the public directory; the claim is verified by the relevant union admin, or by the super admin for above-union tiers.

## Tech stack

| | |
|---|---|
| **Backend** | Go 1.22 · PostgreSQL (`pgx`) · JWT auth (`golang-jwt`) · `bcrypt` |
| **Frontend** | React 19 · Vite · Tailwind CSS v4 · TanStack React Query · React Router v7 · i18next |
| **Testing** | Go `testing` · Vitest · React Testing Library |
| **Infra** | Docker Compose (Postgres + migrate + API + worker) |

## Project structure

```
jago-bahe-konthe-shobai/
├── jago-bahe-backend/      Go API, worker, and Postgres migrations
│   ├── cmd/                Entry points: api, worker, migrate, reset
│   ├── internal/           Domain modules (identity, problem, assignment, resolution, suggestion, scorecard, shared)
│   ├── config/             Typed, environment-driven configuration
│   ├── migrations/         SQL migrations
│   └── scripts/            Demo/reset SQL scripts
└── jago-bahe-frontend/     React (Vite) single-page app
    └── src/
        ├── pages/          Routes split by role: public, resident, official, admin, super, auth
        ├── hooks/          Data-fetching hooks (TanStack Query)
        ├── lib/            API client, i18n, formatting helpers
        └── components/     Shared UI components
```

Each backend domain module (`internal/<module>`) follows the same layered shape: `domain` → `application` → `infrastructure` → `interfaces`.

## Getting started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/) and npm
- [Docker](https://www.docker.com/) (recommended for the backend) or a local PostgreSQL 16 instance

### Backend setup

**With Docker (recommended):**

```bash
cd jago-bahe-backend
cp .env.example .env      # adjust values if needed
make up                   # starts postgres + migrate + api + worker
```

The API listens on `http://localhost:8080` by default.

**Without Docker** (requires a local PostgreSQL instance):

```bash
cd jago-bahe-backend
cp .env.example .env      # set DATABASE_URL to your local Postgres
make migrate-up           # apply migrations
make run                  # go run ./cmd/api
```

Other useful commands:

```bash
make test            # run Go unit tests
make build           # compile the api and worker binaries
make migrate-down    # roll back migrations
make reset-problems  # wipe problems for a clean manual test round (dry run without `confirm`)
```

### Frontend setup

```bash
cd jago-bahe-frontend
npm install
npm run dev
```

The Vite dev server proxies `/api` to `localhost:8080`, so no CORS setup or `VITE_API_URL` is required as long as the backend runs locally on its default port.

```bash
npm run build     # production build
npm run lint       # oxlint
```

## Configuration

Backend configuration is entirely environment-driven (see `jago-bahe-backend/.env.example`):

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `HTTP_ADDR` | API listen address (default `:8080`) |
| `JWT_SECRET`, `JWT_TTL` | Auth token signing secret and lifetime |
| `VALIDITY_THRESHOLD` | Distinct verified residents needed to validate a report |
| `RESPONSE_DEADLINE` | Time an official has to give a first response before escalation |
| `BLOCKER_REVIEW_WINDOW` | Capped time allowed for a blocker to be reviewed |
| `AUTH_RATE_LIMIT`, `WRITE_RATE_LIMIT`, `RATE_LIMIT_WINDOW` | Per-IP rate limits for auth and write endpoints |
| `APP_ENV` | Set to `production` to reject the default dev JWT secret at boot |

The frontend needs no configuration to run against a local backend. Set `VITE_API_URL` only to point it at a remote or non-default API (see `jago-bahe-frontend/.env.example`).

## Testing

```bash
# Backend
cd jago-bahe-backend && make test

# Frontend
cd jago-bahe-frontend && npm run test
```

## Contributing

Issues and pull requests are welcome. Before submitting a change:

1. Run the relevant test suite (`make test` for backend, `npm run test` for frontend).
2. Run the linter (`npm run lint` for frontend; `make fmt` formats Go code).
3. Keep domain logic free of infrastructure concerns — new backend code should follow the existing `domain → application → infrastructure → interfaces` layering.

## License

No license has been specified for this repository yet. All rights are reserved by the author until one is added.
>>>>>>> af67d1683b62174f7db14ffcb3dae5dfea7ccd7f
