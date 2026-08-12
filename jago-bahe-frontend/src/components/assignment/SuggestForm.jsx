import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import OfficialPicker from '../problem/OfficialPicker.jsx'

/** Above-union offices are the only ones worth advising: forwarding INTO a union
 *  would take a report that union's own admin should have decided. */
const ABOVE_UNION_TIERS = ['mp', 'minister', 'upazila_chairman', 'upazila_vice_chairman']

/**
 * One union admin's advice on where an above-union report should be forwarded.
 *
 * It is ADVICE. There is no quorum to reach, no window to beat, and no outcome it
 * can produce — the super admin decides, and may forward at any count including
 * none. The copy says so plainly rather than dressing this up as a vote, because
 * the previous incarnation WAS a vote and an admin who thinks they are voting will
 * read a forward that goes another way as their ballot being ignored (A.3.8).
 *
 * The reason is optional here, unlike the assign screen's override reason: this
 * settles nothing, so it owes the public nothing. The super admin's forward is
 * where a written reason becomes mandatory.
 *
 * @param {{
 *   mySuggestion: import('../../lib/types/models.js').ForwardingSuggestion|null,
 *   onSubmit: (payload: {officialId: string, reason?: string}) => Promise<unknown>,
 *   isSubmitting: boolean,
 *   error?: string,
 * }} props
 */
export default function SuggestForm({ mySuggestion, onSubmit, isSubmitting, error }) {
  const { t } = useTranslation()
  // Pre-filled from the advice already given, so "change my advice" starts from
  // what this admin actually said rather than from a blank form.
  const [officialId, setOfficialId] = useState(mySuggestion?.officialId ?? '')
  const [reason, setReason] = useState(mySuggestion?.reason ?? '')
  const [localError, setLocalError] = useState('')

  const revising = Boolean(mySuggestion)

  function handleSubmit(e) {
    e.preventDefault()
    if (!officialId) {
      setLocalError(t('assignment.errors.officialRequired'))
      return
    }
    setLocalError('')
    onSubmit({ officialId, reason: reason.trim() })
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 rounded-card border border-hairline bg-surface p-4">
      <div className="flex flex-col gap-1">
        <h2 className="font-semibold text-ink">
          {revising ? t('forwarding.advise.revisedTitle') : t('forwarding.advise.title')}
        </h2>
        <p className="text-meta text-muted">{t('forwarding.advise.note')}</p>
      </div>

      <OfficialPicker
        value={officialId}
        onChange={setOfficialId}
        filter={(o) => ABOVE_UNION_TIERS.includes(o.tier)}
        label={t('forwarding.advise.officialLabel')}
      />

      <div className="flex flex-col gap-1">
        <label htmlFor="advice-reason" className="font-medium text-ink">
          {t('forwarding.advise.reason')}{' '}
          <span className="font-normal text-muted">({t('forwarding.optional')})</span>
        </label>
        <textarea
          id="advice-reason"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={3}
          className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
        />
      </div>

      {(localError || error) && <p className="text-sm text-reopened-fg">{localError || error}</p>}

      <Button type="submit" disabled={isSubmitting} className="self-start">
        {revising ? t('forwarding.advise.submitRevised') : t('forwarding.advise.submit')}
      </Button>
    </form>
  )
}
