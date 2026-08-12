import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import StatusBadge from '../problem/StatusBadge.jsx'
import ObservationNoteForm from './ObservationNoteForm.jsx'
import { toPublicStatus } from '../../hooks/useCases.js'
import { isWaiting } from '../../hooks/useObservations.js'
import { formatDate } from '../../lib/date.js'

/**
 * One case below this official on the accountability ladder.
 *
 * The silence flag is AMBER, never red. Red on this platform means "did not
 * pass" (A.6); an official who has not answered yet is being waited on, and the
 * page has to stay a noticeboard rather than a pillory — the same reason Blocked
 * is amber. Colour is never the only signal: the days are spelled out in words.
 *
 * The row links to the PUBLIC problem page, not to /official/cases/{id}: a
 * monitor is not the case's owner and the backend refuses them there (403).
 * Nothing is lost — the plan, updates and evidence are already public.
 *
 * @param {{
 *   row: import('../../lib/types/models.js').ObservedCase,
 *   onNote: (payload: {observationId: string, text: string}) => void,
 *   isNoting: boolean,
 *   noteFailed: boolean,
 * }} props
 */
export default function ObservationRow({ row, onNote, isNoting, noteFailed }) {
  const { t } = useTranslation()
  const waiting = isWaiting(row)
  const openObservation = (row.observations ?? []).find((o) => o.resolvedAt === null)
  const notes = (row.observations ?? []).flatMap((o) => o.notes ?? [])

  return (
    <article
      className={`rounded-card border bg-surface p-4 ${waiting ? 'border-blocked-fg/40' : 'border-hairline'}`}
    >
      <div className="flex items-start justify-between gap-3">
        <Link to={`/problems/${row.problemId}`} className="text-title font-semibold text-ink hover:text-brand">
          {row.problemTitle || t('observation.untitled')}
        </Link>
        <StatusBadge status={toPublicStatus(row.status)} className="shrink-0" />
      </div>

      <p className="mt-1 text-meta text-muted">
        {t('observation.official', { name: row.officialName || row.officialId })}
      </p>

      {waiting ? (
        <p className="mt-3 rounded-control bg-blocked-bg px-3 py-2 text-meta text-blocked-fg">
          {t('observation.silentFor', { count: row.silentDays })}
          {openObservation && openObservation.level > 1 && (
            <> · {t('observation.climbedToYou')}</>
          )}
        </p>
      ) : (
        <p className="mt-3 text-meta text-muted">
          {row.lastActivityAt
            ? t('observation.lastActed', { date: formatDate(row.lastActivityAt) })
            : t('observation.neverActed')}
          {' · '}
          {t('observation.deadline', { date: formatDate(row.deadline) })}
        </p>
      )}

      {/* The history survives the response, so a pattern of repeated silences
          stays visible rather than vanishing the moment each one ends. */}
      {(row.observations ?? []).some((o) => o.resolvedAt !== null) && (
        <p className="mt-2 text-micro text-muted">
          {t('observation.pastSilences', {
            count: (row.observations ?? []).filter((o) => o.resolvedAt !== null).length,
          })}
        </p>
      )}

      {notes.length > 0 && (
        <ul className="mt-3 flex flex-col gap-2 border-t border-hairline pt-3">
          {notes.map((n) => (
            <li key={n.id} className="text-meta text-ink">
              <span className="text-muted tabular-nums">{formatDate(n.createdAt)}</span> — {n.text}
            </li>
          ))}
        </ul>
      )}

      {/* A note is the answer to an escalation, so it is offered only while one is
          open — there is nothing to answer on a case nobody is waiting on. */}
      {openObservation && (
        <ObservationNoteForm
          observationId={openObservation.id}
          onSubmit={onNote}
          isSubmitting={isNoting}
          isError={noteFailed}
        />
      )}
    </article>
  )
}
