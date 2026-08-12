import { client } from './client.js'

/**
 * The cases assigned to the signed-in official. Takes no parameter: the backend
 * infers the caller from the JWT.
 *
 * @returns {Promise<import('../types/models.js').Case[]>}
 */
export function listCases() {
  return client.get('/official/cases').then((res) => res.data)
}

/**
 * A problem's public progress: the official's plan, their answer to the
 * snapshotted top suggestion, the updates, and the fairness flag.
 *
 * Resolves to `null` — never rejects — when there is no case to report. An
 * unknown problem, an unscreened one, an unassigned one and a case no official
 * has opened yet are deliberately indistinguishable, so that telling them apart
 * cannot be used to enumerate what is awaiting screening.
 *
 * @param {string} problemId
 * @returns {Promise<import('../types/models.js').Progress|null>}
 */
export function getProgress(problemId) {
  return client.get(`/problems/${problemId}/progress`).then((res) => res.data)
}

/** @param {string} caseId @returns {Promise<import('../types/models.js').Case>} */
export function getCase(caseId) {
  return client.get(`/official/cases/${caseId}`).then((res) => res.data)
}

/**
 * @param {string} caseId
 * @param {{decision: 'accept' | 'dispute', reason?: string}} payload
 * @returns {Promise<import('../types/models.js').Case>}
 */
export function acknowledgeCase(caseId, payload) {
  return client.post(`/official/cases/${caseId}/acknowledge`, payload).then((res) => res.data)
}

/**
 * The plan is a week-by-week checklist: `tasks` is one task per week (week N =
 * position N), and the backend derives timelineWeeks from its length.
 *
 * @param {string} caseId
 * @param {{strategy: string, tasks: string[], obstacles: string, suggestionResponse: string}} payload
 * @returns {Promise<import('../types/models.js').Case>}
 */
export function submitPlan(caseId, payload) {
  return client.post(`/official/cases/${caseId}/plan`, payload).then((res) => res.data)
}

/**
 * Posts a replacement plan to restart a stalled case (Reopened, or InProgress
 * after an obstacle was resolved). Same shape as submitPlan plus a required
 * `reason` — the public "what changed" that lands on the timeline and audit. The
 * backend enforces the legal source states (409 otherwise).
 *
 * @param {string} caseId
 * @param {{strategy: string, tasks: string[], obstacles: string, suggestionResponse: string, reason: string}} payload
 * @returns {Promise<import('../types/models.js').Case>}
 */
export function revisePlan(caseId, payload) {
  return client.post(`/official/cases/${caseId}/replan`, payload).then((res) => res.data)
}

/**
 * Marks one weekly task of the case's plan completed (a public milestone — it
 * changes no case status). The backend infers the acting official from the JWT and
 * refuses anyone but the case's owner.
 *
 * @param {string} caseId
 * @param {string} taskId
 * @returns {Promise<import('../types/models.js').Case>}
 */
export function completeTask(caseId, taskId) {
  return client.post(`/official/cases/${caseId}/tasks/${taskId}/complete`).then((res) => res.data)
}

/**
 * @param {string} caseId
 * @param {{kind: 'progress' | 'obstacle', text: string}} payload
 * @returns {Promise<import('../types/models.js').Case>}
 */
export function postUpdate(caseId, payload) {
  return client.post(`/official/cases/${caseId}/updates`, payload).then((res) => res.data)
}

/**
 * @param {string} caseId
 * @param {{beforeImageUrl: string, afterImageUrl: string}} payload
 * @returns {Promise<import('../types/models.js').Case>}
 */
export function uploadEvidence(caseId, payload) {
  return client.post(`/official/cases/${caseId}/evidence`, payload).then((res) => res.data)
}

/** @param {string} caseId @returns {Promise<import('../types/models.js').Case>} */
export function markDone(caseId) {
  return client.post(`/official/cases/${caseId}/done`).then((res) => res.data)
}

/**
 * @param {string} caseId
 * @param {{category: string, whatBlocks: string, whoUnblocks: string, proofTried: string}} payload
 * @returns {Promise<import('../types/models.js').Case>}
 */
export function reportObstacle(caseId, payload) {
  return client.post(`/official/cases/${caseId}/report-obstacle`, payload).then((res) => res.data)
}

/**
 * Only the reporting resident may confirm; the backend infers them from the JWT
 * and refuses anyone else (Scaffold §2).
 *
 * @param {string} problemId
 * @param {{outcome: 'solved' | 'not_solved'}} payload
 * @returns {Promise<import('../types/models.js').Problem>}
 */
export function confirmProblem(problemId, payload) {
  return client.post(`/problems/${problemId}/confirm`, { outcome: payload.outcome }).then((res) => res.data)
}
