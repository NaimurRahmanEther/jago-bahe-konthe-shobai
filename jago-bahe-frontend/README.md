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

For Vercel + Render hosting, set Vercel's `VITE_API_URL` to the Render API URL
including `/api`, then rebuild. Set Render's `ALLOWED_ORIGINS` to your exact Vercel
origin (no trailing slash). The included `vercel.json` handles SPA deep links.
Use direct browser-to-Render API calls with the CORS settings described in the backend README.

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
