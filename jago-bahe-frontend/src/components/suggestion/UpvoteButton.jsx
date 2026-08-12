import { useTranslation } from 'react-i18next'
import { toBengaliDigits } from '../../lib/numerals.js'

/**
 * Support for one proposed fix.
 *
 * A real 44×44 target (A.6): thumbs, not cursors, cast these. `aria-pressed`
 * carries the state for a screen reader, so the colour change is never the only
 * signal that a vote is already held.
 *
 * @param {{count: number, active: boolean, onToggle: () => void, disabled?: boolean}} props
 */
export default function UpvoteButton({ count, active, onToggle, disabled }) {
  const { t } = useTranslation()

  return (
    <button
      type="button"
      onClick={onToggle}
      disabled={disabled}
      aria-pressed={active}
      aria-label={t('suggestion.upvote')}
      className={`flex size-14 shrink-0 flex-col items-center justify-center gap-0.5 rounded-control border font-semibold transition-colors duration-150 disabled:opacity-50 ${
        active
          ? 'border-brand bg-brand text-white'
          : 'border-hairline bg-surface text-ink hover:border-brand hover:bg-brand-tint'
      }`}
    >
      <span aria-hidden="true" className="text-micro leading-none">
        ▲
      </span>
      {/* Bengali numerals — the app is Bangla-only, and a Latin digit here beside
          Bengali body text is the kind of seam A.5.3 rule 5 exists to close. This
          number never passes through t(), so it takes the helper. */}
      <span className="tabular-nums leading-none">{toBengaliDigits(count)}</span>
    </button>
  )
}
