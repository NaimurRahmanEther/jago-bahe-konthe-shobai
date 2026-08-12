import { describe, it, expect } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import ProgressPanel from './ProgressPanel.jsx'

// A minimal public-progress payload with a two-week checklist, one week done.
const progress = {
  caseId: 'case-1',
  problemId: 'prob-1',
  status: 'InProgress',
  createdAt: '2026-07-01T00:00:00.000Z',
  deadline: '2026-08-01T00:00:00.000Z',
  acknowledged: true,
  acknowledgedAt: '2026-07-02T00:00:00.000Z',
  plan: {
    id: 'plan-1',
    strategy: 'Phased repair',
    timelineWeeks: 2,
    obstacles: '',
    suggestionResponse: 'Adopting the culvert idea',
    answeredSuggestion: 'Build a culvert',
    tasks: [
      { id: 't1', weekNumber: 1, task: 'Clear the drain inlet', completed: true, completedAt: '2026-07-08T00:00:00.000Z' },
      { id: 't2', weekNumber: 2, task: 'Lay the new pipe', completed: false, completedAt: null },
    ],
  },
  updates: [],
  evidence: [],
  blockedOnHigherAuthority: false,
}

describe('ProgressPanel weekly checklist', () => {
  it('renders each week task with its completion state', () => {
    render(<ProgressPanel progress={progress} />)

    expect(screen.getByText('সাপ্তাহিক কাজ')).toBeInTheDocument()

    const doneItem = screen.getByText('Clear the drain inlet').closest('li')
    const pendingItem = screen.getByText('Lay the new pipe').closest('li')

    // Never colour alone (A.6): a completed week carries a screen-reader "সম্পন্ন",
    // a pending one "বাকি আছে".
    expect(within(doneItem).getByText('সম্পন্ন')).toBeInTheDocument()
    expect(within(pendingItem).getByText('বাকি আছে')).toBeInTheDocument()

    // The completed week is struck through; the pending one is not.
    expect(doneItem.querySelector('.line-through')).not.toBeNull()
    expect(pendingItem.querySelector('.line-through')).toBeNull()
  })
})

describe('ProgressPanel obstacle history (cause of not solving)', () => {
  // A resolved (denied) obstacle on a case that is back at InProgress: the point
  // is that it PERSISTS on the public record after the case left Blocked.
  const withResolvedObstacle = {
    ...progress,
    status: 'InProgress',
    obstacles: [
      {
        id: 'obs-1',
        caseId: 'case-1',
        category: 'budget',
        whatBlocks: 'No allocation this quarter',
        whoUnblocks: 'Upazila engineer',
        proofTried: 'Wrote to the UNO twice',
        adjudication: 'denied',
        realCount: 2,
        notConvincedCount: 1,
        unblockingPlans: [],
        adjudicatedAt: '2026-07-10T00:00:00.000Z',
        createdAt: '2026-07-05T00:00:00.000Z',
        resolvedAt: '2026-07-10T00:00:00.000Z',
      },
    ],
  }

  it('shows the obstacle cause and verdict even when the case is no longer Blocked', () => {
    render(<ProgressPanel progress={withResolvedObstacle} />)

    const heading = screen.getByRole('heading', { name: 'কী বাধা দিয়েছে' })
    const section = heading.closest('div')

    expect(within(section).getByText('No allocation this quarter')).toBeInTheDocument()
    // The named category and the adjudication verdict both render.
    expect(within(section).getByText('বাজেট/বরাদ্দ')).toBeInTheDocument()
    expect(within(section).getByText('বাধাটি কর্মকর্তার আওতাধীন বলে রায় হয়েছে')).toBeInTheDocument()
  })

  it('renders no obstacle section when there are none', () => {
    render(<ProgressPanel progress={progress} />)
    expect(screen.queryByRole('heading', { name: 'কী বাধা দিয়েছে' })).not.toBeInTheDocument()
  })
})
