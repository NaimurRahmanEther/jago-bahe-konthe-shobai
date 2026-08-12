import { client } from './client.js'

/**
 * The acting admin's queue — everything in THEIR OWN union they may still forward
 * to an official. Both Reported and Validated rows: since B17 the validation count
 * is evidence the admin weighs rather than a gate they wait on (CLAUDE.md A.3.1),
 * so `useAdminQueue` / `useValidatingQueue` split this one response by status.
 *
 * The union scope is the server's, from the JWT — there is no parameter for it.
 *
 * @returns {Promise<import('../types/models.js').QueueItem[]>}
 */
export function listQueue() {
  return client.get('/admin/queue').then((res) => res.data)
}

/**
 * @param {string} problemId
 * @param {{officialId: string, priority?: string, overrideReason?: string}} payload
 * @returns {Promise<import('../types/models.js').Assignment>}
 */
export function assignWithinUnion(problemId, payload) {
  return client.post(`/admin/problems/${problemId}/assign`, payload).then((res) => res.data)
}

/**
 * The above-union reports this admin may ADVISE on — scoped by advice scope (the
 * upazila, or the whole seat), which is a wider and different set than `listQueue`'s
 * own-union scope. The scope is the server's, from the JWT.
 *
 * These reports are deliberately absent from `listQueue`: an above-union report is
 * the super admin's to forward, so it is not on the list of what this admin may
 * assign (CLAUDE.md A.3.8).
 *
 * @returns {Promise<import('../types/models.js').ForwardingItem[]>}
 */
export function listMyForwarding() {
  return client.get('/admin/forwarding').then((res) => res.data)
}

/**
 * Record or REPLACE this admin's advice on where an above-union report should go.
 * It settles nothing — there is no quorum and no window, and the super admin may
 * forward at any count including none. Advice is revisable, so calling this again
 * replaces the previous row rather than adding one.
 *
 * The adviser comes from the JWT — advice is never client-asserted.
 *
 * @param {string} problemId
 * @param {{officialId: string, reason?: string}} payload
 * @returns {Promise<import('../types/models.js').ForwardingSuggestion>}
 */
export function suggestForwarding(problemId, payload) {
  return client
    .post(`/admin/problems/${problemId}/suggest-forwarding`, payload)
    .then((res) => res.data)
}

/**
 * The super admin's decision surface: every above-union report awaiting a forward,
 * seat-wide, with the union admins' advice on it. Same row shape as
 * `listMyForwarding`, so the two surfaces can never disagree about what the seat
 * has been advised — but `mySuggestion` is always null here, since the super admin
 * advises on nothing.
 *
 * @returns {Promise<import('../types/models.js').ForwardingItem[]>}
 */
export function listForwardingQueue() {
  return client.get('/super/queue').then((res) => res.data)
}

/**
 * Forward an above-union report to an official.
 *
 * `reason` is required when the choice departs from the advisers' top suggestion OR
 * from the official the reporter pointed the report at — and when those two
 * disagree, no choice satisfies both, so a reason is always required. THE BACKEND
 * DECIDES THAT, not the client: the UI may mirror the rule to mark the field, but a
 * 400 `reason_required` is the authority (CLAUDE.md A.5.7).
 *
 * @param {string} problemId
 * @param {{officialId: string, priority?: string, deadline?: string, reason?: string}} payload
 * @returns {Promise<import('../types/models.js').Assignment>}
 */
export function forwardProblem(problemId, payload) {
  return client.post(`/super/problems/${problemId}/forward`, payload).then((res) => res.data)
}
