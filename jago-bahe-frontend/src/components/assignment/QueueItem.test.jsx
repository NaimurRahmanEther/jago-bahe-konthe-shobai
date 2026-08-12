import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import QueueItem from './QueueItem.jsx'

vi.mock('../../hooks/useOfficials.js', () => ({
  useOfficials: () => ({
    data: [{ id: 'off-1', name: 'করিম উদ্দিন', tier: 'ward_member', areaId: 'ward-1' }],
  }),
}))

// The fixture mirrors queueItemDTO field-for-field. That is the whole point of
// this file: every field below was read by this component and sent by nothing.
const ROW = {
  problemId: 'prob-2',
  title: 'রাস্তার বাতিগুলো জ্বলছে না',
  status: 'Reported',
  address: 'স্কুল রোড, ওয়ার্ড ২',
  pointedOfficialId: 'off-1',
  areaId: 'ward-2',
  routing: 'union',
  validCount: 3,
  validationThreshold: 5,
}

const renderRow = (overrides = {}) =>
  render(
    <MemoryRouter>
      <QueueItem item={{ ...ROW, ...overrides }} />
    </MemoryRouter>,
  )

describe('QueueItem', () => {
  // THE REGRESSION PIN. This component read `item.id`, `item.location.address`,
  // `item.pointedOfficial` and `item.vote` — none of which the API ever sent — so
  // any non-empty queue threw a TypeError on `location.address` and white-screened
  // both /admin and /admin/queue. It went unnoticed because no problem had ever
  // reached the queue against the real API (no resident could be verified, so
  // nothing reached Validated), leaving only the empty branch ever rendered.
  it('renders from the real DTO shape', () => {
    renderRow()
    expect(screen.getByText('রাস্তার বাতিগুলো জ্বলছে না')).toBeInTheDocument()
    expect(screen.getByText(/স্কুল রোড, ওয়ার্ড ২/)).toBeInTheDocument()
  })

  // The official is joined from the cached directory rather than duplicated into
  // every queue row.
  it('resolves the pointed official from the directory', () => {
    renderRow()
    expect(screen.getByText(/করিম উদ্দিন/)).toBeInTheDocument()
  })

  // Every row on this queue is union-routed. Reading `routing` as undefined once
  // silently sent every above-union problem to the union-assign screen, where the
  // backend then refused it with ErrWrongRoute — a decision the UI had no business
  // making. Since B20 the backend does not put those rows here at all: an
  // above-union report is the super admin's to forward, and it surfaces on
  // ForwardingItem instead (A.3.8). So there is one destination and no branch.
  it('routes to the assign screen', () => {
    renderRow({ routing: 'union' })
    expect(screen.getByRole('link')).toHaveAttribute('href', '/admin/problems/prob-2/assign')
  })

  // The label no longer branches either, so it must not have regressed to naming
  // the vote that is gone.
  it('labels the row as a union-level decision, never a vote', () => {
    renderRow()
    expect(screen.getByText('স্থানীয় পর্যায় — প্রশাসক সিদ্ধান্ত নেবেন')).toBeInTheDocument()
    expect(screen.queryByText(/ভোট/)).not.toBeInTheDocument()
  })

  // The count the admin judges by, in Bengali numerals (A.5.3 rule 5). Since B17
  // it gates nothing — a Reported row at 3 of 5 is forwardable right now — so it
  // has to be readable rather than merely present.
  it('shows validation progress in Bengali numerals', () => {
    renderRow()
    const progress = screen.getByText(/জন যাচাই করেছেন/)
    expect(progress).toHaveTextContent('৩ / ৫ জন যাচাই করেছেন')
    expect(progress.textContent).not.toMatch(/[0-9]/)
  })
})
