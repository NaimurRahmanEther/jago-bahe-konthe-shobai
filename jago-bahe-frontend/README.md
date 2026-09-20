# Frontend

React application built with Vite, TanStack Query, and i18next. Use Node.js 24
(the runtime used for the current checks) and install locked dependencies:

```sh
npm ci
npm run dev
```

Start the backend on port 8080. Vite forwards same-origin `/api` requests there.
Optional `VITE_API_URL` changes the browser's API base URL; a different origin
requires CORS support from that API. Values prefixed with VITE_ are public build-time
configuration: never put secrets in them.

For Vercel + Render hosting, use these Vercel project settings:

| Setting | Value |
| --- | --- |
| Root Directory | `jago-bahe-frontend` |
| Framework Preset | Vite |
| Build Command | `npm run build` |
| Output Directory | `dist` |
| Environment variable (Production and Preview) | `VITE_API_URL=/api` |

Copy `.env.example` to `.env` for local configuration. The local `.env` is ignored
by Git; set the variable in the Vercel dashboard separately. Deploy the current
`vercel.json` with the frontend: its first rewrite forwards `/api/:path*` to
`https://jago-bahe-api.onrender.com/api/:path*`, before the SPA fallback.
Redeploy after changing environment variables because Vite embeds them at build time.

Check `https://jago-bahe-konthe-shobai.vercel.app/api/health` after deployment.
It should return JSON with `db: true`, not HTML. HTML means the deployed API
rewrite is missing or the project Root Directory is wrong. `/api/areas` should
also return JSON. The bare Render `/api` path has no handler; use `/api/health`
to check the API. No backend router change is needed.

Alternatively, direct browser-to-Render requests require both settings:

```dotenv
# Vercel environment (and local frontend .env if desired)
VITE_API_URL=https://jago-bahe-api.onrender.com/api
# Render environment
ALLOWED_ORIGINS=https://jago-bahe-konthe-shobai.vercel.app
```

Redeploy both services for that alternative. The API base must include `/api`;
the allowed origin must have no path or trailing slash. With the recommended
same-origin Vercel proxy, browser requests do not need cross-origin CORS headers.

```sh
npm run lint
npm test
npm run build
```

The production bundle is written to `dist/`. Serve it with an SPA fallback to
`index.html` and proxy `/api` to the backend. `npm run preview` is for checking the
built bundle locally. Production settings are embedded at build time.

`vite.config.js` configures the application build and development proxy.
`vitest.config.js` configures tests independently, so Vitest's esbuild configuration
does not conflict with Vite's Oxc pipeline. Tests are colocated with components and
hooks; shared setup lives in `src/test/`.

The live API test suite is opt-in. Use a disposable seeded backend and set
`VITE_LIVE=1` and `VITE_API_URL=http://localhost:8080/api` before running
`npx vitest run src/lib/api/live.integration.test.js`. Those tests perform writes.

See the [root README](../README.md) for folder ownership and contribution conventions.
