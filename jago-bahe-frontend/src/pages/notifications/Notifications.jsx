import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useNotifications,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  isUnread,
} from '../../hooks/useNotifications.js'
import NotificationRow from '../../components/notification/NotificationRow.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import Button from '../../components/ui/Button.jsx'

/**
 * বিজ্ঞপ্তি — the eight things that happen to your reports and your cases which
 * no other page would tell you (B21).
 *
 * It is a per-caller surface, so it takes no parameter and is reachable by every
 * signed-in role: a resident, an official, an admin and the super admin each have
 * their own, derived from the token.
 *
 * Opening this page does NOT mark everything read. That would make a GET mutate —
 * which this codebase does nowhere — and would destroy the page for anyone who
 * opens it to see what is waiting rather than to dismiss it. A row is marked read
 * when it is followed; "সব পড়া হয়েছে" is the explicit escape hatch.
 */
export default function Notifications() {
  const { t } = useTranslation()
  const { data: rows, isLoading, isError, refetch } = useNotifications()
  const markRead = useMarkNotificationRead()
  const markAll = useMarkAllNotificationsRead()

  const unreadRows = (rows ?? []).filter(isUnread)

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-h1 font-semibold text-ink">{t('notification.title')}</h1>
          <p className="text-muted">{t('notification.subtitle')}</p>
        </div>
        {unreadRows.length > 0 && (
          <Button variant="secondary" onClick={() => markAll.mutate()} disabled={markAll.isPending}>
            {markAll.isPending ? t('notification.markingAll') : t('notification.markAll')}
          </Button>
        )}
      </div>

      {markAll.isError && <p className="text-meta text-rejected-fg">{t('notification.markAllError')}</p>}

      {isLoading && <SkeletonList count={4} />}
      {isError && <ErrorState message={t('notification.error')} onRetry={refetch} />}
      {!isLoading && !isError && rows?.length === 0 && (
        <EmptyState
          title={t('notification.empty.title')}
          description={t('notification.empty.body')}
          action={
            <Link
              to="/problems"
              className="inline-flex min-h-11 items-center rounded-control bg-brand px-4 text-surface hover:bg-brand-dark"
            >
              {t('notification.empty.action')}
            </Link>
          }
        />
      )}

      {!isLoading && !isError && rows && rows.length > 0 && (
        <section className="flex flex-col gap-3" aria-labelledby="notifications-list-title">
          <h2 id="notifications-list-title" className="font-semibold text-ink">
            {t('notification.sectionTitle')}
          </h2>
          {rows.map((n) => (
            <NotificationRow key={n.id} notification={n} onRead={(id) => markRead.mutate(id)} />
          ))}
        </section>
      )}
    </div>
  )
}
