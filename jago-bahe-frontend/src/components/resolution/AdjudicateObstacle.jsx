import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import { formatDate } from '../../lib/date.js'

/**
 * The admin/moderator authority action on a blocked case's obstacle (Concept
 * §8/§9): confirm the obstacle is real (responsibility moves up, the official is
 * protected, the case stays Blocked) or deny it (within the official's power —
 * the case bounces back to InProgress and may count against them). This binding
 * verdict, not the advisory public vote, sets the scorecard consequence.
 *
 * Backend authority: the flip is enforced by POST /admin/obstacles/{id}/adjudicate;
 * this UI only collects the decision (role guard is UX only) and reflects state.
 * A two-step confirm guards the irreversible ruling.
 *
 * @param {{
 *   obstacle: import('../../lib/types/models.js').Obstacle,
 *   onAdjudicate: (decision: 'confirm' | 'deny') => void,
 *   isSubmitting: boolean,
 *   isError: boolean,
 * }} props
 */
export default function AdjudicateObstacle({ obstacle, onAdjudicate, isSubmitting, isError }) {
  const { t } = useTranslation()
  const [pendingDecision, setPendingDecision] = useState(null)

  const ruled = obstacle.adjudication === 'confirmed' || obstacle.adjudication === 'denied'

  // Already ruled — a quiet, read-only record of the verdict.
  if (ruled) {
    const confirmed = obstacle.adjudication === 'confirmed'
    return (
      <div className="rounded-card border border-hairline bg-surface p-4">
        <div className="flex items-center justify-between gap-3">
          <p className="text-sm font-semibold text-ink">{t('obstacle.adjudicate.title')}</p>
          <span
            className={`inline-flex rounded-full px-2 py-0.5 text-[11px] font-bold ${
              confirmed ? 'bg-blocked-bg text-blocked-fg' : 'bg-brand-tint text-brand-dark'
            }`}
          >
            {confirmed ? t('obstacle.adjudicate.confirmedBadge') : t('obstacle.adjudicate.deniedBadge')}
          </span>
        </div>
        <p className="mt-2 text-sm text-muted">
          {confirmed ? t('obstacle.adjudicate.confirmedNote') : t('obstacle.adjudicate.deniedNote')}
        </p>
        {obstacle.adjudicatedAt && (
          <p className="mt-1 text-xs text-muted">
            {t('obstacle.adjudicate.ruledOn', {
              date: formatDate(obstacle.adjudicatedAt, 'numeric'),
            })}
          </p>
        )}
      </div>
    )
  }

  return (
    <div className="rounded-card border border-hairline bg-surface p-4">
      <p className="text-sm font-semibold text-ink">{t('obstacle.adjudicate.title')}</p>
      <p className="mt-1 text-sm text-muted">{t('obstacle.adjudicate.intro')}</p>

      {pendingDecision ? (
        // Step two: confirm the irreversible ruling, with its consequence spelled out.
        <div className="mt-3 rounded-card border border-hairline bg-canvas p-3">
          <p className="text-sm text-ink">
            {pendingDecision === 'confirm'
              ? t('obstacle.adjudicate.confirmHint', { who: obstacle.whoUnblocks })
              : t('obstacle.adjudicate.denyHint')}
          </p>
          <div className="mt-3 flex gap-3">
            <Button
              className="min-h-12 flex-1"
              onClick={() => onAdjudicate(pendingDecision)}
              disabled={isSubmitting}
            >
              {t('obstacle.adjudicate.confirmRuling')}
            </Button>
            <Button
              variant="secondary"
              className="min-h-12 flex-1"
              onClick={() => setPendingDecision(null)}
              disabled={isSubmitting}
            >
              {t('common.cancel')}
            </Button>
          </div>
        </div>
      ) : (
        // Step one: choose the verdict.
        <div className="mt-3 flex flex-col gap-2 sm:flex-row">
          <Button
            className="min-h-12 flex-1"
            onClick={() => setPendingDecision('confirm')}
            disabled={isSubmitting}
          >
            {t('obstacle.adjudicate.confirm')}
          </Button>
          <Button
            variant="secondary"
            className="min-h-12 flex-1"
            onClick={() => setPendingDecision('deny')}
            disabled={isSubmitting}
          >
            {t('obstacle.adjudicate.deny')}
          </Button>
        </div>
      )}

      {isError && <p className="mt-2 text-sm text-reopened-fg">{t('obstacle.adjudicate.error')}</p>}
    </div>
  )
}
