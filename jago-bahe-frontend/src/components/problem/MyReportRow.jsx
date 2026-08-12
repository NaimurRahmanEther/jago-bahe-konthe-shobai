import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import ProblemCard from './ProblemCard.jsx'
import ReporterActions from './ReporterActions.jsx'
import ProgressPanel from '../resolution/ProgressPanel.jsx'
import { useProgress } from '../../hooks/useCases.js'
// Shared with the public problem detail page, which asks the same question. Two
// copies of this list is how the two views drift apart — see lib/problemStatus.js.
import { hasCase } from '../../lib/problemStatus.js'

/** @param {{problem: import('../../lib/types/models.js').Problem, index?: number}} props */
export default function MyReportRow({ problem, index = 0 }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)

  // Lazy: one query per *opened* row, not one per row on mount.
  const { data: progress, isLoading, isError, refetch } = useProgress(problem.id, {
    enabled: expanded && hasCase(problem.status),
  })

  return (
    <div className="flex flex-col">
      <ProblemCard problem={problem} index={index} />

      {/* Edit while unvalidated, withdraw before assignment, delete at any time —
          this is the reporter's own list, the natural home for managing a report.
          onDeleted is a no-op on purpose: we are already on /me, and the row simply
          disappears when the list query refetches. Without it the component would
          navigate to the page we are on. */}
      <ReporterActions problem={problem} className="mt-2" onDeleted={() => {}} />

      {problem.status === 'Rejected' && problem.rejectionReason && (
        <p className="mt-2 rounded-control bg-rejected-bg px-3 py-2 text-sm text-rejected-fg">
          {t('problem.detail.rejectedReason')}: {t(`problem.rejectionReason.${problem.rejectionReason}`)}
        </p>
      )}

      {problem.status === 'Done' && (
        <p className="mt-2 rounded-control bg-done-bg px-3 py-2 text-sm text-done-fg">
          {t('profile.awaitingConfirmation')}
        </p>
      )}

      {hasCase(problem.status) && (
        <>
          <button
            type="button"
            onClick={() => setExpanded((v) => !v)}
            aria-expanded={expanded}
            className="mt-1 min-h-11 self-start px-1 text-left text-sm font-medium text-brand hover:text-brand-dark"
          >
            {expanded ? t('progress.hide') : t('progress.show')}
          </button>
          {expanded && (
            <ProgressPanel progress={progress} isLoading={isLoading} isError={isError} onRetry={refetch} />
          )}
        </>
      )}
    </div>
  )
}
