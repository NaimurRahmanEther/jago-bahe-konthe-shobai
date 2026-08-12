import { useTranslation } from 'react-i18next'
import { useAreas } from '../../hooks/useAreas.js'
import ErrorState from '../ui/ErrorState.jsx'
import { SkeletonCard } from '../ui/Skeleton.jsx'

/** @param {{value: import('../../lib/types/models.js').Location, onChange: (next: import('../../lib/types/models.js').Location) => void}} props */
export default function LocationPicker({ value, onChange }) {
  const { t } = useTranslation()
  // Unions and the pourashava — the pilot seeds no wards, so a location resolves
  // to the local unit that has an admin, and the ward goes in the address line.
  const { data: areas, isLoading, isError, refetch } = useAreas({ level: 'union' })

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <label className="font-medium text-ink">{t('problem.report.location')}</label>

        {isLoading && <SkeletonCard />}

        {isError && <ErrorState message={t('problem.report.areasError')} onRetry={refetch} />}

        {/* Without this branch a seat with no areas renders an empty row of nothing —
            the reporter would see a label above a void and no way to know why. */}
        {!isLoading && !isError && areas?.length === 0 && (
          <p className="text-sm text-muted">{t('problem.report.areasEmpty')}</p>
        )}

        {!isLoading && !isError && areas?.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {areas.map((area) => (
              <button
                key={area.id}
                type="button"
                onClick={() => onChange({ ...value, areaId: area.id })}
                aria-pressed={value.areaId === area.id}
                className={`min-h-11 rounded-control border px-3 text-sm font-medium ${
                  value.areaId === area.id
                    ? 'border-brand bg-brand-tint text-brand-dark'
                    : 'border-hairline bg-surface text-ink'
                }`}
              >
                {area.name}
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="address" className="font-medium text-ink">
          {t('problem.report.address')}
        </label>
        <input
          id="address"
          type="text"
          value={value.address ?? ''}
          onChange={(e) => onChange({ ...value, address: e.target.value })}
          placeholder={t('problem.report.addressPlaceholder')}
          className="min-h-11 w-full rounded-control border border-hairline bg-surface px-3 text-ink focus-visible:outline-2 focus-visible:outline-brand"
        />
      </div>
    </div>
  )
}
