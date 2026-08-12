import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useForwardingQueueItem, useForwardProblem } from '../../hooks/useAssignments.js'
import { useOfficials } from '../../hooks/useOfficials.js'
import AdviceTally from '../../components/assignment/AdviceTally.jsx'
import OfficialPicker from '../../components/problem/OfficialPicker.jsx'
import Button from '../../components/ui/Button.jsx'
import ValidationBar from '../../components/ui/ValidationBar.jsx'
import Spinner from '../../components/ui/Spinner.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

const ABOVE_UNION_TIERS = ['mp', 'minister', 'upazila_chairman', 'upazila_vice_chairman']

/**
 * Mirrors the backend's domain.RequiresReason so the field can mark itself before
 * the request is sent. It MIRRORS the rule; it does not own it — a 400
 * `reason_required` is the authority, and this screen renders that too (A.5.7).
 *
 * Two independent triggers: the choice departs from the advisers' top suggestion,
 * or from the official the reporter pointed the report at. When those two disagree
 * NO choice satisfies both, so a reason is always required — that is the intent
 * rather than an edge case, because disagreement is exactly when the public
 * deserves the rationale.
 *
 * `top` is empty both when nobody advised AND when the advisers tied, which
 * removes the first trigger — a tie is genuine disagreement, and treating one side
 * of it as "the top" would make this requirement turn on a coin flip.
 */
function requiresReason(forwarded, top, pointed) {
  if (!forwarded) return false
  if (pointed && forwarded !== pointed) return true
  if (top && forwarded !== top) return true
  return false
}

/**
 * Where the super admin forwards one above-union report.
 *
 * The screen puts the REPORTER's choice and the ADMINS' advice side by side,
 * because that pairing is the decision: the resident asked for X, the seat's
 * admins advise Y, and the super admin answers in public.
 */
export default function ForwardPanel() {
  const { t } = useTranslation()
  const { id } = useParams()
  const navigate = useNavigate()
  const { data: item, isLoading, isError, refetch } = useForwardingQueueItem(id)
  const { data: officials } = useOfficials()
  const forward = useForwardProblem()

  const [officialId, setOfficialId] = useState('')
  const [priority, setPriority] = useState('normal')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  if (isError) return <ErrorState message={t('forwarding.super.error')} onRetry={refetch} />
  if (!item) return <ErrorState message={t('forwarding.notFound')} onRetry={refetch} />

  const pointed = officials?.find((o) => o.id === item.pointedOfficialId)
  const top = officials?.find((o) => o.id === item.topOfficialId)
  const needsReason = requiresReason(officialId, item.topOfficialId, item.pointedOfficialId)

  async function handleSubmit(e) {
    e.preventDefault()
    if (!officialId) {
      setError(t('assignment.errors.officialRequired'))
      return
    }
    if (needsReason && !reason.trim()) {
      setError(t('forwarding.super.reasonRequired'))
      return
    }
    setError('')
    try {
      await forward.mutateAsync({ problemId: item.problemId, officialId, priority, reason: reason.trim() })
      navigate('/super/queue')
    } catch (err) {
      // The backend decides the rule, so its refusal is rendered as a field error
      // even when the mirror above thought the reason was optional — a drift
      // between the two must surface here, not as a generic failure.
      setError(
        err?.code === 'reason_required'
          ? t('forwarding.super.reasonRequired')
          : t('forwarding.super.failed'),
      )
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Link to="/super/queue" className="text-meta font-medium text-brand">
        {t('forwarding.back')}
      </Link>

      <div className="flex flex-col gap-2">
        <h1 className="text-h1 font-semibold text-ink">{item.title}</h1>
        <p className="text-meta text-muted">{item.address}</p>
        <ValidationBar count={item.validCount} threshold={item.validationThreshold} className="mt-1" />
        <Link to={`/problems/${item.problemId}`} className="text-meta font-medium text-brand">
          {t('forwarding.viewReport')}
        </Link>
      </div>

      {/* The two inputs to the decision, side by side. Neither binds. */}
      <div className="grid gap-3 sm:grid-cols-2">
        <section className="flex flex-col gap-1 rounded-card border border-hairline bg-surface p-4">
          <h2 className="text-meta font-medium text-muted">{t('forwarding.pointed')}</h2>
          <p className="text-ink">
            {pointed ? (
              <>
                {pointed.name}{' '}
                <span className="text-muted">({t(`problem.tier.${pointed.tier}`)})</span>
              </>
            ) : (
              <span className="text-muted">{t('forwarding.pointedUnknown')}</span>
            )}
          </p>
        </section>

        <section className="flex flex-col gap-1 rounded-card border border-hairline bg-surface p-4">
          <h2 className="text-meta font-medium text-muted">{t('forwarding.topAdvice')}</h2>
          <p className="text-ink">
            {top ? (
              <>
                {top.name}{' '}
                <span className="text-muted">
                  ({t('forwarding.ofAdvisers', { count: item.topCount })})
                </span>
              </>
            ) : (
              <span className="text-muted">
                {item.suggestions?.length > 0 ? t('forwarding.tiedNoTop') : t('forwarding.noAdvice')}
              </span>
            )}
          </p>
        </section>
      </div>

      <section className="flex flex-col gap-2">
        <h2 className="font-semibold text-ink">{t('forwarding.tallyTitle')}</h2>
        <AdviceTally suggestions={item.suggestions} topOfficialId={item.topOfficialId} />
      </section>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4 rounded-card border border-hairline bg-surface p-4">
        <h2 className="font-semibold text-ink">{t('forwarding.super.decideTitle')}</h2>

        <OfficialPicker
          value={officialId}
          onChange={setOfficialId}
          filter={(o) => ABOVE_UNION_TIERS.includes(o.tier)}
          label={t('forwarding.super.officialLabel')}
        />

        <div className="flex flex-col gap-2">
          <label htmlFor="forward-priority" className="font-medium text-ink">
            {t('assignment.priority')}
          </label>
          <select
            id="forward-priority"
            value={priority}
            onChange={(e) => setPriority(e.target.value)}
            className="min-h-11 rounded-control border border-hairline bg-surface px-3 text-ink"
          >
            <option value="low">{t('assignment.priorities.low')}</option>
            <option value="normal">{t('assignment.priorities.normal')}</option>
            <option value="high">{t('assignment.priorities.high')}</option>
          </select>
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="forward-reason" className="font-medium text-ink">
            {t('forwarding.super.reason')}{' '}
            <span className={`font-normal ${needsReason ? 'text-brand-dark' : 'text-muted'}`}>
              ({needsReason ? t('assignment.required') : t('forwarding.optional')})
            </span>
          </label>
          {needsReason && (
            <p className="text-meta text-muted">{t('forwarding.super.reasonWhy')}</p>
          )}
          <textarea
            id="forward-reason"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={3}
            className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
          />
        </div>

        {error && <p className="text-sm text-reopened-fg">{error}</p>}

        <Button type="submit" disabled={forward.isPending} className="self-start">
          {t('forwarding.super.submit')}
        </Button>
      </form>
    </div>
  )
}
