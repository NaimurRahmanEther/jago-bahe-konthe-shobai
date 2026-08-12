import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import StatusBadge from '../problem/StatusBadge.jsx'
import ValidationBar from '../ui/ValidationBar.jsx'
import { useOfficials } from '../../hooks/useOfficials.js'

/**
 * One above-union report awaiting a forward — the SAME row on both surfaces, the
 * union admin's advisory list and the super admin's decision queue, because the
 * two must never disagree about what the seat has been advised (the backend serves
 * them one DTO for exactly this reason).
 *
 * What differs is only the affordance: the admin's row leads to a screen where
 * they advise, the super admin's to one where they decide. Neither is derived from
 * the role — the caller passes `href`, so this component never has to know who is
 * looking at it.
 *
 * @param {{
 *   item: import('../../lib/types/models.js').ForwardingItem,
 *   href: string,
 *   index?: number,
 * }} props
 */
export default function ForwardingItem({ item, href, index = 0 }) {
  const { t } = useTranslation()
  const { data: officials } = useOfficials()

  // The directory is bounded (one seat) and already cached under ['officials'], so
  // resolving names here costs nothing and keeps ids off the payload (A.3.4).
  const nameOf = (id) => officials?.find((o) => o.id === id)?.name
  const pointedName = nameOf(item.pointedOfficialId)
  const topName = nameOf(item.topOfficialId)
  const adviceCount = item.suggestions?.length ?? 0

  return (
    <Link
      to={href}
      style={{ animationDelay: `${Math.min(index, 5) * 40}ms` }}
      className="block rounded-card border border-hairline bg-surface p-4 transition duration-150 ease-standard hover:border-brand hover:shadow-md animate-rise-in"
    >
      <div className="flex items-start justify-between gap-3">
        <h3 className="text-title font-semibold leading-snug text-ink">{item.title}</h3>
        <StatusBadge status={item.status} className="shrink-0" />
      </div>

      <p className="mt-2 text-meta text-muted">
        {item.address}
        {pointedName && ` · ${t('forwarding.pointedShort', { name: pointedName })}`}
      </p>

      <ValidationBar count={item.validCount} threshold={item.validationThreshold} className="mt-3" />

      {/* The tally, in words rather than a second progress bar: there is nothing to
          fill toward. Advice has no quorum and no target — the super admin may
          forward at any count, including none (A.3.8) — so a bar would draw a
          finish line that does not exist, which is the same mistake the public
          validation bar was corrected for in F21. */}
      <p className="mt-2 text-meta font-medium text-brand-dark">
        {adviceCount === 0
          ? t('forwarding.noAdvice')
          : topName
            ? t('forwarding.tallyTop', { count: item.topCount, total: adviceCount, name: topName })
            : t('forwarding.tallyTied', { count: adviceCount })}
      </p>
    </Link>
  )
}
