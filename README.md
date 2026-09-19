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
