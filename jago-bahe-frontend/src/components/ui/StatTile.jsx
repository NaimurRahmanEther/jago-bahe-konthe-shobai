import { toBengaliDigits } from '../../lib/numerals.js'

/**
 * One calm number over its label. Extracted at its third use (Scorecard,
 * AdminHome, MyReports) per A.5.8 — a repeated Tailwind class string becomes a
 * component.
 *
 * `tone` names the meaning, not the colour, so callers cannot quietly invent a
 * new one: `blocked` is amber because waiting on a higher authority is not
 * failure, and nothing here is ever red.
 *
 * The value bypasses `t()`, so it converts to Bengali numerals here — i18next's
 * formatter only reaches interpolated values (A.5.3).
 *
 * @param {{value: number|string, label: string, tone?: 'neutral'|'resolved'|'blocked'|'brand'}} props
 */
const TONES = {
  neutral: 'text-ink',
  resolved: 'text-resolved-fg',
  blocked: 'text-blocked-fg',
  brand: 'text-brand-dark',
}

export default function StatTile({ value, label, tone = 'neutral' }) {
  return (
    <div className="rounded-card border border-hairline bg-surface p-4 transition-colors duration-150 ease-standard hover:border-brand">
      <p className={`text-[28px] font-semibold leading-tight tabular-nums ${TONES[tone]}`}>
        {toBengaliDigits(value)}
      </p>
      <p className="mt-0.5 text-meta text-muted">{label}</p>
    </div>
  )
}
