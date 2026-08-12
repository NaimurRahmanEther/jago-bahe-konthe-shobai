import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { isUnread } from '../../hooks/useNotifications.js'
import { formatDate } from '../../lib/date.js'

/**
 * One in-app notification.
 *
 * The whole row is a single <Link> to the PUBLIC problem page. A <button> inside
 * an <a> is invalid HTML and breaks keyboard navigation (A.3.2.1 rule 1), so
 * marking read fires from the anchor's own onClick rather than from a nested
 * control — and the row keeps its full-size tap target.
 *
 * The unread marker is a brand dot PLUS the word "নতুন". Colour is never the only
 * signal (Guideline §2 rule 1), and the colour is brand teal rather than red:
 * both reds are taken — bright is Reopened, muted maroon is Rejected — and a
 * waiting message is not an adverse verdict.
 *
 * @param {{
 *   notification: import('../../lib/types/models.js').Notification,
 *   onRead: (id: string) => void,
 * }} props
 */
export default function NotificationRow({ notification, onRead }) {
  const { t } = useTranslation()
  const unread = isUnread(notification)

  return (
    <Link
      to={`/problems/${notification.problemId}`}
      onClick={() => {
        if (unread) onRead(notification.id)
      }}
      className={`block rounded-card border bg-surface p-4 transition-shadow duration-150 ease-standard hover:border-brand/40 hover:shadow-md ${
        unread ? 'border-brand/40' : 'border-hairline'
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <p className="text-title font-semibold text-ink">
          {notification.problemTitle || t('notification.untitled')}
        </p>
        {unread && (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-brand-tint px-2 py-0.5 text-micro font-medium text-brand-dark">
            <span className="h-1.5 w-1.5 rounded-full bg-brand" aria-hidden="true" />
            {t('notification.unread')}
          </span>
        )}
      </div>

      <p className="mt-1 text-ink">{sentence(notification, t)}</p>

      <p className="mt-2 text-meta text-muted tabular-nums">{formatDate(notification.createdAt)}</p>
    </Link>
  )
}

/**
 * The Bangla sentence for one notification.
 *
 * `detail` is type-specific and is ONLY ever read inside this switch — that is
 * the whole reason the branch exists rather than one interpolated key. An unknown
 * type falls back to the title alone rather than rendering a raw key: the backend
 * whitelists eight in its CHECK constraint, so a ninth reaching here means the
 * two halves have drifted and the honest thing is to say less, not to shout.
 */
function sentence(n, t) {
  switch (n.type) {
    case 'problem_rejected':
      // The ground is the enum the admin was permitted to reject on, and the four
      // already have Bangla labels from the moderation slice — reuse them rather
      // than writing a second set that can disagree.
      return t('notification.type.problem_rejected', {
        ground: t(`problem.rejectionReason.${n.detail}`, n.detail),
      })
    case 'problem_assigned':
      return t('notification.type.problem_assigned', { name: n.detail })
    case 'case_assigned':
      return t('notification.type.case_assigned', { date: formatDate(n.detail) })
    case 'obstacle_declared':
      return t('notification.type.obstacle_declared', { who: n.detail })
    case 'obstacle_adjudicated':
      // One type, two opposite outcomes: confirmed moves responsibility up and
      // protects the official; denied bounces the work straight back to them.
      return t(
        n.detail === 'denied'
          ? 'notification.type.obstacle_adjudicated.denied'
          : 'notification.type.obstacle_adjudicated.confirmed',
      )
    case 'problem_approved':
    case 'confirmation_requested':
    case 'case_reopened':
      return t(`notification.type.${n.type}`)
    default:
      return ''
  }
}
