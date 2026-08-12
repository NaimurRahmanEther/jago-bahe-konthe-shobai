import { client } from './client.js'

// Identity review: official claims, resident verification, and the super admin's
// oversight feed. Every call takes its actor from the JWT — never a caller param —
// and the backend decides who may act (an admin over its own union's residents and
// union-level claims; the super admin over above-union claims and oversight).

/**
 * The claims this reviewer may decide. ONE endpoint for both admin and super
 * admin: the server filters the list by the office's tier (a union admin sees
 * union-level claims, the super admin sees above-union ones), so the client never
 * branches on role — CLAUDE.md A.1.1 / the identity handler's own note.
 * @returns {Promise<import('../types/models.js').PendingClaim[]>}
 */
export function listPendingClaims() {
  return client.get('/claims/pending').then((res) => res.data)
}

/**
 * Approve a claim — binds the account to the office it named. This is what sets
 * accounts.official_id, the link the official's dashboard and admin scoping both
 * depend on.
 * @param {string} claimId
 * @returns {Promise<import('../types/models.js').OfficialClaim>}
 */
export function approveClaim(claimId) {
  return client.post(`/claims/${claimId}/approve`).then((res) => res.data)
}

/**
 * Reject a claim with a reason shown to the claimant. The reason is required —
 * the backend refuses an empty one.
 * @param {string} claimId
 * @param {{reason: string}} payload
 * @returns {Promise<import('../types/models.js').OfficialClaim>}
 */
export function rejectClaim(claimId, payload) {
  return client.post(`/claims/${claimId}/reject`, payload).then((res) => res.data)
}

/**
 * The admin's OWN union's unverified residents. Only a verified resident counts
 * toward V, so this queue is the gate that makes the validity threshold reachable.
 * @returns {Promise<import('../types/models.js').User[]>}
 */
export function listPendingResidents() {
  return client.get('/admin/residents/pending').then((res) => res.data)
}

/**
 * Verify a resident of the admin's own union.
 * @param {string} residentId account id
 * @returns {Promise<{verified: boolean}>}
 */
export function verifyResident(residentId) {
  return client.post(`/admin/residents/${residentId}/verify`).then((res) => res.data)
}

/**
 * The super admin's oversight feed: recent moderator decisions across the seat. It
 * is a read-only projection of the audit log — the record the platform already
 * keeps — so its entries are AuditEntry-shaped and there is deliberately no
 * counterpart that reverses any decision (Scaffold §2, "Oversight, not override").
 * @returns {Promise<import('../types/models.js').AuditEntry[]>}
 */
export function getOversight() {
  return client.get('/super/oversight').then((res) => res.data)
}
