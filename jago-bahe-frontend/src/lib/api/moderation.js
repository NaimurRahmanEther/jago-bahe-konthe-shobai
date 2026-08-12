import { client } from './client.js'

/**
 * The four grounds a report may be taken down on. The backend is authoritative
 * (it rejects anything else with a 400); this mirrors the enum so the UI can
 * offer a select instead of free text. An admin may never reject on merit —
 * whether a genuine report is real is the community's call via V.
 * @type {readonly string[]}
 */
export const REJECTION_REASONS = ['spam', 'abusive', 'duplicate', 'wrong_area']

/**
 * The screening queue: the PendingApproval reports in the caller's union that are
 * awaiting their approve/reject decision. These are not yet public — the admin's
 * decision is what publishes or rejects them.
 * @returns {Promise<import('../types/models.js').Problem[]>}
 */
export function listModeration() {
  return client.get('/admin/moderation').then((res) => res.data)
}

/**
 * Approve a pending report: it becomes Reported (public but unvalidated) and
 * enters the feed. From there the community decides its merit via V.
 *
 * @param {string} problemId
 * @returns {Promise<import('../types/models.js').Problem>} the problem, now Reported (published)
 */
export function approveProblem(problemId) {
  return client.post(`/admin/problems/${problemId}/approve`).then((res) => res.data)
}

/**
 * Reject a report on one of the fixed grounds — during screening (a pending
 * report) or as a pre-assignment takedown of an already-public one. It becomes
 * Rejected, which is public with its reason attached.
 *
 * @param {string} problemId
 * @param {{reason: string, note?: string}} payload reason must be one of REJECTION_REASONS
 * @returns {Promise<import('../types/models.js').Problem>} the problem, now Rejected — public, with its reason
 */
export function rejectProblem(problemId, payload) {
  return client.post(`/admin/problems/${problemId}/reject`, payload).then((res) => res.data)
}
