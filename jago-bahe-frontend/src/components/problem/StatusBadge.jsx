import { useTranslation } from 'react-i18next'

// Design Guideline §2 — tint bg + same-hue text + dot + label. Never color alone.
// Blocked is amber (waiting, not failure). Red means "did not pass", and
// brightness carries the meaning: bright red is Reopened alone (live, back to
// work); Rejected is the muted maroon (terminal, off the record).
const STATUS_TOKENS = {
  PendingApproval: { bg: 'bg-pendingapproval-bg', fg: 'text-pendingapproval-fg', dot: 'bg-pendingapproval-dot' },
  Reported: { bg: 'bg-reported-bg', fg: 'text-reported-fg', dot: 'bg-reported-dot' },
  Validated: { bg: 'bg-validated-bg', fg: 'text-validated-fg', dot: 'bg-validated-dot' },
  Assigned: { bg: 'bg-assigned-bg', fg: 'text-assigned-fg', dot: 'bg-assigned-dot' },
  InProgress: { bg: 'bg-progress-bg', fg: 'text-progress-fg', dot: 'bg-progress-dot' },
  Blocked: { bg: 'bg-blocked-bg', fg: 'text-blocked-fg', dot: 'bg-blocked-dot' },
  Done: { bg: 'bg-done-bg', fg: 'text-done-fg', dot: 'bg-done-dot' },
  Resolved: { bg: 'bg-resolved-bg', fg: 'text-resolved-fg', dot: 'bg-resolved-dot' },
  Reopened: { bg: 'bg-reopened-bg', fg: 'text-reopened-fg', dot: 'bg-reopened-dot' },
  Rejected: { bg: 'bg-rejected-bg', fg: 'text-rejected-fg', dot: 'bg-rejected-dot' },
  Withdrawn: { bg: 'bg-withdrawn-bg', fg: 'text-withdrawn-fg', dot: 'bg-withdrawn-dot' },
}

/** @param {{status: import('../../lib/types/models.js').ProblemStatus, className?: string}} props */
export default function StatusBadge({ status, className = '' }) {
  const { t } = useTranslation()
  const tokens = STATUS_TOKENS[status]

  // A status outside the eleven-state lifecycle shouldn't occur — the backend
  // whitelists them — so render nothing rather than a broken badge.
  if (!tokens) return null

  // 13px, not 12: the badge is the app's visual backbone (Guideline §1 — "spend
  // your boldness on the status system"), and it was set smaller than the meta
  // text it sat beside.
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[13px] font-semibold leading-normal ${tokens.bg} ${tokens.fg} ${className}`}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${tokens.dot}`} aria-hidden="true" />
      {t(`problem.status.${status}`)}
    </span>
  )
}
