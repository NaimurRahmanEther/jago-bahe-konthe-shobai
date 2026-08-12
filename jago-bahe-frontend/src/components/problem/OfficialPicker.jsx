import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useOfficials } from '../../hooks/useOfficials.js'

/**
 * @param {{
 *   value: string,
 *   onChange: (officialId: string) => void,
 *   filter?: (official: import('../../lib/types/models.js').Official) => boolean,
 *   label?: string,
 * }} props
 *
 * `filter` narrows the pickable set — the super admin's forward screen passes one
 * so only above-union offices can be chosen, since forwarding INTO a union would
 * take a report that union's own admin should have decided. It is a convenience,
 * not a guard: the backend refuses a union-level target with `wrong_route`
 * regardless (CLAUDE.md A.5.7).
 */
export default function OfficialPicker({ value, onChange, filter, label }) {
  const { t } = useTranslation()
  const { data: officials, isLoading } = useOfficials()
  const [query, setQuery] = useState('')

  const pickable = useMemo(
    () => (filter ? (officials ?? []).filter(filter) : (officials ?? [])),
    [officials, filter],
  )

  const selected = useMemo(() => pickable.find((o) => o.id === value) ?? null, [pickable, value])

  const matches = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return pickable
    return pickable.filter((o) => o.name.toLowerCase().includes(q))
  }, [pickable, query])

  if (selected && !query) {
    return (
      <div className="flex flex-col gap-2">
        <label className="font-medium text-ink">{label ?? t('problem.report.official')}</label>
        <div className="flex min-h-11 items-center justify-between gap-2 rounded-control border border-hairline bg-surface px-3">
          <span className="flex items-center gap-2 text-sm text-ink">
            {selected.name}
            <span className="rounded-full bg-brand-tint px-2 py-0.5 text-xs font-medium text-brand-dark">
              {t(`problem.tier.${selected.tier}`)}
            </span>
          </span>
          <button type="button" onClick={() => onChange('')} className="min-h-11 text-sm font-medium text-brand">
            {t('problem.report.change')}
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <label htmlFor="official-search" className="font-medium text-ink">
        {label ?? t('problem.report.official')}
      </label>
      <input
        id="official-search"
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t('problem.report.officialSearchPlaceholder')}
        className="min-h-11 w-full rounded-control border border-hairline bg-surface px-3 text-ink focus-visible:outline-2 focus-visible:outline-brand"
      />
      {isLoading && <p className="text-sm text-muted">{t('common.loading')}</p>}
      {!isLoading && (
        <ul className="max-h-48 overflow-y-auto rounded-control border border-hairline bg-surface">
          {matches.map((o) => (
            <li key={o.id}>
              <button
                type="button"
                onClick={() => {
                  onChange(o.id)
                  setQuery('')
                }}
                className="flex min-h-11 w-full items-center justify-between gap-2 px-3 text-left text-sm hover:bg-canvas"
              >
                <span>{o.name}</span>
                <span className="rounded-full bg-brand-tint px-2 py-0.5 text-xs font-medium text-brand-dark">
                  {t(`problem.tier.${o.tier}`)}
                </span>
              </button>
            </li>
          ))}
          {matches.length === 0 && <li className="px-3 py-2 text-sm text-muted">{t('common.empty')}</li>}
        </ul>
      )}
    </div>
  )
}
