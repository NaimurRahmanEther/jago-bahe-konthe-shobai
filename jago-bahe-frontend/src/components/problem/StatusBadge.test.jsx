import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import StatusBadge from './StatusBadge.jsx'

// The badge renders null for a status it has no tokens for, which fails silently
// — the label simply vanishes rather than throwing. So every state the backend can
// send is pinned here by the literal Bangla a user actually reads: asserting
// i18n.t(key) would compare the key to itself and pass on a blank translation
// (CLAUDE.md A.5.3 rule 3).
describe('StatusBadge', () => {
  const LABELS = {
    PendingApproval: 'অনুমোদনের অপেক্ষায়',
    Reported: 'জানানো হয়েছে',
    Validated: 'যাচাইকৃত',
    Assigned: 'নির্ধারিত',
    InProgress: 'কাজ চলছে',
    Blocked: 'আটকে আছে',
    Done: 'সম্পন্ন',
    Resolved: 'সমাধান হয়েছে',
    Reopened: 'পুনরায় খোলা',
    Rejected: 'প্রত্যাখ্যাত',
    Withdrawn: 'প্রত্যাহার করা হয়েছে',
  }

  it('renders every lifecycle state with a label, never color alone', () => {
    // Mirrors problem/domain/status.go's eleven-state vocabulary — including
    // PendingApproval, the pre-publication gate's hidden state.
    expect(Object.keys(LABELS)).toHaveLength(11)

    for (const [status, label] of Object.entries(LABELS)) {
      const { unmount } = render(<StatusBadge status={status} />)
      // A visible text label, not just a tint: color is never the only carrier.
      expect(screen.getByText(label)).toBeInTheDocument()
      unmount()
    }
  })

  it('renders nothing for an unknown status', () => {
    // The badge renders null for a status it has no tokens for, so an out-of-
    // vocabulary value fails loudly (no badge) rather than a broken half-badge.
    const { container } = render(<StatusBadge status="Nonsense" />)
    expect(container.firstChild).toBeNull()
  })

  it('gives Blocked amber and Rejected the muted maroon, not Reopened bright red', () => {
    const { container: blocked } = render(<StatusBadge status="Blocked" />)
    expect(blocked.firstChild.className).toContain('blocked-bg')

    const { container: rejected } = render(<StatusBadge status="Rejected" />)
    expect(rejected.firstChild.className).toContain('rejected-bg')
    expect(rejected.firstChild.className).not.toContain('reopened-bg')
  })
})
