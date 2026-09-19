# Backend hosting

## Render

The root render.yaml creates a free Docker API in Singapore. It uses an
externally managed PostgreSQL database such as Neon; no worker or Render
database is deployed.

In the Render service environment, set `DATABASE_URL` to your external database
connection string and set `ALLOWED_ORIGINS` to your frontend origin. Render
generates the JWT secret. At each startup, the container runs migrations and
starts the API only if they succeed. This uses no paid pre-deploy hook. Health
checks use /api/health on port 10000. Automatic deploys are disabled; deploy the
API manually after pushing changes.

The free API sleeps after 15 minutes without traffic. Your external database's
availability, backup, and billing rules are managed by its provider. This is a
temporary demo setup. Stay within free usage allowances and review billing
limits in Render. See https://render.com/docs/free for current limitations.

Automatic overdue escalation and observation updates are not scheduled because
the worker is no longer deployed. The worker source remains available for manual use.

Production mode refuses the three publicly documented seed passwords at login,
including re-hashed copies, and refuses using them for new accounts. Seed records
and history remain intact. Development-mode demo logins still work. Activate each
legitimate seeded account by assigning a unique password with the included command.

Free Render services do not provide a service Shell. To activate a seeded account,
run the password command locally from the backend directory against the deployed
DB: set DATABASE_URL to Render's external connection URL, temporarily allow your
own public IP in database access settings, then run:

```powershell
$secret = Read-Host 'New password' -AsSecureString
$credential = New-Object System.Net.NetworkCredential('', $secret)
$credential.Password | go run ./cmd/setpassword acct-seed-super-1
Remove-Variable secret, credential
```

Remove the temporary database IP access rule afterwards. Never commit database
credentials. This updates only the explicitly named existing account.

Use an independently generated password of 12-72 bytes for each account. The
command reads stdin, prints no password, and updates only that account's password
hash. It does not change permissions or delete data. For union admin IDs, see
`migrations/000018_seed_dhamoirhat.up.sql`. If the API was previously public with
demo credentials, rotate JWT_SECRET and redeploy to invalidate all existing tokens;
password changes alone cannot revoke previously issued JWTs.

### Vercel connection

Use a direct browser connection from Vercel to Render:

1. In Render's Blueprint prompt, set `ALLOWED_ORIGINS` to the exact Vercel origin,
   e.g. `https://your-project.vercel.app`, without a trailing slash. Add custom
   domains as comma-separated origins. Wildcards are rejected.
2. Set Vercel's `VITE_API_URL` to `https://your-api.onrender.com/api` and rebuild.
3. The frontend `vercel.json` contains only the SPA fallback. Remove any older
   `/api` external rewrite or dashboard proxy rule. This supersedes the earlier
   recommendation to proxy API requests through Vercel.
4. Check login, writes, OPTIONS requests and the API health endpoint after deploy.

Check deployment logs and perform live integration checks before public launch.

Field definitions: [Render Blueprint reference](https://render.com/docs/blueprint-spec).

The Dockerfile lives in `jago-bahe-backend/`; `docker-compose.yml` lives at the
repository root. It runs the backend API, PostgreSQL, and migrations on `app-net`. The two-stage Dockerfile builds static
Go binaries and copies them and the migrations into a non-root Alpine runtime.

## Start from the repository root

```sh
cp jago-bahe-backend/.env.docker.example jago-bahe-backend/.env
# Set independent random POSTGRES_PASSWORD and JWT_SECRET values in that file.
docker compose --env-file jago-bahe-backend/.env config --quiet
docker compose --env-file jago-bahe-backend/.env up -d --build --remove-orphans
docker compose --env-file jago-bahe-backend/.env ps -a
curl --fail http://localhost:8080/api/health
```

If the backend .env already exists, add the missing settings without overwriting
it. Generate each secret with `openssl rand -hex 32`. Single-quote values containing
dollar signs. Never commit .env. The development JWT placeholder is rejected in
production. The build context excludes .env files.

Use `--env-file` in every command: service `env_file` supplies container variables,
while the CLI option also supplies Compose's `${POSTGRES_PASSWORD}` substitutions.
The project uses POSTGRES_USER, POSTGRES_PASSWORD and POSTGRES_DB. The Go processes
use DATABASE_URL plus PGUSER, PGPASSWORD and PGDATABASE; DB_HOST/DB_PORT are not
application settings. Compose overrides local database settings to connect to `db`.

The API is published on port 8080 on all host interfaces. Configure your server
firewall and HTTPS reverse proxy for hosting. PostgreSQL has no published host port.
The API starts only after the database is healthy and migrations finish.

## Updates and operations

Back up the database, pull the updated code, and run from the repository root:

```sh
docker compose --env-file jago-bahe-backend/.env build --pull
docker compose --env-file jago-bahe-backend/.env stop backend
docker compose --env-file jago-bahe-backend/.env run --rm migrate
# Continue only if migrations succeeded:
docker compose --env-file jago-bahe-backend/.env up -d --remove-orphans
docker compose --env-file jago-bahe-backend/.env logs --tail=100 backend migrate
```

When switching from the previous backend-directory Compose file, stop the old
stack first. The project name `jago-bahe-backend` and volume key `pgdata` preserve
the previous default database volume. If you previously used a custom project
name, continue supplying that name with `-p`.

`docker compose --env-file jago-bahe-backend/.env down` preserves the database.
Adding `-v` permanently deletes its volume. `make up`, `make down`, and `make migrate`
from the backend directory also use the root Compose file.
Changing POSTGRES_PASSWORD in .env does not rotate an existing database password;
update the password inside PostgreSQL too.

Migrations use this project's `cmd/migrate`. Existing databases created using the
old `migrate/migrate` image need migration-history reconciliation before using this
runner because their `schema_migrations` table format differs. Back up first.
Review seeded accounts and credentials in `migrations/` before public hosting.
