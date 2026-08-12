import { client } from './client.js'

/**
 * @param {{area?: string, status?: string, official?: string}} [filters]
 * @returns {Promise<import('../types/models.js').Problem[]>}
 */
export function listProblems(filters) {
  return client.get('/problems', { params: filters }).then((res) => res.data)
}

/**
 * The route is ungated but not caller-blind: a `PendingApproval` report is
 * readable only by its reporter and its own union's admin, and 404s for anyone
 * else. The identity comes from the JWT, never a parameter.
 *
 * @param {string} id
 * @returns {Promise<import('../types/models.js').Problem>}
 */
export function getProblem(id) {
  return client.get(`/problems/${id}`).then((res) => res.data)
}

/**
 * @param {{title: string, description: string, location: import('../types/models.js').Location, pointedOfficialId: string, proposedSolution?: string, imageUrl?: string}} payload
 * @returns {Promise<import('../types/models.js').Problem>}
 */
export function reportProblem(payload) {
  return client.post('/problems', payload).then((res) => res.data)
}

/**
 * The caller's own reports, at every status — including any taken down as
 * Rejected, with the ground attached.
 *
 * Takes NO parameter: the reporter is derived from the token. There is
 * deliberately no way to ask for someone else's reports (Scaffold §5, "There is
 * no reporter filter").
 *
 * @returns {Promise<import('../types/models.js').Problem[]>}
 */
export function listMyProblems() {
  return client.get('/me/problems').then((res) => res.data)
}

/**
 * @param {string} id
 * @param {import('../types/models.js').ValidationChoice} vote
 * @returns {Promise<import('../types/models.js').Problem>}
 */
export function validateProblem(id, vote) {
  return client.post(`/problems/${id}/validate`, { vote }).then((res) => res.data)
}

/**
 * Edit the caller's own report. The backend refuses (403) anyone who is not the
 * reporter and (409) any report past its editable window; the caller is taken from
 * the token, never the payload. areaId and pointedOfficialId are fixed at filing
 * and ignored even if sent.
 *
 * @param {string} id
 * @param {{title: string, description: string, location: import('../types/models.js').Location, proposedSolution?: string}} payload
 * @returns {Promise<import('../types/models.js').Problem>}
 */
export function updateProblem(id, payload) {
  return client.patch(`/problems/${id}`, payload).then((res) => res.data)
}

/**
 * Withdraw the caller's own report — a soft delete: it becomes `Withdrawn` and
 * stays on the public record with the reason on its audit trail. The backend
 * refuses (403) a non-reporter and (409) an already-assigned report.
 *
 * @param {string} id
 * @param {string} [note] optional free-text reason
 * @returns {Promise<import('../types/models.js').Problem>}
 */
export function withdrawProblem(id, note) {
  return client.post(`/problems/${id}/withdraw`, { note }).then((res) => res.data)
}

/**
 * Permanently delete the caller's own report. This is not withdraw's stronger
 * sibling — it is a different thing. Nothing survives it: the report, the
 * validation votes and suggestions others left on it, any case an official opened
 * (their plan, updates and evidence), and its audit trail are all erased, and the
 * official's scorecard changes to match. There is no undo and no window: it works
 * at any status. Prefer `withdrawProblem` wherever a retraction will do.
 *
 * The backend refuses (403) a non-reporter and (404) an unknown report. Responds
 * 204 with no body, so there is nothing to return.
 *
 * @param {string} id
 * @returns {Promise<void>}
 */
export function deleteProblem(id) {
  return client.delete(`/problems/${id}`).then(() => undefined)
}
