import { useTranslation } from 'react-i18next'
import { useAdminQueue } from '../../hooks/useAssignments.js'
import { useDateFiltered } from '../../hooks/useDateRange.js'
import QueueItem from '../../components/assignment/QueueItem.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

export default function Queue() {
  const { t } = useTranslation()
  const { data: queue, isLoading, isError, refetch } = useAdminQueue()
  const dates = useDateFiltered(queue, 'createdAt')

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-h1 font-semibold text-ink">{t('assignment.queue.title')}</h1>

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('assignment.queue.error')} onRetry={refetch} />}

      {!isLoading && !isError && queue?.length === 0 && <EmptyState title={t('assignment.queue.empty')} />}

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
