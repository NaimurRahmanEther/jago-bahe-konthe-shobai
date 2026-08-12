import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import CaseDetail from './CaseDetail.jsx'

// Fix the route param; the page reads the case id from the URL.
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal()),
  useParams: () => ({ id: 'case-1' }),
}))

const mutationStub = { mutate: vi.fn(), mutateAsync: vi.fn().mockResolvedValue({}), isPending: false }
const useCase = vi.fn()

vi.mock('../../hooks/useCases.js', () => ({
  useCase: (...args) => useCase(...args),
  useAcknowledgeCase: () => mutationStub,
  useSubmitPlan: () => mutationStub,
  useRevisePlan: () => mutationStub,
  usePostUpdate: () => mutationStub,
  useUploadEvidence: () => mutationStub,
  useMarkDone: () => mutationStub,
  useReportObstacle: () => mutationStub,
  useCompleteTask: () => mutationStub,
  toPublicStatus: (s) => s,
}))

vi.mock('../../hooks/useProblems.js', () => ({
  useProblem: () => ({ data: { title: 'Broken footpath', location: { address: 'Ward 3' }, audit: [] } }),
}))

// The working surface pulls in several children with their own data deps; stub the
// heavy ones so this test is purely about the replan affordance.
vi.mock('../../components/resolution/PlanForm.jsx', () => ({ default: () => <div>plan-form</div> }))
vi.mock('../../components/resolution/UpdateTimeline.jsx', () => ({ default: () => <div>timeline</div> }))
vi.mock('../../components/resolution/EvidenceUpload.jsx', () => ({ default: () => <div>evidence</div> }))
vi.mock('../../components/resolution/ObstacleForm.jsx', () => ({ default: () => <div>obstacle-form</div> }))
vi.mock('../../components/resolution/ObstacleJudgment.jsx', () => ({ default: () => <div>obstacle-judgment</div> }))
vi.mock('../../components/resolution/AuditTrail.jsx', () => ({ default: () => <div>audit</div> }))

function makeCase(status) {
  return {
    id: 'case-1',
    problemId: 'prob-1',
    status,
    evidence: [],
    updates: [],
    obstacles: [],
    plan: { id: 'plan-1', strategy: 'x', suggestionResponse: 'y', tasks: [] },
    disputeReason: '',
  }
}

function renderStatus(status) {
  useCase.mockReturnValue({ data: makeCase(status), isLoading: false, isError: false, refetch: vi.fn() })
  render(
    <MemoryRouter>
      <CaseDetail />
    </MemoryRouter>,
  )
}

const REPLAN_OPEN = 'নতুন পরিকল্পনায় পুনরায় শুরু করুন'

describe('CaseDetail replan affordance', () => {
  beforeEach(() => vi.clearAllMocks())

  it('offers restart on an InProgress case', () => {
    renderStatus('InProgress')
    expect(screen.getByRole('button', { name: REPLAN_OPEN })).toBeInTheDocument()
  })

  it('offers restart on a Reopened case', () => {
    renderStatus('Reopened')
    expect(screen.getByRole('button', { name: REPLAN_OPEN })).toBeInTheDocument()
  })

  it('does NOT offer restart on a Blocked case (must be unblocked first)', () => {
    renderStatus('Blocked')
    expect(screen.queryByRole('button', { name: REPLAN_OPEN })).not.toBeInTheDocument()
  })

  it('does NOT offer restart on a Done case', () => {
    renderStatus('Done')
    expect(screen.queryByRole('button', { name: REPLAN_OPEN })).not.toBeInTheDocument()
  })
})
