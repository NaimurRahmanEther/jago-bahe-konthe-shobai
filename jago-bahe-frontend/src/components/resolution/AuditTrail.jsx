import { useTranslation } from 'react-i18next'
import { formatDate } from '../../lib/date.js'

/**
 * The public audit trail: actor · action · date, oldest→newest.
 *
 * Collapsed behind a native <details> rather than always open. Nothing is removed —
 * the Guideline asks this to feel "plain, complete, and permanent", and it still is,
 * one keystroke away with its length stated up front. What collapsing buys is that
 * the longest, densest block on the page stops competing with the official's reply
 * for a reader arriving to find out what is being done about a problem.
 *
 * <details> rather than useState on purpose: it is keyboard-operable and screen-reader
 * announced for free, it works before hydration, and it needs no state to get wrong.
 *
 * The monospace stays. It is the one genuinely distinct surface on the page and it
 * already does the job the redesign is trying to do everywhere else — this block was
 * never the part that was hard to tell apart.
 *
 * @param {{entries: import('../../lib/types/models.js').AuditEntry[]}} props
 */
export default function AuditTrail({ entries }) {
  const { t } = useTranslation()

  return (
    <details className="group">
      <summary className="flex min-h-11 cursor-pointer list-none items-center gap-2 font-medium text-brand">
        {/* A caret drawn in text, not an icon font: A.6 pairs icons with words, and
            this one is decorative — the words carry the meaning. */}
        <span aria-hidden="true" className="transition-transform duration-150 ease-standard group-open:rotate-90">
          ›
        </span>
        {entries.length > 0
          ? t('problem.detail.auditSummary', { count: entries.length })
          : t('problem.detail.auditSummaryEmpty')}
      </summary>

      {entries.length === 0 ? (
        <p className="pt-2 text-muted">{t('audit.empty')}</p>
      ) : (
        <ul className="flex flex-col pt-2">
          {entries.map((entry) => (
            <li
              key={entry.id}
              className="flex flex-wrap items-baseline gap-x-2 border-t border-hairline py-2 font-mono text-micro first:border-t-0"
            >
              <span className="text-muted">{formatDate(entry.createdAt, 'numeric')}</span>
              <span className="font-semibold text-ink">{entry.actor}</span>
              <span className="text-ink">{t(`audit.action.${entry.action}`, entry.action)}</span>
              {entry.reason && <span className="text-muted">— {entry.reason}</span>}
            </li>
          ))}
        </ul>
      )}
    </details>
  )
}
