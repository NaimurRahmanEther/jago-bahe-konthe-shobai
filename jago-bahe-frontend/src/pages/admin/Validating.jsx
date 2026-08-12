import { useTranslation } from 'react-i18next'
import { useValidatingQueue } from '../../hooks/useAssignments.js'
import { useDateFiltered } from '../../hooks/useDateRange.js'
import QueueItem from '../../components/assignment/QueueItem.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

/**
 * Approved reports the community is still validating — the window the admin had
 * no view of before B17. Rows are ordered closest-to-threshold first and link
 * straight to the assign screen: the count is evidence, not a gate, so any of
 * these may be forwarded to an official at any moment (CLAUDE.md A.3.1).
 *
 * Deliberately the same four-branch shape as Queue.jsx rather than a new pattern.
 */
export default function Validating() {
  const { t } = useTranslation()
  const { data: queue, isLoading, isError, refetch } = useValidatingQueue()
  const dates = useDateFiltered(queue, 'createdAt')

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-h1 font-semibold text-ink">{t('validating.title')}</h1>
      <p className="text-meta text-muted">{t('validating.subtitle')}</p>

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('validating.error')} onRetry={refetch} />}

      {!isLoading && !isError && queue?.length === 0 && <EmptyState title={t('validating.empty')} />}

      {!isLoading && !isError && queue?.length > 0 && (
        <>
          <DateRangeFilter value={dates.range} onChange={dates.setRange} label={t('filter.date.reportDate')} />

          {dates.isActive && (
            <p className="text-meta text-muted">
              {t('filter.date.showing', { shown: dates.visible.length, total: dates.total })}
            </p>
          )}

          {dates.visible.length === 0 ? (
            <EmptyState title={t('filter.date.emptyRange')} />
          ) : (
            <div className="flex flex-col gap-3">
              {dates.visible.map((item, i) => (
                <QueueItem key={item.problemId} item={item} index={i} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
