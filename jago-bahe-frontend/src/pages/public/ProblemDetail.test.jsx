import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ProblemDetail from './ProblemDetail.jsx'
import * as problemsApi from '../../lib/api/problems.js'
import * as resolutionApi from '../../lib/api/resolution.js'
import * as suggestionsApi from '../../lib/api/suggestions.js'
import * as officialsApi from '../../lib/api/officials.js'

// The api modules ARE the boundary (A.5.5); these stub them. Every export must be
// listed — the hooks `import * as`, so an omitted name is not an import error, it is
// `undefined` at call time.
vi.mock('../../lib/api/problems.js', () => ({
  listProblems: vi.fn(),
  getProblem: vi.fn(),
  reportProblem: vi.fn(),
  listMyProblems: vi.fn(),
  validateProblem: vi.fn(),
  updateProblem: vi.fn(),
  withdrawProblem: vi.fn(),
  deleteProblem: vi.fn(),
}))
vi.mock('../../lib/api/resolution.js', () => ({
  listCases: vi.fn(),
  getProgress: vi.fn(),
  getCase: vi.fn(),
  acknowledgeCase: vi.fn(),
  submitPlan: vi.fn(),
  postUpdate: vi.fn(),
  uploadEvidence: vi.fn(),
  markDone: vi.fn(),
  reportObstacle: vi.fn(),
  confirmProblem: vi.fn(),
}))
vi.mock('../../lib/api/suggestions.js', () => ({
  listSuggestions: vi.fn(),
  proposeSuggestion: vi.fn(),
  upvoteSuggestion: vi.fn(),
}))
vi.mock('../../lib/api/officials.js', () => ({ listOfficials: vi.fn() }))

vi.mock('../../auth/useAuth.js', () => ({
  useAuth: () => ({ user: { id: 'res-1', verified: true }, role: 'resident', updateUser: vi.fn() }),
}))

const problem = (over = {}) => ({
  id: 'p1',
  title: 'রাস্তার বাতি নষ্ট',
  description: 'তিন সপ্তাহ ধরে অন্ধকার।',
  status: 'InProgress',
  location: { areaId: 'union-1', address: 'ওয়ার্ড ৩' },
  reporterId: 'res-9',
  pointedOfficialId: 'off-1',
  assignedOfficialId: 'off-1',
  validCount: 3,
  myVote: null,
  audit: [{ id: 'a1', actor: 'res-9', action: 'reported', createdAt: '2026-01-02T10:00:00Z' }],
  ...over,
})

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/problems/p1']}>
        <Routes>
          <Route path="/problems/:id" element={<ProblemDetail />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  problemsApi.getProblem.mockResolvedValue(problem())
  suggestionsApi.listSuggestions.mockResolvedValue([])
  officialsApi.listOfficials.mockResolvedValue([{ id: 'off-1', name: 'করিম উদ্দিন', tier: 'union_chairman' }])
  resolutionApi.getProgress.mockResolvedValue(null)
})

describe('ProblemDetail structure', () => {
  // THE regression pin for the four-act redesign. The page used to be a flat column
  // of 14 siblings, four of which wore the byte-identical card wrapper and titled
  // themselves with `<p className="text-sm font-semibold">` — not headings, so the
  // page had no outline below its <h1> and nothing separated one idea from the next.
  it('organises the page into four named regions under one h1', async () => {
    renderPage()
    expect(await screen.findByRole('heading', { level: 1, name: 'রাস্তার বাতি নষ্ট' })).toBeInTheDocument()

    const headings = screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)
    expect(headings).toEqual([
      'যাচাই',
      expect.stringContaining('প্রস্তাবিত সমাধান'),
      'কর্তৃপক্ষের পদক্ষেপ',
      'সম্পূর্ণ রেকর্ড',
    ])
  })

  // Proposals BEFORE the authority's reply: the community proposes, the official
  // answers. The old order put the answer above the question.
  it('places the proposals ahead of the authority section', async () => {
    renderPage()
    await screen.findByRole('heading', { level: 1 })
    const headings = screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)
    const proposals = headings.findIndex((h) => h.includes('প্রস্তাবিত সমাধান'))
    const authority = headings.indexOf('কর্তৃপক্ষের পদক্ষেপ')
    expect(proposals).toBeLessThan(authority)
  })

  // Every section title rhymes with something else on the page in Bangla, so regions
  // must be addressable by accessible name — this is A.5.3 rule 4 made executable.
  it('exposes each section as a named region', async () => {
    renderPage()
    await screen.findByRole('heading', { level: 1 })
    for (const name of ['যাচাই', 'কর্তৃপক্ষের পদক্ষেপ', 'সম্পূর্ণ রেকর্ড']) {
      expect(screen.getByRole('region', { name })).toBeInTheDocument()
    }
  })

  // The reporter's own proposed fix used to sit in the header at 14px muted — the
  // person who found the problem was the quietest voice about fixing it. It now sits
  // with the community's proposals.
  it('groups the reporter’s own proposal with the community’s', async () => {
    problemsApi.getProblem.mockResolvedValue(problem({ proposedSolution: 'নতুন বাতি লাগানো হোক' }))
    renderPage()
    await screen.findByRole('heading', { level: 1 })
    const region = screen.getByRole('region', { name: /প্রস্তাবিত সমাধান/ })
    expect(within(region).getByText('নতুন বাতি লাগানো হোক')).toBeInTheDocument()
  })

  // Unassigned reports still get the section: on a public accountability page,
  // "no official has taken this up" is a fact worth publishing, not an absence.
  it('states plainly when no authority has acted yet', async () => {
    problemsApi.getProblem.mockResolvedValue(
      problem({ status: 'Reported', assignedOfficialId: undefined }),
    )
    renderPage()
    await screen.findByRole('heading', { level: 1 })
    const region = screen.getByRole('region', { name: 'কর্তৃপক্ষের পদক্ষেপ' })
    expect(within(region).getByText(/এখনো কোনো কর্মকর্তাকে/)).toBeInTheDocument()
  })

  it('collapses the audit trail behind a disclosure that states its length', async () => {
    renderPage()
    await screen.findByRole('heading', { level: 1 })
    const region = screen.getByRole('region', { name: 'সম্পূর্ণ রেকর্ড' })
    // Bengali numerals, not Latin (A.5.3 rule 5).
    expect(within(region).getByText('সব কার্যক্রম দেখুন (১টি)')).toBeInTheDocument()
    expect(within(region).queryByText('সব কার্যক্রম দেখুন (1টি)')).toBeNull()
  })
})
