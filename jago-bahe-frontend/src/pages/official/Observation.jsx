import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/useAuth.js'
import { useObservations, useNoteObservation, isWaiting } from '../../hooks/useObservations.js'
import { useDateFiltered } from '../../hooks/useDateRange.js'
import ObservationRow from '../../components/official/ObservationRow.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

/**
 * পর্যবেক্ষণ — the officials below this one, and whether any of them has gone
 * quiet on a case (B19).
 *
 * Two things reach this page and the difference is the design (Concept §7):
 * every case this official is the DIRECT monitor of, from the moment it was
 * assigned — monitoring is continuous so that a higher authority can never later
 * claim they did not know — and cases from further down the ladder, but only once
 * silence has climbed to them.
 *
 * It is a watching surface. The one control is a note; nothing here reassigns or
 * closes a case, because silence moves visibility up the ladder and leaves the
 * work where it belongs.
 */
export default function Observation() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { data: rows, isLoading, isError, refetch } = useObservations()
  const note = useNoteObservation()

  // Same branch as OfficialDashboard: an unapproved claim means no officialId, so
  // the query never runs and an empty page would read as "nobody has gone quiet".
  const claimPending = !user?.officialId

  // Deadline, matching OfficialDashboard and the date each row already prints.
  // The waiting/watching split is applied AFTER, so a chosen span narrows both
  // sections and neither can silently keep rows the other has dropped.
  const dates = useDateFiltered(rows, 'deadline')
  const waiting = dates.visible.filter(isWaiting)
  const watching = dates.visible.filter((r) => !isWaiting(r))

  const rowProps = {
    onNote: (payload) => note.mutate(payload),
    isNoting: note.isPending,
    noteFailed: note.isError,
  }

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-h1 font-semibold text-ink">{t('observation.title')}</h1>
      <p className="text-muted">{t('observation.subtitle')}</p>

      {claimPending && (
        <EmptyState
          title={t('resolution.dashboard.claimPending.title')}
          description={t('resolution.dashboard.claimPending.body')}
        />
      )}

      {!claimPending && isLoading && <SkeletonList count={3} />}
      {!claimPending && isError && <ErrorState message={t('observation.error')} onRetry={refetch} />}
      {!claimPending && !isLoading && !isError && rows?.length === 0 && (
        <EmptyState title={t('observation.empty.title')} description={t('observation.empty.body')} />
      )}

      {!claimPending && !isLoading && !isError && rows?.length > 0 && (
        <DateRangeFilter
          value={dates.range}
          onChange={dates.setRange}
          label={t('filter.date.deadlineDate')}
          direction="future"
        />
      )}

      {!claimPending && !isLoading && !isError && rows?.length > 0 && dates.visible.length === 0 && (
        <EmptyState title={t('filter.date.emptyRange')} />
      )}

      {!claimPending && !isLoading && !isError && waiting.length > 0 && (
        <section className="flex flex-col gap-3" aria-labelledby="observation-waiting-title">
          <h2 id="observation-waiting-title" className="font-semibold text-ink">
            {t('observation.waitingTitle')}
          </h2>
          {waiting.map((row) => (
            <ObservationRow key={row.caseId} row={row} {...rowProps} />
          ))}
        </section>
      )}

      {!claimPending && !isLoading && !isError && watching.length > 0 && (
        <section className="flex flex-col gap-3" aria-labelledby="observation-watching-title">
          <h2 id="observation-watching-title" className="font-semibold text-ink">
            {t('observation.watchingTitle')}
          </h2>
          <p className="text-meta text-muted">{t('observation.watchingHint')}</p>
          {watching.map((row) => (
            <ObservationRow key={row.caseId} row={row} {...rowProps} />
          ))}
        </section>
      )}
    </div>
  )
}
