import { client } from './client.js'

/**
 * The signed-in caller's own in-app notifications, newest first (B21).
 *
 * Takes NO parameter: the recipient is the caller, read from the JWT — the same
 * rule as listMyProblems and listObservations. A recipientId here would be an
 * index of what a named person is being told about their own reports, and the
 * authorship graph is a safety matter rather than a preference (A.3.2 rule 1).
 *
 * The backend derives BOTH of the caller's identities from the token: their
 * account id, and — for an official — their directory office id, which is what a
 * case notification is addressed to.
 * @returns {Promise<import('../types/models.js').Notification[]>}
 */
export function listNotifications() {
  return client.get('/me/notifications').then((res) => res.data)
}

/**
 * The bell's badge.
 *
 * A separate call from the list because the bell mounts on every page: making it
 * fetch fifty rows would put that payload behind every navigation, and would make
 * the badge wrong the moment the list hit its cap.
 * @returns {Promise<import('../types/models.js').UnreadCount>}
 */
export function unreadNotificationCount() {
  return client.get('/me/notifications/unread-count').then((res) => res.data)
}

/**
 * Mark one notification read. 204, and idempotent — a double-tap is a no-op that
 * still succeeds rather than a 404.
 *
 * Someone else's id answers 404, never 403: an id that exists and belongs to
 * another person must be indistinguishable from one that does not exist.
 * @param {string} id
 * @returns {Promise<void>}
 */
export function markNotificationRead(id) {
  return client.post(`/me/notifications/${id}/read`).then((res) => res.data)
}

/**
 * Clear the whole badge. An EXPLICIT action, never a side effect of opening the
 * list — reading the page to see what is waiting must not destroy the record of
 * what was new.
 * @returns {Promise<{marked: number}>}
 */
export function markAllNotificationsRead() {
  return client.post('/me/notifications/read-all').then((res) => res.data)
}
