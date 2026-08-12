import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as notificationApi from '../lib/api/notifications.js'
import { useAuth } from '../auth/useAuth.js'

/**
 * Keyed by the caller's id although neither endpoint takes a parameter — the
 * third instance of the pattern A.5.5 rule 2 documents, and the one where it
 * matters most.
 *
 * These are PRIVATE messages. Without the key, logging out and back in as a
 * different account on the same tab would serve the previous person's
 * notifications straight from cache. Do NOT "simplify" these to constant keys.
 *
 * `all` is the invalidation prefix: it matches both the list and the count, so
 * the badge and the page can never disagree after a write.
 */
export const notificationKeys = {
  all: ['notifications'],
  mine: (userId) => ['notifications', 'list', userId ?? ''],
  unread: (userId) => ['notifications', 'unread', userId ?? ''],
}

/**
 * refetchOnWindowFocus is TRUE here and NOWHERE ELSE in the app.
 *
 * The global default in lib/queryClient.js is false, deliberately — most screens
 * are records that do not change while you look away. A notification list is the
 * opposite: its whole job is to be current, and the app has no websocket and
 * deliberately no refetchInterval (a poll on every open tab, for a seat this
 * size, buys freshness nobody asked for at a cost paid on every page). Coming
 * back to the tab is the cheapest honest refresh signal there is. Scoped, not
 * global — if this ever moves into queryClient.js it changes every screen.
 *
 * @returns {import('@tanstack/react-query').UseQueryResult<import('../lib/types/models.js').Notification[]>}
 */
export function useNotifications() {
  const { user } = useAuth()
  return useQuery({
    queryKey: notificationKeys.mine(user?.id),
    queryFn: () => notificationApi.listNotifications(),
    enabled: Boolean(user?.id),
    refetchOnWindowFocus: true,
  })
}

/** The bell's badge. Shares its invalidation prefix with the list. */
export function useUnreadCount() {
  const { user } = useAuth()
  return useQuery({
    queryKey: notificationKeys.unread(user?.id),
    queryFn: () => notificationApi.unreadNotificationCount(),
    enabled: Boolean(user?.id),
    refetchOnWindowFocus: true,
  })
}

/**
 * Mark one read.
 *
 * It invalidates rather than writing an optimistic value: the ValidationVote
 * lesson (Part D) is that claiming success the server has not given ends with the
 * UI thanking someone for something that did not happen.
 */
export function useMarkNotificationRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id) => notificationApi.markNotificationRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

/** Clear the badge. */
export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => notificationApi.markAllNotificationsRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

/**
 * Whether a row is still unread. Derived HERE rather than recomputed in the bell,
 * the page and the row, so the three cannot disagree about what "new" means — the
 * isWaiting(row) precedent in useObservations.js.
 *
 * `readAt` is present-and-null while unread, never absent, so `=== null` is the
 * honest check: `!n.readAt` would also swallow a field that had gone missing.
 * @param {import('../lib/types/models.js').Notification} n
 */
export function isUnread(n) {
  return n.readAt === null
}
