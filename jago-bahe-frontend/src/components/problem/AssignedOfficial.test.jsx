import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import AssignedOfficial from './AssignedOfficial.jsx'

const h = vi.hoisted(() => ({ officials: [] }))
vi.mock('../../hooks/useOfficials.js', () => ({ useOfficials: () => ({ data: h.officials }) }))

const problem = (over = {}) => ({
  id: 'p1',
  pointedOfficialId: 'off-1',
  assignedOfficialId: 'off-1',
  ...over,
})

beforeEach(() => {
  h.officials = [
    { id: 'off-1', name: 'আবুল কালাম', tier: 'union_chairman', areaId: 'union-1' },
    { id: 'off-2', name: 'রফিকুল ইসলাম', tier: 'upazila_chairman', areaId: 'upazila-1' },
  ]
})

describe('AssignedOfficial', () => {
  it('names the official actually working on the report', () => {
    render(<AssignedOfficial problem={problem()} />)
    expect(screen.getByText('আবুল কালাম')).toBeInTheDocument()
    expect(screen.getByText('ইউনিয়ন চেয়ারম্যান')).toBeInTheDocument()
  })

  // An unassigned report must render nothing at all — an "unassigned" strip on
  // every new report would be noise, and the status badge already says as much.
  it('renders nothing before the report is assigned', () => {
    const { container } = render(<AssignedOfficial problem={problem({ assignedOfficialId: undefined })} />)
    expect(container).toBeEmptyDOMElement()
  })

  // The accountability case. Publishing the pointed/assigned gap while withholding
  // the admin's reason would invite the reader to assume the worst of a decision
  // that is usually routine, so the two are shown together or not at all.
  it('shows the override with the public choice and the admin reason', () => {
    render(
      <AssignedOfficial
        problem={problem({
          pointedOfficialId: 'off-1',
          assignedOfficialId: 'off-2',
          overrideReason: 'এটি উপজেলা পরিষদের এখতিয়ারভুক্ত',
        })}
      />,
    )
    expect(screen.getByText('রফিকুল ইসলাম')).toBeInTheDocument()
    expect(screen.getByText(/জনগণ নির্দেশ করেছিলেন: আবুল কালাম/)).toBeInTheDocument()
    expect(screen.getByText(/এটি উপজেলা পরিষদের এখতিয়ারভুক্ত/)).toBeInTheDocument()
  })

  // No override happened, so there is nothing to justify and no reason to imply
  // one was needed.
  it('says nothing about an override when the admin confirmed the public choice', () => {
    render(<AssignedOfficial problem={problem()} />)
    expect(screen.queryByText(/জনগণ নির্দেশ করেছিলেন/)).toBeNull()
  })

  // The directory load is independent of the problem load, so the id can arrive
  // first. Saying "an official" is true; rendering blank would read as "nobody",
  // which is the opposite of the fact.
  it('still reports that someone is assigned when the directory has not loaded', () => {
    h.officials = undefined
    render(<AssignedOfficial problem={problem()} />)
    expect(screen.getByText('একজন কর্মকর্তা')).toBeInTheDocument()
  })
})
