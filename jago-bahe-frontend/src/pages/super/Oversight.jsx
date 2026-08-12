import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useOversight } from '../../hooks/useOversight.js'
import { useForwardingQueue } from '../../hooks/useAssignments.js'
import AuditTrail from '../../components/resolution/AuditTrail.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

// A read-only feed of moderator decisions across the seat. There is deliberately
// NO action here — the role oversees union admins, it does not override them
// (Scaffold §2, "Oversight, not override"). Any button that changed a decision
// would break the one rule this page exists to embody.
//
// The forwarding banner below is a LINK, not a control, and that distinction is
// the point: forwarding is a fresh decision on a report no union admin was
// entitled to decide, so it lives on its own page (/super/queue) rather than
// putting the seat's first super-admin action on the page whose whole argument is
// having none (A.3.8).
export default function Oversight() {
  const { t } = useTranslation()
  const { data: entries, isLoading, isError, refetch } = useOversight()
  const { data: waiting } = useForwardingQueue()

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-h1 font-semibold text-ink">{t('super.oversight.title')}</h1>
        <p className="text-sm text-muted">{t('super.oversight.subtitle')}</p>
      </div>

      {waiting?.length > 0 && (
        <Link
          to="/super/queue"
          className="flex min-h-11 items-center justify-between gap-3 rounded-card border border-brand bg-brand-tint px-4 py-3 transition duration-150 ease-standard hover:shadow-md"
        >
          <span className="font-medium text-brand-dark">
            {t('forwarding.super.waiting', { count: waiting.length })}
          </span>
          <span aria-hidden="true" className="text-brand-dark">
            →
          </span>
        </Link>
      )}

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('super.oversight.error')} onRetry={refetch} />}

      {!isLoading && !isError && entries?.length === 0 && <EmptyState title={t('super.oversight.empty')} />}

      {!isLoading && !isError && entries?.length > 0 && (
        <div className="rounded-card border border-hairline bg-surface p-4">
          <AuditTrail entries={entries} />
        </div>
      )}
    </div>
  )
}
