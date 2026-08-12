import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import OfficialPicker from '../problem/OfficialPicker.jsx'

/**
 * @param {{
 *   publicChoice: import('../../lib/types/models.js').Official,
 *   onConfirm: (payload: {officialId: string, priority: string, overrideReason?: string}) => Promise<unknown>,
 *   isSubmitting: boolean,
 * }} props
 */
export default function AssignForm({ publicChoice, onConfirm, isSubmitting }) {
  const { t } = useTranslation()
  const [overriding, setOverriding] = useState(false)
  const [overrideOfficialId, setOverrideOfficialId] = useState('')
  const [reason, setReason] = useState('')
  const [priority, setPriority] = useState('normal')
  const [error, setError] = useState('')

  function handleConfirm() {
    onConfirm({ officialId: publicChoice.id, priority })
  }

  function handleOverride(e) {
    e.preventDefault()
    if (!overrideOfficialId) {
      setError(t('assignment.errors.officialRequired'))
      return
    }
    if (!reason.trim()) {
      setError(t('assignment.errors.reasonRequired'))
      return
    }
    setError('')
    onConfirm({ officialId: overrideOfficialId, priority, overrideReason: reason.trim() })
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-card border border-hairline bg-surface p-4">
        <p className="text-sm text-muted">{t('assignment.publicChoice')}</p>
        <p className="mt-1 font-semibold text-ink">
          {publicChoice.name}{' '}
          <span className="font-normal text-muted">({t(`problem.tier.${publicChoice.tier}`)})</span>
        </p>
      </div>

      <div className="flex flex-col gap-2">
        <label htmlFor="priority" className="font-medium text-ink">
          {t('assignment.priority')}
        </label>
        <select
          id="priority"
          value={priority}
          onChange={(e) => setPriority(e.target.value)}
          className="min-h-11 rounded-control border border-hairline bg-surface px-3 text-ink"
        >
          <option value="low">{t('assignment.priorities.low')}</option>
          <option value="normal">{t('assignment.priorities.normal')}</option>
          <option value="high">{t('assignment.priorities.high')}</option>
        </select>
      </div>

      {!overriding ? (
        <div className="flex flex-wrap gap-3">
          <Button onClick={handleConfirm} disabled={isSubmitting}>
            {t('assignment.confirm')}
          </Button>
          <Button variant="secondary" onClick={() => setOverriding(true)} disabled={isSubmitting}>
            {t('assignment.override')}
          </Button>
        </div>
      ) : (
        <form onSubmit={handleOverride} className="flex flex-col gap-4 rounded-card border border-hairline bg-surface p-4">
          <OfficialPicker value={overrideOfficialId} onChange={setOverrideOfficialId} />
          <div className="flex flex-col gap-1">
            <label htmlFor="override-reason" className="font-medium text-ink">
              {t('assignment.reason')} <span className="font-normal text-brand-dark">({t('assignment.required')})</span>
            </label>
            <textarea
              id="override-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
              className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
            />
          </div>
          {error && <p className="text-sm text-reopened-fg">{error}</p>}
          <div className="flex gap-3">
            <Button type="submit" disabled={isSubmitting}>
              {t('assignment.submitOverride')}
            </Button>
            <Button type="button" variant="secondary" onClick={() => setOverriding(false)} disabled={isSubmitting}>
              {t('common.cancel')}
            </Button>
          </div>
        </form>
      )}
    </div>
  )
}
