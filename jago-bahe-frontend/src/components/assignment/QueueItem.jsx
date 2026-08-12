import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import StatusBadge from '../problem/StatusBadge.jsx'
import ValidationBar from '../ui/ValidationBar.jsx'
import { useOfficials } from '../../hooks/useOfficials.js'

/**
 * One row of an admin queue — both the assignment queue and the validating list.
 *
 * The validation bar is here because since B17 the count is what the admin judges
 * by: they decide when a report is trustworthy enough to forward, rather than
 * waiting for it to cross V (CLAUDE.md A.3.1). It is the shared ValidationBar, not
 * a second bar — Guideline §7 allows exactly one.
 *
 * This component read four fields the API never sent (`id`, `location.address`,
 * `pointedOfficial`, `routing`), so any non-empty queue threw on
 * `location.address` and every row linked to `/admin/problems/undefined/assign`.
 * It went unseen because nothing ever reached the queue against the real API. The
 * DTO now carries `problemId`, `address` and `routing`; the official is resolved
 * from the directory the app already caches.
 *
 * Every row here is union-routed. This used to branch on `routing` to send
 * above-union rows to a vote screen; since B20 those reports never reach this
 * queue at all — they are the super admin's to forward, and the backend leaves
 * them off the list of what this admin may assign (CLAUDE.md A.3.8). They appear
 * on `ForwardingItem` instead, where the admin advises.
 *
 * @param {{item: import('../../lib/types/models.js').QueueItem, index?: number}} props
 */
export default function QueueItem({ item, index = 0 }) {
  const { t } = useTranslation()
  const { data: officials } = useOfficials()

  // The directory is bounded (one seat) and already cached under ['officials'] by
  // OfficialPicker, so joining here costs nothing and keeps the queue payload from
  // duplicating names the client already holds.
  const pointedOfficial = officials?.find((o) => o.id === item.pointedOfficialId)

  return (
    <Link
      to={`/admin/problems/${item.problemId}/assign`}
      style={{ animationDelay: `${Math.min(index, 5) * 40}ms` }}
      className="block rounded-card border border-hairline bg-surface p-4 transition duration-150 ease-standard hover:border-brand hover:shadow-md animate-rise-in"
    >
      <div className="flex items-start justify-between gap-3">
        <h3 className="text-title font-semibold leading-snug text-ink">{item.title}</h3>
        <StatusBadge status={item.status} className="shrink-0" />
      </div>
      <p className="mt-2 text-meta text-muted">
        {item.address}
        {pointedOfficial &&
          ` · ${pointedOfficial.name} (${t(`problem.tier.${pointedOfficial.tier}`)})`}
      </p>

      <ValidationBar
        count={item.validCount}
        threshold={item.validationThreshold}
        className="mt-3"
      />

      <p className="mt-2 text-meta font-medium text-brand-dark">
        {t('assignment.queue.unionLevel')}
      </p>
    </Link>
  )
}
