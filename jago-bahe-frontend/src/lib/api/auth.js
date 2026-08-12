import { client } from './client.js'

// Scaffold Spec §5 lists register's key body fields as name/phone/nid/unionId,
// but login needs phone+password, so a password is sent too. Confirmed against
// the real DTO (identity/interfaces/http/dto.go registerRequest), which takes
// name/phone/password/nid/unionId — the field list in §5 is "key body fields",
// not an exhaustive one.

/**
 * @param {{name: string, phone: string, password: string, nid: string, unionId: string}} payload
 * @returns {Promise<{token: string}>}
 */
export function register(payload) {
  return client.post('/auth/register', payload).then((res) => res.data)
}

/**
 * Register an official against an EXISTING directory office. Registration never
 * creates an office — the directory records real elections — so the payload names
 * an officialId to claim and carries no tier or area of its own (the office
 * supplies both). The backend returns a token, but the caller must NOT treat this
 * as a session: the account acts as nothing until a reviewer approves the claim,
 * so RegisterOfficial navigates to /login with a claim-pending notice instead.
 * @param {{name: string, phone: string, password: string, nid: string, officialId: string}} payload
 * @returns {Promise<{token: string, user: import('../types/models.js').User}>}
 */
export function registerOfficial(payload) {
  return client.post('/auth/register/official', payload).then((res) => res.data)
}

/**
 * @param {{phone: string, password: string}} payload
 * @returns {Promise<{token: string, role: import('../types/models.js').Role, user: import('../types/models.js').User}>}
 */
export function login(payload) {
  return client.post('/auth/login', payload).then((res) => res.data)
}
