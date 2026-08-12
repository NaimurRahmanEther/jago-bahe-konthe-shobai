import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/useAuth.js'
import { useCases, toPublicStatus } from '../../hooks/useCases.js'
import { useScorecard } from '../../hooks/useScorecard.js'
import { useObservations, isWaiting } from '../../hooks/useObservations.js'
import { useDateFiltered } from '../../hooks/useDateRange.js'
import { formatDate } from '../../lib/date.js'
import StatusBadge from '../../components/problem/StatusBadge.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import StatTile from '../../components/ui/StatTile.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

export default function OfficialDashboard() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { data: cases, isLoading, isError, refetch } = useCases()
  // Filtered on DEADLINE, not on when the case opened: the deadline is the date
  // each row prints and the one an official plans around ("what is due this
  // week"). Filtering a date the row does not show would be unfalsifiable to the
  // person reading it.
  const dates = useDateFiltered(cases, 'deadline')
  // The official's own scorecard powers the summary. useScorecard is already
  // enabled: Boolean(officialId), so it stays idle while a claim is pending.
  const {
    data: stats,
    isLoading: statsLoading,
    isError: statsError,
    refetch: refetchStats,
  } = useScorecard(user?.officialId)

  // B19: how many officials below this one have gone quiet. Same query key as the
  // observation page, so opening the page costs no second request.
  const { data: observed } = useObservations()
  const waitingCount = (observed ?? []).filter(isWaiting).length

  // A registered official whose claim is not yet approved has no officialId (B11
  // sets accounts.official_id only on approval), so useCases stays disabled and
  // returns no data. Without this branch the page renders a blank body that reads
  // as "you have no work" rather than "your claim is pending" — the F13 dead end.
  const claimPending = !user?.officialId

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-h1 font-semibold text-ink">{t('resolution.dashboard.title')}</h1>

      {claimPending && (
        <EmptyState
          title={t('resolution.dashboard.claimPending.title')}
          description={t('resolution.dashboard.claimPending.body')}
        />
      )}

      {/* Summary first, then the cases. The tile labels ("সমাধান হয়েছে", "আটকে আছে")
          repeat the status words below, so the section carries an accessible name
          (CLAUDE.md A.5.3 rule 4). Hidden while a claim is pending — no officialId
          means the scorecard query never runs. */}
      {!claimPending && (
        <section className="flex flex-col gap-3" aria-labelledby="official-summary-title">
          <h2 id="official-summary-title" className="font-semibold text-ink">
            {t('resolution.dashboard.summaryTitle')}
          </h2>
          {statsLoading && <SkeletonList count={1} />}
          {statsError && <ErrorState message={t('scorecard.error')} onRetry={refetchStats} />}
          {!statsLoading && !statsError && stats && (
            <>
              <div className="grid grid-cols-2 gap-3">
                <StatTile value={stats.resolved} label={t('scorecard.resolved')} tone="resolved" />
                <StatTile value={stats.pending} label={t('scorecard.pending')} />
                <StatTile value={stats.blocked} label={t('scorecard.blocked')} tone="blocked" />
                <StatTile value={stats.avgResponseDays} label={t('scorecard.avgResponse')} />
              </div>
              {stats.blocked > 0 && (
                <div className="rounded-card bg-blocked-bg p-3 text-sm text-blocked-fg">
                  {t('scorecard.fairnessNote')}
                </div>
              )}
            </>
          )}
        </section>
      )}

      {/* "A copy surfaces on the monitor's dashboard" (Concept §7) — the link is
          the surfacing. It appears only when somebody below has actually gone
          quiet, so an official who monitors nobody never sees it. */}
      {!claimPending && waitingCount > 0 && (
        <Link
          to="/official/observations"
          className="rounded-card bg-blocked-bg px-4 py-3 text-blocked-fg hover:underline"
        >
          {t('observation.dashboardAlert', { count: waitingCount })}
        </Link>
      )}

      {!claimPending && isLoading && <SkeletonList count={3} />}

      {!claimPending && isError && <ErrorState message={t('resolution.dashboard.error')} onRetry={refetch} />}

      {!claimPending && !isLoading && !isError && cases?.length === 0 && (
        <EmptyState title={t('resolution.dashboard.empty')} />
      )}

      {!claimPending && !isLoading && !isError && cases?.length > 0 && (
        <>
          <DateRangeFilter
            value={dates.range}
            onChange={dates.setRange}
            label={t('filter.date.deadlineDate')}
            direction="future"
          />

          {dates.isActive && (
            <p className="text-meta text-muted">
              {t('filter.date.showing', { shown: dates.visible.length, total: dates.total })}
            </p>
          )}

          {dates.visible.length === 0 ? (
            <EmptyState title={t('filter.date.emptyRange')} />
          ) : (
            <div className="flex flex-col gap-3">
              {dates.visible.map((c, i) => (
                <Link
                  key={c.id}
                  to={`/official/cases/${c.id}`}
                  style={{ animationDelay: `${Math.min(i, 5) * 40}ms` }}
                  className="block rounded-card border border-hairline bg-surface p-4 transition duration-150 ease-standard hover:border-brand hover:shadow-md animate-rise-in"
                >
                  <div className="flex items-start justify-between gap-3">
                    <p className="font-semibold text-ink">{t('resolution.dashboard.caseLabel', { id: c.id })}</p>
                    <StatusBadge status={toPublicStatus(c.status)} className="shrink-0" />
                  </div>
                  <p className="mt-2 text-sm text-muted">
                    {t('resolution.dashboard.deadline', { date: formatDate(c.deadline, 'numeric') })}
                  </p>
                </Link>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
