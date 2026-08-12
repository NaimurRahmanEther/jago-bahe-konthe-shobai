import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useOfficials } from '../../hooks/useOfficials.js'
import { useAreas } from '../../hooks/useAreas.js'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

const TIER_ORDER = [
  'ward_member',
  'women_member',
  'pourashava_councillor',
  'union_chairman',
  'pourashava_mayor',
  'upazila_chairman',
  'upazila_vice_chairman',
  'mp',
  'minister',
]

export default function OfficialDirectory() {
  const { t } = useTranslation()
  const { data: officials, isLoading, isError, refetch } = useOfficials()
  // The seat's geography names the area an official holds. Not a third loading
  // state: the row reads fine without it, so an unresolved id simply prints
  // nothing rather than 'ward-1' or a spinner beside a name.
  const { data: areas } = useAreas()
  const areaName = (id) => areas?.find((a) => a.id === id)?.name

  return (
    <div className="flex w-full flex-col gap-4">
      <h1 className="text-h1 font-semibold text-ink">{t('directory.title')}</h1>

      {isLoading && <SkeletonList count={4} />}

      {isError && <ErrorState message={t('directory.error')} onRetry={refetch} />}

      {!isLoading && !isError && officials?.length === 0 && <EmptyState title={t('directory.empty')} />}

      {!isLoading && !isError && officials?.length > 0 && (
        <div className="flex flex-col gap-6">
          {TIER_ORDER.filter((tier) => officials.some((o) => o.tier === tier)).map((tier) => (
            <div key={tier}>
              <h2 className="flex items-center gap-2 text-micro font-semibold uppercase tracking-wider text-muted after:h-px after:flex-1 after:bg-hairline after:content-['']">
                {t(`problem.tier.${tier}`)}
              </h2>
              <div className="mt-2 flex flex-col gap-2">
                {officials
                  .filter((o) => o.tier === tier)
                  .map((o) => (
                    <Link
                      key={o.id}
                      to={`/officials/${o.id}/scorecard`}
                      className="group flex animate-fade-in items-center gap-3 rounded-card border border-hairline bg-surface p-3 transition duration-150 ease-standard hover:border-brand hover:shadow-md"
                    >
                      {/* The initial is decorative — the name sits right beside it, so
                          announcing the letter again would just stutter. */}
                      <span
                        aria-hidden="true"
                        className="grid size-10 shrink-0 place-items-center rounded-full bg-brand-tint font-semibold text-brand-dark"
                      >
                        {o.name.trim().charAt(0)}
                      </span>
                      <span className="min-w-0">
                        <span className="block font-medium text-ink transition-colors duration-150 ease-standard group-hover:text-brand-dark">
                          {o.name}
                        </span>
                        {areaName(o.areaId) && <span className="block text-meta text-muted">{areaName(o.areaId)}</span>}
                      </span>
                    </Link>
                  ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
