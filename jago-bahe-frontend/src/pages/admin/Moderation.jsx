import { useTranslation } from 'react-i18next'
import { useModerationQueue, useApproveProblem, useRejectProblem } from '../../hooks/useModeration.js'
import { useDateFiltered } from '../../hooks/useDateRange.js'
import ModerationItem from '../../components/problem/ModerationItem.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

export default function Moderation() {
  const { t } = useTranslation()
  const { data: queue, isLoading, isError, refetch } = useModerationQueue()
  const dates = useDateFiltered(queue, 'createdAt')
  const approve = useApproveProblem()
  const reject = useRejectProblem()

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-h1 font-semibold text-ink">{t('moderation.title')}</h1>
        <p className="text-sm text-muted">{t('moderation.subtitle')}</p>
      </div>

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('moderation.error')} onRetry={refetch} />}

      {!isLoading && !isError && queue?.length === 0 && <EmptyState title={t('moderation.empty')} />}

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
              {dates.visible.map((problem, i) => (
                <ModerationItem
                  key={problem.id}
                  problem={problem}
                  index={i}
                  isSubmitting={approve.isPending || reject.isPending}
                  onApprove={() => approve.mutate(problem.id)}
                  onReject={(payload) => reject.mutate({ problemId: problem.id, ...payload })}
                />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
