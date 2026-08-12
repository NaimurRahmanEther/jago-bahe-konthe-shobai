import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePendingClaims, useApproveClaim, useRejectClaim } from '../../hooks/useClaims.js'
import Button from '../../components/ui/Button.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

/**
 * One claim awaiting a decision. This page is a SINGLE route for both the union
 * admin and the super admin; the backend has already filtered the list by the
 * office's tier, so there is no role branching here.
 */
function ClaimRow({ claim, index, onApprove, onReject, isSubmitting }) {
  const { t } = useTranslation()
  const [rejecting, setRejecting] = useState(false)
  const [reason, setReason] = useState('')
  const [reasonError, setReasonError] = useState('')

  function submitReject() {
    if (!reason.trim()) {
      setReasonError(t('claims.reasonRequired'))
      return
    }
    onReject(reason.trim())
  }

  return (
    <div
      style={{ animationDelay: `${Math.min(index, 5) * 40}ms` }}
      className="flex flex-col gap-3 rounded-card border border-hairline bg-surface p-4 animate-rise-in"
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex flex-col gap-1">
          <p className="font-semibold text-ink">{claim.claimantName}</p>
          <p className="text-meta text-muted">
            {t('claims.office')}: {claim.officialName}
            <span className="ml-2 rounded-full bg-brand-tint px-2 py-0.5 text-micro font-medium text-brand-dark">
              {t(`problem.tier.${claim.officialTier}`)}
            </span>
          </p>
          {claim.claimantNid && (
            <p className="text-meta text-muted">
              {t('claims.nid')}: {claim.claimantNid}
            </p>
          )}
        </div>
      </div>

      {!rejecting ? (
        <div className="flex flex-wrap gap-2">
          <Button onClick={onApprove} disabled={isSubmitting}>
            {t('claims.approve')}
          </Button>
          <Button variant="secondary" onClick={() => setRejecting(true)} disabled={isSubmitting}>
            {t('claims.reject')}
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-2 border-t border-hairline pt-3">
          <label htmlFor={`reason-${claim.id}`} className="font-medium text-ink">
            {t('claims.reasonLabel')}
          </label>
          <textarea
            id={`reason-${claim.id}`}
            value={reason}
            onChange={(e) => {
              setReason(e.target.value)
              setReasonError('')
            }}
            rows={2}
            placeholder={t('claims.reasonPlaceholder')}
            className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
          />
          {reasonError && <p className="text-sm text-reopened-fg">{reasonError}</p>}
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={submitReject} disabled={isSubmitting}>
              {t('claims.confirmReject')}
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setRejecting(false)
                setReason('')
                setReasonError('')
              }}
              disabled={isSubmitting}
            >
              {t('claims.cancel')}
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

export default function ClaimsReview() {
  const { t } = useTranslation()
  const { data: claims, isLoading, isError, refetch } = usePendingClaims()
  const approve = useApproveClaim()
  const reject = useRejectClaim()
  const isSubmitting = approve.isPending || reject.isPending

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-h1 font-semibold text-ink">{t('claims.title')}</h1>
        <p className="text-sm text-muted">{t('claims.subtitle')}</p>
      </div>

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('claims.error')} onRetry={refetch} />}

      {!isLoading && !isError && claims?.length === 0 && <EmptyState title={t('claims.empty')} />}

      {!isLoading && !isError && claims?.length > 0 && (
        <div className="flex flex-col gap-3">
          {claims.map((claim, i) => (
            <ClaimRow
              key={claim.id}
              claim={claim}
              index={i}
              isSubmitting={isSubmitting}
              onApprove={() => approve.mutate(claim.id)}
              onReject={(reason) => reject.mutate({ claimId: claim.id, reason })}
            />
          ))}
        </div>
      )}
    </div>
  )
}
