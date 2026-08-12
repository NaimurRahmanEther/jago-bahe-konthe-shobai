import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import UpvoteButton from './UpvoteButton.jsx'
import EmptyState from '../ui/EmptyState.jsx'
import { useUpvoteSuggestion } from '../../hooks/useSuggestions.js'

const ERROR_KEYS = {
  not_verified: 'suggestion.notVerified',
  not_area_resident: 'suggestion.notAreaResident',
}

/**
 * The community's proposed fixes for a problem, ranked by the backend.
 *
 * The top suggestion is what the official is required to answer in their public
 * plan (Concept §4, Figure 3), so it is drawn to look like the thing being put to
 * an authority rather than the first row of a list.
 *
 * Ranking is never recomputed here — `isTop` is backend-owned. Note that TIES ALL
 * CARRY isTop: the service marks every suggestion sharing the highest count, so
 * more than one row can legitimately be highlighted, and nothing here may assume
 * exactly one. A suggestion with zero upvotes is never top, so a problem nobody
 * has upvoted shows no highlight at all — correct, and the reason the highlight
 * can look "broken" on a quiet report.
 *
 * @param {{problemId: string, suggestions: import('../../lib/types/models.js').Suggestion[]}} props
 */
export default function SuggestionList({ problemId, suggestions }) {
  const { t } = useTranslation()
  const upvoteSuggestion = useUpvoteSuggestion(problemId)
  // Only the id currently in flight, so one row's spinner does not disable the
  // rest. Which suggestions the reader has upvoted comes from the server —
  // s.myUpvote — not from here. It used to be a local Set seeded empty on every
  // mount, so a reload re-offered the button on something already upvoted and the
  // toggle then silently WITHDREW it.
  const [pendingId, setPendingId] = useState(null)
  const [error, setError] = useState('')

  async function toggleUpvote(id) {
    setError('')
    setPendingId(id)
    try {
      await upvoteSuggestion.mutateAsync(id)
    } catch (err) {
      setError(t(ERROR_KEYS[err?.code] ?? 'suggestion.upvoteFailed'))
    } finally {
      setPendingId(null)
    }
  }

  if (suggestions.length === 0) {
    return <EmptyState title={t('suggestion.empty')} description={t('suggestion.emptyInvite')} />
  }

  return (
    <>
      {/* Ordinary rows are FLAT — hairline-separated list items, not cards. They used
          to carry `rounded-card border border-hairline bg-surface p-4`, the exact class
          string of the section card containing them, so the page drew identical cards
          inside identical cards with no nesting cue. Reserving the card for the top
          suggestion is what makes it read as elevated rather than merely tinted. */}
      <ul className="flex flex-col gap-2">
        {suggestions.map((s) => (
          <li
            key={s.id}
            className={
              s.isTop
                ? 'flex items-start justify-between gap-4 rounded-card border-l-4 border-brand bg-brand-tint p-4'
                : 'flex items-start justify-between gap-4 border-t border-hairline py-3 first:border-t-0'
            }
          >
            <div className="min-w-0">
              {s.isTop && (
                <span className="mb-1.5 inline-flex items-center rounded-full bg-brand px-2.5 py-1 text-micro font-semibold text-white">
                  {t('suggestion.top')}
                </span>
              )}
              {/* The proposal itself, at card-title weight. It used to render at
                  14px — smaller than the page's body text — which made the
                  community's actual idea the least prominent thing on a card
                  devoted to it. */}
              <p className={s.isTop ? 'text-title text-ink' : 'text-ink'}>{s.text}</p>
              {s.isTop && <p className="mt-1.5 text-meta text-brand-dark">{t('suggestion.topExplainer')}</p>}
            </div>
            <UpvoteButton
              count={s.upvoteCount}
              active={Boolean(s.myUpvote)}
              onToggle={() => toggleUpvote(s.id)}
              disabled={pendingId === s.id}
            />
          </li>
        ))}
      </ul>
      {error && (
        <p role="alert" className="mt-2 text-meta text-reopened-fg">
          {error}
        </p>
      )}
    </>
  )
}
