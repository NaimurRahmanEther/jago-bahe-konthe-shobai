import { useTranslation } from 'react-i18next'
import { useOfficials } from '../../hooks/useOfficials.js'

/**
 * The union admins' advice on one above-union report, in full: who advised, whom
 * they named, and why.
 *
 * THE ADVISER IS NAMED, deliberately. The admin vote this replaced was a secret
 * ballot, and that secrecy was earned by the ballot's power — a vote that BINDS has
 * a claim to protection from pressure. Advice binds nobody: it settles nothing, and
 * the super admin may forward against all of it. What it does instead is put the
 * seat's reasoning on the record, and reasoning nobody will stand behind is not
 * worth recording (A.3.8).
 *
 * Shown on both panels, so the adviser and the decider are looking at exactly the
 * same evidence.
 *
 * @param {{
 *   suggestions: import('../../lib/types/models.js').ForwardingSuggestion[],
 *   topOfficialId?: string,
 * }} props
 */
export default function AdviceTally({ suggestions, topOfficialId }) {
  const { t } = useTranslation()
  const { data: officials } = useOfficials()

  if (!suggestions || suggestions.length === 0) {
    return <p className="text-sm text-muted">{t('forwarding.noAdvice')}</p>
  }

  return (
    <ul className="flex flex-col gap-2">
      {suggestions.map((s) => {
        const official = officials?.find((o) => o.id === s.officialId)
        // Marked, never sorted to the top: the list stays in the order the advice
        // arrived, so the record reads as what happened rather than as a ranking.
        const isTop = Boolean(topOfficialId) && s.officialId === topOfficialId
        return (
          <li
            key={s.id}
            className={`rounded-control border p-3 ${
              isTop ? 'border-brand bg-brand-tint' : 'border-hairline bg-surface'
            }`}
          >
            <p className="text-sm font-medium text-ink">
              {official?.name ?? s.officialId}
              {official && (
                <span className="ml-2 font-normal text-muted">
                  ({t(`problem.tier.${official.tier}`)})
                </span>
              )}
              {isTop && (
                <span className="ml-2 rounded-full bg-surface px-2 py-0.5 text-micro font-medium text-brand-dark">
                  {t('forwarding.topChip')}
                </span>
              )}
            </p>
            {s.reason && <p className="mt-1 text-meta text-muted">{s.reason}</p>}
          </li>
        )
      })}
    </ul>
  )
}
