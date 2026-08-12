/**
 * Status vocabulary shared by the screens that reason about it.
 *
 * The backend owns these names — problem/domain/status.go is the authority, and
 * models.js's ProblemStatus typedef mirrors it. What lives here is the small
 * DERIVED sets more than one screen needs, kept in one place so two screens cannot
 * quietly disagree about the same question.
 */

/**
 * The statuses that can have a case behind them.
 *
 * A case is materialized when a problem is assigned, so these are the only
 * statuses for which asking about progress can answer anything but "nothing yet".
 * Both the public problem detail page and the reporter's own row gate their
 * progress read on this: the endpoint is null-shaped and would answer harmlessly
 * either way, but a request that can only ever return null is one worth not making.
 *
 * Shared rather than copied because a second copy is how the two views drift —
 * the same reasoning A.3.1.1 constraint 1 gives for not letting the assignment
 * context keep its own copy of the problem context's vocabulary.
 */
export const HAS_CASE = ['Assigned', 'InProgress', 'Blocked', 'Done', 'Resolved', 'Reopened']

/** @param {string} status */
export function hasCase(status) {
  return HAS_CASE.includes(status)
}
