import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMyProblems } from '../../hooks/useProblems.js'
import { useDateRange } from '../../hooks/useDateRange.js'
import { useAuth } from '../../auth/useAuth.js'
import MyReportRow from '../../components/problem/MyReportRow.jsx'
import Button from '../../components/ui/Button.jsx'
import StatTile from '../../components/ui/StatTile.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

/**
 * The resident's home: what they reported, and how it is going.
 *
 * Until this page, a resident who filed a report had nowhere to find it again once
 * it was in the feed alongside everyone else's.
 */
export default function MyReports() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { data: problems, isLoading, isError, refetch } = useMyProblems()
  const dates = useDateRange()

  // The date range narrows the LIST; the tiles stay all-time. They are a summary
  // of everything this resident has ever filed, and making them follow the filter
  // would turn "total" into a second copy of the row count directly beneath it.
  const visible = useMemo(() => (problems ?? []).filter((p) => dates.inRange(p.createdAt)), [problems, dates])

  // Counting the rows the server returned is presentation, not authority: these
  // are states, never anything verdict-shaped. A resident's own page must not
  // grade the officials working their reports (Concept §10).
  const stats = {
    total: problems?.length ?? 0,
    // PendingApproval: the report is with its union admin, awaiting the screening
    // decision that publishes it. This is the first thing that happens to a new
    // report, so it is the stat the resident most needs to see.
    awaitingReview: problems?.filter((p) => p.status === 'PendingApproval').length ?? 0,
    inProgress: problems?.filter((p) => ['Assigned', 'InProgress', 'Blocked', 'Reopened', 'Done'].includes(p.status)).length ?? 0,
    resolved: problems?.filter((p) => p.status === 'Resolved').length ?? 0,
  }

  return (
    <div className="flex w-full flex-col gap-5">
      <div className="flex flex-col gap-0.5">
        <h1 className="text-balance text-h1 font-semibold text-ink">{t('home.greeting', { name: user?.name })}</h1>
        <p className="text-meta text-muted">{t('profile.subtitle')}</p>
      </div>

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('profile.error')} onRetry={refetch} />}

      {!isLoading && !isError && problems?.length === 0 && (
        <EmptyState
          title={t('profile.empty.title')}
          action={
            <Link to="/report">
              <Button>{t('profile.empty.action')}</Button>
            </Link>
          }
        />
      )}

      {!isLoading && !isError && problems?.length > 0 && (
        <>
          {/* A named region: several of these tile labels are also badge labels on
              the rows below (A.5.3 rule 4), so the landmark is what lets a reader —
              and a test — tell the tile from the badge. */}
          <section aria-labelledby="profile-stats-heading" className="flex flex-col gap-2">
            <h2 id="profile-stats-heading" className="sr-only">
              {t('profile.stats.title')}
            </h2>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <StatTile value={stats.total} label={t('profile.stats.total')} tone="brand" />
              <StatTile value={stats.awaitingReview} label={t('profile.stats.awaitingReview')} />
              <StatTile value={stats.inProgress} label={t('profile.stats.inProgress')} />
              <StatTile value={stats.resolved} label={t('profile.stats.resolved')} tone="resolved" />
            </div>
          </section>

          <DateRangeFilter value={dates.range} onChange={dates.setRange} label={t('filter.date.reportDate')} />

          {dates.isActive && (
            <p className="text-meta text-muted">
              {t('filter.date.showing', { shown: visible.length, total: problems.length })}
            </p>
          )}

          {visible.length === 0 ? (
            <EmptyState title={t('filter.date.emptyRange')} />
          ) : (
            <div className="flex flex-col gap-3">
              {visible.map((problem, i) => (
                <MyReportRow key={problem.id} problem={problem} index={i} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
