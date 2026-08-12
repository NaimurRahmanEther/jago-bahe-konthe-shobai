// Default to the same-origin /api path, which the Vite dev proxy forwards to the
// backend (avoids CORS). Set VITE_API_URL to an absolute URL to hit a remote API
// directly (that backend must then send CORS headers).
//
// This is the only env the app reads. There is no mock switch: the mock layer was
// deleted in F15 and must not come back — see CLAUDE.md A.5.5.
export const API_URL = import.meta.env.VITE_API_URL ?? '/api'
