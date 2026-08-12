import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import StatusBadge from './StatusBadge.jsx'
import { REJECTION_REASONS } from '../../lib/api/moderation.js'

/**
 * One report awaiting screening: the admin approves it into public view, or takes
 * it down on one of four fixed grounds. There is no "reject on merit" — whether a
 * genuine report is real is the community's call via V, not an admin's (Concept
 * §4); the admin only clears spam, abuse, duplicates, and wrong-area reports.
 *
 * @param {{problem: object, index?: number, onApprove: () => void, onReject: (payload: {reason: string, note?: string}) => void, isSubmitting?: boolean}} props
 */
export default function ModerationItem({ problem, index = 0, onApprove, onReject, isSubmitting = false }) {
  const { t } = useTranslation()
  const [rejecting, setRejecting] = useState(false)
  const [reason, setReason] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState('')

  function submitRejection(e) {
    e.preventDefault()
    // The enum is backend-authoritative; this only stops an obviously empty
    // submission from making the round trip.
    if (!reason) {
      setError(t('moderation.reasonRequired'))
      return
    }
    setError('')
    onReject({ reason, note: note.trim() || undefined })
  }

  return (
    <div
      style={{ animationDelay: `${Math.min(index, 5) * 40}ms` }}
      className="rounded-card border border-hairline bg-surface p-4 animate-rise-in"
    >
      <div className="flex items-start justify-between gap-3">
        <Link to={`/problems/${problem.id}`} className="font-semibold leading-snug text-ink hover:text-brand">
          {problem.title}
        </Link>
        <StatusBadge status={problem.status} className="shrink-0" />
      </div>
      <p className="mt-2 text-sm text-muted">{problem.location.address}</p>
      <p className="mt-2 text-sm text-ink">{problem.description}</p>

      {!rejecting && (
        <div className="mt-4 flex flex-wrap gap-2">
          <Button onClick={onApprove} disabled={isSubmitting}>
            {t('moderation.approve')}
          </Button>
          <Button variant="secondary" onClick={() => setRejecting(true)} disabled={isSubmitting}>
            {t('moderation.reject')}
          </Button>
        </div>
      )}

      {rejecting && (
        <form onSubmit={submitRejection} className="mt-4 flex flex-col gap-3 border-t border-hairline pt-4">
          <div className="flex flex-col gap-1">
            <label htmlFor={`reason-${problem.id}`} className="text-sm font-medium text-ink">
              {t('moderation.reasonLabel')}
            </label>
            <select
              id={`reason-${problem.id}`}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              className="min-h-11 rounded-control border border-hairline bg-surface px-3 text-ink"
            >
              <option value="">{t('moderation.reasonPlaceholder')}</option>
              {REJECTION_REASONS.map((r) => (
                <option key={r} value={r}>
                  {t(`problem.rejectionReason.${r}`)}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1">
            <label htmlFor={`note-${problem.id}`} className="text-sm font-medium text-ink">
              {t('moderation.noteLabel')}
            </label>
            <input
              id={`note-${problem.id}`}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              className="min-h-11 rounded-control border border-hairline bg-surface px-3 text-ink"
            />
          </div>

          {error && <p className="text-sm text-rejected-fg">{error}</p>}
          <p className="text-sm text-muted">{t('moderation.publicNote')}</p>

          <div className="flex flex-wrap gap-2">
            <Button type="submit" disabled={isSubmitting}>
              {t('moderation.confirmReject')}
            </Button>
            <Button variant="ghost" onClick={() => setRejecting(false)} disabled={isSubmitting}>
              {t('moderation.cancel')}
            </Button>
          </div>
        </form>
      )}
    </div>
  )
}
