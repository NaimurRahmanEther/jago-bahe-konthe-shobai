import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import ProblemCard from './ProblemCard.jsx'

// The card looks up the pointed official via a query hook; stub it so the test
// is about the card's rendering, not data fetching.
vi.mock('../../hooks/useOfficials.js', () => ({
  useOfficials: () => ({ data: [{ id: 'off-1', name: 'Karim Uddin', tier: 'ward_member', areaId: 'ward-1' }] }),
}))

const problem = {
  id: 'p1',
  title: 'Broken road',
  description: 'x',
  location: { areaId: 'union-1', address: 'Ward 1' },
  reporterId: 'r1',
  pointedOfficialId: 'off-1',
  status: 'Reported',
  validCount: 2,
  validationThreshold: 5,
  myVote: null,
  createdAt: new Date().toISOString(),
}

function renderCard(p = problem) {
  return render(
    <MemoryRouter>
      <ProblemCard problem={p} />
    </MemoryRouter>,
  )
}

describe('ProblemCard', () => {
  it('shows title, pointed official, status badge, and the validation count', () => {
    renderCard()
    expect(screen.getByText('Broken road')).toBeInTheDocument()
    expect(screen.getByText(/Karim Uddin/)).toBeInTheDocument()
    expect(screen.getByText('জানানো হয়েছে')).toBeInTheDocument()
    // Bengali numerals, not Latin — the app is Bangla-only and the digits render
    // through i18next's `number` formatter (A.5.3).
    expect(screen.getByText('২ জন যাচাই করেছেন')).toBeInTheDocument()
  })

  // The public no longer sees a target. Reaching V unlocks nothing since B17, so a
  // denominator here would advertise a gate that does not exist (A.3.1.1). "X / V"
  // survives on admin surfaces only — if this assertion ever fails, check whether
  // ValidationBar crept back into the feed.
  it('shows no threshold and no progress bar', () => {
    renderCard()
    expect(screen.queryByText('২ / ৫ জন যাচাই করেছেন')).toBeNull()
    expect(screen.queryByRole('progressbar')).toBeNull()
  })

  it('reads "nobody yet" rather than a zero count', () => {
    renderCard({ ...problem, validCount: 0 })
    expect(screen.getByText('এখনও কেউ যাচাই করেননি')).toBeInTheDocument()
  })

  // The card shows the validation result and offers nothing to act on: casting a
  // vote belongs on the detail page, where the reader has the report in front of
  // them. A compact ValidationVote lived here briefly and was taken back out.
  it('offers no vote controls — the count is a result, not a prompt', () => {
    renderCard()
    expect(screen.queryByRole('button')).toBeNull()
    expect(screen.queryByRole('button', { name: 'বৈধ' })).toBeNull()
  })

  it('links to the problem detail page', () => {
    renderCard()
    expect(screen.getByRole('link')).toHaveAttribute('href', '/problems/p1')
  })

  it('shows an explicit details affordance that is not a second link', () => {
    renderCard()
    expect(screen.getByText('বিস্তারিত দেখুন')).toBeInTheDocument()
    // The card itself is the anchor. An <a> or <button> for "details" would be
    // an interactive element nested inside one — invalid HTML, and it breaks
    // keyboard navigation. One link, always. This is the constraint that sent the
    // vote buttons back to the detail page rather than breaking up the anchor.
    expect(screen.getAllByRole('link')).toHaveLength(1)
    expect(screen.queryByRole('button')).toBeNull()
  })

  it('renders the reporter photo only when present', () => {
    const { rerender } = renderCard()
    expect(document.querySelector('img')).toBeNull()
    rerender(
      <MemoryRouter>
        <ProblemCard problem={{ ...problem, imageUrl: 'data:image/png;base64,abc' }} />
      </MemoryRouter>,
    )
    expect(document.querySelector('img')).not.toBeNull()
  })
})
