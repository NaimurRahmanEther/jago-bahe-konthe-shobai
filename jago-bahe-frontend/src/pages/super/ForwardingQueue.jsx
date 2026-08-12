import { useTranslation } from 'react-i18next'
import { useForwardingQueue } from '../../hooks/useAssignments.js'
import { useDateFiltered } from '../../hooks/useDateRange.js'
import ForwardingItem from '../../components/assignment/ForwardingItem.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

/**
 * The super admin's decision surface: every above-union report awaiting a forward,
 * seat-wide, most-advised first.
 *
 * This is the role's ONLY action over problems, and it is not an override. The
 * super admin still cannot reverse a union admin's decision, and no endpoint for
 * that exists or may be added — forwarding is a FRESH decision on a report no union
 * admin was entitled to decide in the first place (A.3.8). /super stays the
 * read-only oversight feed for exactly that reason.
 */
export default function ForwardingQueue() {
  const { t } = useTranslation()
  const { data: rows, isLoading, isError, refetch } = useForwardingQueue()
  // A filter, never a control: narrowing what this page shows is not a decision
  // about any report, so /super/queue keeps holding exactly one action.
  const dates = useDateFiltered(rows, 'createdAt')

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-h1 font-semibold text-ink">{t('forwarding.super.title')}</h1>
        <p className="text-sm text-muted">{t('forwarding.super.subtitle')}</p>
      </div>

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('forwarding.super.error')} onRetry={refetch} />}

      {!isLoading && !isError && rows?.length === 0 && <EmptyState title={t('forwarding.super.empty')} />}

      {!isLoading && !isError && rows?.length > 0 && (
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
                <ForwardingItem
                  key={item.problemId}
                  item={item}
                  href={`/super/problems/${item.problemId}/forward`}
                  index={i}
                />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
