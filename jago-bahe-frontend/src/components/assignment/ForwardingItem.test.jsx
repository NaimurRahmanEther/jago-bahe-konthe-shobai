import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import ForwardingItem from './ForwardingItem.jsx'

vi.mock('../../hooks/useOfficials.js', () => ({
  useOfficials: () => ({
    data: [
      { id: 'off-upz-chair', name: 'আজহার আলী', tier: 'upazila_chairman', areaId: 'upazila-1' },
      { id: 'off-upz-vice', name: 'সোহেল রানা', tier: 'upazila_vice_chairman', areaId: 'upazila-1' },
      { id: 'off-mp', name: 'শহীদুজ্জামান সরকার', tier: 'mp', areaId: 'seat-1' },
    ],
  }),
}))

// Mirrors forwardingItemDTO field-for-field.
const ROW = {
  problemId: 'prob-9',
  title: 'উপজেলা সড়কের সেতুটি ভেঙে পড়েছে',
  status: 'Validated',
  address: 'উপজেলা সড়ক সেতু',
  areaId: 'union-1',
  pointedOfficialId: 'off-upz-chair',
  scope: 'upazila',
  validCount: 6,
  validationThreshold: 5,
  suggestions: [],
  topOfficialId: '',
  topCount: 0,
  mySuggestion: null,
}

const suggestion = (id, officialId) => ({
  id,
  adminAccountId: `admin-${id}`,
  officialId,
  createdAt: '2026-07-20T09:00:00.000Z',
})

const renderRow = (overrides = {}, href = '/admin/forwarding/prob-9') =>
  render(
    <MemoryRouter>
      <ForwardingItem item={{ ...ROW, ...overrides }} href={href} />
    </MemoryRouter>,
  )

describe('ForwardingItem', () => {
  it('renders from the real DTO shape', () => {
    renderRow()
    expect(screen.getByText('উপজেলা সড়কের সেতুটি ভেঙে পড়েছে')).toBeInTheDocument()
    expect(screen.getByText(/উপজেলা সড়ক সেতু/)).toBeInTheDocument()
  })

  // The reporter's choice is one of the two things a forward is weighed against,
  // so it belongs on the row rather than only in the panel.
  it('names the official the reporter pointed the report at', () => {
    renderRow()
    expect(screen.getByText(/আজহার আলী/)).toBeInTheDocument()
  })

  // The same component serves both surfaces; only the destination differs, and it
  // comes from the caller so the row never has to know who is looking at it.
  it('links wherever the caller sends it', () => {
    renderRow({}, '/super/problems/prob-9/forward')
    expect(screen.getByRole('link')).toHaveAttribute('href', '/super/problems/prob-9/forward')
  })

  it('says plainly when nobody has advised yet', () => {
    renderRow()
    expect(screen.getByText('এখনও কোনো প্রশাসক পরামর্শ দেননি')).toBeInTheDocument()
  })

  it('names the leading advice and the count in Bengali numerals', () => {
    renderRow({
      suggestions: [suggestion('s1', 'off-upz-chair'), suggestion('s2', 'off-upz-chair'), suggestion('s3', 'off-mp')],
      topOfficialId: 'off-upz-chair',
      topCount: 2,
    })
    const tally = screen.getByText(/জনের মধ্যে/)
    expect(tally).toHaveTextContent('৩ জনের মধ্যে ২ জন বলেছেন আজহার আলী')
    expect(tally.textContent).not.toMatch(/[0-9]/)
  })

  // THE LOAD-BEARING CASE. A tie yields topOfficialId '' with a non-empty
  // suggestions list, and those two states must read differently: "nobody advised"
  // and "the advisers disagree" are opposite facts. Collapsing them — by treating
  // '' as "no advice", or by breaking the tie to name a winner — would tell the
  // super admin the seat is silent when it is in fact split (A.3.8).
  it('reports a tie as disagreement, not as an absence of advice', () => {
    renderRow({
      suggestions: [suggestion('s1', 'off-upz-chair'), suggestion('s2', 'off-upz-vice')],
      topOfficialId: '',
      topCount: 0,
    })
    expect(screen.getByText('২ জন পরামর্শ দিয়েছেন — কেউ এগিয়ে নেই')).toBeInTheDocument()
    expect(screen.queryByText('এখনও কোনো প্রশাসক পরামর্শ দেননি')).not.toBeInTheDocument()
  })
})
