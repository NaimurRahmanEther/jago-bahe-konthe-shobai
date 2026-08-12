import { useTranslation } from 'react-i18next'
import { useMyForwarding } from '../../hooks/useAssignments.js'
import { useDateFiltered } from '../../hooks/useDateRange.js'
import ForwardingItem from '../../components/assignment/ForwardingItem.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

/**
 * The union admin's advisory list: above-union reports they may say where to send.
 *
 * These are deliberately absent from /admin/queue — an above-union report is the
 * super admin's to forward, so it is not on the list of what this admin may assign
 * (A.3.8). This list is also scoped differently: by ADVICE SCOPE (the upazila, or
 * the seat), not by the admin's own union, so a report from a neighbouring union
 * appears here though nothing of it appears in the assignment queue.
 */
export default function Forwarding() {
  const { t } = useTranslation()
  const { data: rows, isLoading, isError, refetch } = useMyForwarding()
  const dates = useDateFiltered(rows, 'createdAt')

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-h1 font-semibold text-ink">{t('forwarding.admin.title')}</h1>
        <p className="text-sm text-muted">{t('forwarding.admin.subtitle')}</p>
      </div>

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('forwarding.admin.error')} onRetry={refetch} />}

      {!isLoading && !isError && rows?.length === 0 && <EmptyState title={t('forwarding.admin.empty')} />}

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
                  href={`/admin/forwarding/${item.problemId}`}
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
