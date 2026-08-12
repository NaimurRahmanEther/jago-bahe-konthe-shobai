import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { formatDate } from '../../lib/date.js'

/**
 * The seat's public decision record, as a readable list.
 *
 * A SEPARATE component from AuditTrail, deliberately — do not generalise one into
 * the other. AuditTrail is the quiet monospace trail on a problem the reader is
 * already looking at, so an actor id and an action are context enough, and the
 * Guideline asks it to stay "plain, complete, permanent, not decorative". This is
 * seat-wide, so every row must say on its own WHO acted and about WHICH report —
 * otherwise the aggregate that makes a pattern visible is a wall of opaque ids.
 *
 * @param {{entries: import('../../lib/types/models.js').ActivityEntry[]}} props
 */
export default function ActivityFeed({ entries }) {
  const { t } = useTranslation()

  return (
    <ul className="flex flex-col">
      {entries.map((entry) => (
        <li key={entry.id} className="border-t border-hairline py-3 first:border-t-0">
          <p className="text-ink">
            {/* The actor leads: this page exists so a resident can see WHO decided.
                An unresolved id still renders — a name the server could not resolve
                costs the reader precision, not the row. `system` has no name to
                resolve and gets its own label: the platform acting by rule is a
                different kind of fact from a person deciding. */}
            <span className="font-semibold">
              {entry.actorName || (entry.actorId === 'system' ? t('activity.systemActor') : entry.actorId)}
            </span>
            {' — '}
            {t(`audit.action.${entry.action}`, entry.action)}
          </p>

          {entry.targetType === 'problem' && entry.problemTitle && (
            <Link to={`/problems/${entry.targetId}`} className="text-meta font-medium text-brand">
              {entry.problemTitle}
            </Link>
          )}

          {/* A rejection's ground is part of the public record (A.3.1 constraint 5):
              a takedown is accountable, not a disappearance. */}
          {entry.reason && <p className="text-meta text-muted">{entry.reason}</p>}

          <p className="text-micro text-muted">
            {formatDate(entry.createdAt, 'numeric')}
          </p>
        </li>
      ))}
    </ul>
  )
}
