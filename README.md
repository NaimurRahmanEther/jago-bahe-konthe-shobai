# জাগো বাহে কণ্ঠে সবাই

### Jago Bahe Konthe Shobai — Community reporting and public accountability

A Bangla-focused civic platform connecting residents, local representatives, and administrators around public-service problems in Bangladesh. Report an issue, contribute suggestions, follow an official's response, and track the outcome through public progress and accountability records.

[Frontend](https://jago-bahe-konthe-shobai.vercel.app/) · [Backend health](https://jago-bahe-api.onrender.com/api/health) · [Frontend guide](jago-bahe-frontend/README.md) · [Backend operations](jago-bahe-backend/README.md)

## Contents

- [Features and roles](#features-and-roles)
- [Report workflow](#report-workflow)
- [Technology and architecture](#technology-and-architecture)
- [Run locally](#run-locally)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [API overview](#api-overview)
- [Development and testing](#development-and-testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features and roles

| Role | Capabilities |
| --- | --- |
| Visitor | Browse published problems, geography, officials, public activity, progress, and scorecards. |
| Resident | Submit and manage reports; eligible verified residents can validate issues, contribute suggestions, and participate in resolution workflows. |
| Official | Claim an official position, acknowledge assigned cases, submit plans, post updates and evidence, report obstacles, and mark work done. |
| Union admin | Review reports, verify residents, review eligible official claims, assign problems within the union, and advise on above-union forwarding. |
| Super admin | Review oversight records and decide above-union forwarding through the applicable role-controlled workflows. |

Other features include suggestion upvotes, obstacle review and unblocking plans, in-app notifications, public decision records, and observation lists for monitoring overdue cases.

## Report workflow

1. **Report:** a resident submits an issue associated with an area. New reports start as `PendingApproval`.
2. **Review:** the union admin approves publication or rejects the report on a supported ground.
3. **Community input:** published reports receive validation votes and suggestions. Reaching the configured vote threshold changes `Reported` to `Validated`.
4. **Assignment:** admins assign eligible reports within their union. Above-union forwarding is decided by the super admin with advice from union admins. Validation counts inform decisions; reaching the threshold is not required for assignment.
5. **Work and evidence:** the official acknowledges the case, records a plan, provides updates, and uploads evidence. Obstacles have their own review workflow.
6. **Resolution:** completed work enters the confirmation process, with progress and outcomes recorded publicly.

Reports can also be withdrawn, rejected, blocked, or reopened according to the applicable rules.

**Background processing:** the worker handles overdue escalation and observation updates. It runs separately from the API. The current Render blueprint and Docker Compose configuration do not start a worker.

## Technology and architecture

| Layer | Technology |
| --- | --- |
| Frontend | React 19, Vite 8, React Router 7 |
| UI and localization | Tailwind CSS 4, i18next |
| API requests and server state | Axios, TanStack React Query |
| Backend | Go 1.22+, standard-library HTTP routing |
| Database | PostgreSQL, pgx |
| Authentication | JWT, bcrypt, role-based route guards |
| Testing | Go testing, Vitest, React Testing Library |
| Hosting configuration | Vercel frontend, Docker API on Render, external PostgreSQL |

```text
.
├── jago-bahe-frontend/
│   ├── src/
│   │   ├── components/       Shared interface components
│   │   ├── config/           Frontend environment configuration
│   │   ├── hooks/            Data-fetching and application hooks
│   │   ├── lib/              API client and shared utilities
│   │   └── pages/            Public and role-specific screens
│   └── vercel.json           API proxy and SPA routing
├── jago-bahe-backend/
│   ├── cmd/                  API, worker, migration and maintenance commands
│   ├── config/               Environment loading and validation
│   ├── internal/             Business contexts and shared services
│   ├── migrations/           Database schema and seed migrations
│   ├── pkg/                  Authentication, HTTP and security helpers
│   └── Dockerfile
├── docker-compose.yml
└── render.yaml
```

Backend contexts separate domain rules, application use cases, persistence, and HTTP interfaces. The [API composition root](jago-bahe-backend/cmd/api/router.go) connects repositories, adapters, handlers, and middleware. Business contexts cover identity, problems, suggestions, assignments, resolution, and scorecards; geography, audit, and notifications are shared services.

## Run locally

### Prerequisites

- Go 1.22 or newer.
- Node.js 24 and npm, matching the frontend guide's development runtime.
- An existing PostgreSQL database; the optional Docker database uses PostgreSQL 16.
- Git. Docker Compose is needed only for the container setup.

Run these commands from your cloned repository. Copy environment examples only when the destination `.env` does not already exist.

### Backend

```sh
cd jago-bahe-backend
cp .env.example .env
```

In PowerShell, you can use `Copy-Item .env.example .env`. Edit the file to point to your database:

```dotenv
DATABASE_URL=postgres://postgres:YOUR_PASSWORD@localhost:5432/jago?sslmode=disable
HTTP_ADDR=:8080
JWT_SECRET=YOUR_RANDOM_SECRET
JWT_TTL=24h
```

For hosted PostgreSQL, use your provider's connection string and TLS parameters. Then apply migrations and start the API:

```sh
go mod download
go run ./cmd/migrate up
go run ./cmd/api
```

Open `http://localhost:8080/api/health`. A healthy connection returns:

```json
{"db":true,"status":"ok"}
```

To enable overdue escalation, open another terminal in `jago-bahe-backend` and run:

```sh
go run ./cmd/worker
```

### Frontend

In another terminal, from the repository root:

```sh
cd jago-bahe-frontend
npm ci
npm run dev
```

Open the URL printed by Vite. The frontend defaults to `/api`, and the development proxy forwards those requests to `http://localhost:8080`. No frontend environment file is required for this local setup.

On Windows, use `npm.cmd` if PowerShell blocks the `npm.ps1` launcher.

### Docker alternative

From the repository root, copy `jago-bahe-backend/.env.docker.example` to `jago-bahe-backend/.env` if no environment file exists. Set a real `JWT_SECRET` and a `DATABASE_URL` reachable from the containers, then run:

```sh
docker compose --env-file jago-bahe-backend/.env config --quiet
docker compose --env-file jago-bahe-backend/.env up -d --build
```

This starts migrations and the API in production mode. A hosted PostgreSQL database can be used directly; the optional `local-db` profile adds a database container. See the [backend operations guide](jago-bahe-backend/README.md) for profile settings and maintenance. Run the frontend separately.

## Configuration

Backend settings belong in `jago-bahe-backend/.env` locally or in the backend host's environment settings.

| Variable | Purpose | Default or requirement |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `HTTP_ADDR` | API listening address | `:8080` |
| `APP_ENV` | Production validation and password safeguards | Set to `production` when hosting |
| `JWT_SECRET` | Token signing secret | Use a random secret; the development placeholder is rejected in production |
| `JWT_TTL` | Token lifetime | `24h` |
| `ALLOWED_ORIGINS` | Comma-separated browser origins for direct cross-origin API calls | Exact origins without paths, wildcards, or trailing slashes |
| `VALIDITY_THRESHOLD` | Distinct valid votes needed for `Validated` status | `5` |
| `RESPONSE_DEADLINE` | Official response deadline | `168h` |
| `BLOCKER_REVIEW_WINDOW` | Obstacle review window | `72h` |

The frontend reads **`VITE_API_URL`**, defaulting to `/api`. Absolute API URLs must include `/api`. Frontend values are embedded during the build, so changing them requires rebuilding. Never put database credentials or JWT secrets in `VITE_` variables.

Actual `.env` files are ignored by Git. Use the [backend example](jago-bahe-backend/.env.example) and [frontend example](jago-bahe-frontend/.env.example) as templates.

## Deployment

### Render backend

The repository's [render.yaml](render.yaml) defines a Docker API service backed by external PostgreSQL.

1. Connect the repository to Render using the blueprint.
2. Supply `DATABASE_URL`. The blueprint generates `JWT_SECRET` and sets production configuration.
3. Set `ALLOWED_ORIGINS` to `https://jago-bahe-konthe-shobai.vercel.app` when prompted. This permits direct browser requests from that origin.
4. Deploy. The container applies migrations before starting the API.
5. Check the backend's `/api/health` endpoint.

The blueprint disables automatic deployments and does not provision a database or worker. Deploy backend updates manually; run a separate worker for automatic escalation. Production authentication rejects known demo passwords. The backend guide describes how to activate legitimate seeded accounts with unique passwords.

### Vercel frontend

| Setting | Value |
| --- | --- |
| Root Directory | `jago-bahe-frontend` |
| Framework Preset | `Vite` |
| Build Command | `npm run build` |
| Output Directory | `dist` |
| Environment variable name | `VITE_API_URL` |
| Environment variable value | `/api` |

Enter the variable name and value in **separate fields**, enable it for Production and Preview, and deploy the latest commit. Include [vercel.json](jago-bahe-frontend/vercel.json): its API rewrite must precede the SPA fallback.

```text
Browser → Vercel /api/... → Render /api/... → PostgreSQL
```

Browser API requests stay on the frontend origin. For a fork, update the rewrite's Render destination to your API hostname.

After deployment, open the frontend's `/api/health` and `/api/areas` URLs. Both should return JSON; health should report `db: true`.

**Direct connection alternative:** set Vercel's `VITE_API_URL` to `https://jago-bahe-api.onrender.com/api` and Render's `ALLOWED_ORIGINS` to the exact frontend origin. Redeploy both services. Add any additional frontend domains to the comma-separated origin list.

## API overview

Application endpoints use the `/api` prefix. Selected public endpoints:

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/` | API service identification |
| `GET` | `/api/health` | Service and database health |
| `GET` | `/api/areas` | Public geography |
| `GET` | `/api/problems` | Published problem feed |
| `GET` | `/api/officials` | Officials directory |
| `GET` | `/api/officials/{id}/scorecard` | Official accountability scorecard |
| `GET` | `/api/officials/{id}/record` | Official public record |
| `GET` | `/api/seat/overview` | Seat-level overview |
| `POST` | `/api/auth/register` | Resident registration |
| `POST` | `/api/auth/register/official` | Official registration |
| `POST` | `/api/auth/login` | Login |

Protected requests use `Authorization: Bearer <token>`, with role and business-rule checks. Each context's `interfaces/http` directory defines its full route set and request payloads.

The bare `/api` path has no health handler. Use `/api/health` to check connectivity.

## Development and testing

Backend, from `jago-bahe-backend`:

```sh
go test ./...
go build ./...
```

Frontend, from `jago-bahe-frontend`:

```sh
npm run lint
npm test
npm run build
```

The production frontend bundle is written to `dist/`. `npm run preview` serves the build for local inspection; it is not a production hosting setup.

Backend HTTP integration tests require `TEST_DATABASE_URL` pointing to a migrated, seeded test database. Frontend live tests are opt-in through `VITE_LIVE=1` and a suitable `VITE_API_URL`. These tests perform writes: use disposable test data. See the [frontend guide](jago-bahe-frontend/README.md) for the live-test command.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| Vercel `/api/health` returns HTML | Deploy the API rewrite and confirm the frontend Root Directory. |
| Requests hit the wrong host or path | Check `VITE_API_URL`, include `/api` for absolute URLs, and rebuild after changes. |
| Vercel rejects the variable name | Enter `VITE_API_URL` as the name and `/api` as the value, without backslashes or an equals sign in the name. |
| Direct Render requests fail with CORS errors | Add the exact frontend origin to `ALLOWED_ORIGINS` and redeploy the backend. |
| Health returns `db: false` | Check database availability, credentials, TLS settings, and backend logs. |
| A new report is absent from the feed | Reports remain private while `PendingApproval`; check moderation. |
| Overdue cases do not escalate | Verify that a separate worker is running against the same database. |
| Demo credentials fail in production | Production rejects known seed passwords; activate the account with a unique password. |

## Contributing

1. Create a branch for a focused change.
2. Keep backend business rules in the relevant domain or application layer and frontend API calls in the shared API modules.
3. Update tests when behavior changes and run the relevant checks.
4. Document new environment variables, migrations, and setup requirements.
5. Open a pull request describing the problem, resulting behavior, and validation performed.

Do not commit credentials, local environment files, or generated build output.

## License

No license file is currently included. An explicit license is needed to define permissions for reuse and redistribution.

