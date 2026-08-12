import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ProblemFeed from './ProblemFeed.jsx'
import * as problemsApi from '../../lib/api/problems.js'
import * as officialsApi from '../../lib/api/officials.js'
import * as areasApi from '../../lib/api/areas.js'

// The api modules ARE the boundary and these stub them (A.5.5). Every export must
// be listed: useProblems.js does `import * as`, so an omitted name is not an
// import error — it is `undefined` at call time.
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
vi.mock('../../lib/api/officials.js', () => ({ listOfficials: vi.fn() }))
vi.mock('../../lib/api/areas.js', () => ({ listAreas: vi.fn() }))
vi.mock('../../auth/useAuth.js', () => ({ useAuth: () => ({ user: null, role: null }) }))

const problem = (over) => ({
  location: { areaId: 'union-1', address: 'ফতেপুর বাজার' },
  reporterId: 'res-9',
  pointedOfficialId: 'off-1',
  validCount: 0,
  validationThreshold: 5,
  myVote: null,
  ...over,
})

// Two reported in July, one resolved in June. The mix is the point: the status
// chips must recount when the window changes, not just the list.
const PROBLEMS = [
  problem({ id: 'p-1', title: 'জুলাইয়ের প্রথম রিপোর্ট', status: 'Reported', createdAt: '2026-07-03T01:00:00+06:00' }),
  problem({ id: 'p-2', title: 'জুলাইয়ের দ্বিতীয় রিপোর্ট', status: 'Reported', createdAt: '2026-07-20T09:00:00Z' }),
  problem({ id: 'p-3', title: 'জুনের পুরোনো রিপোর্ট', status: 'Resolved', createdAt: '2026-06-10T09:00:00Z' }),
]

function renderFeed() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProblemFeed />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(problemsApi.listProblems).mockResolvedValue(PROBLEMS)
  vi.mocked(officialsApi.listOfficials).mockResolvedValue([])
  vi.mocked(areasApi.listAreas).mockResolvedValue([
    { id: 'union-1', name: 'আড়ানগর ইউনিয়ন', level: 'union', parentId: 'upazila-1' },
  ])
})

describe('ProblemFeed date filter', () => {
  it('narrows the feed to the reports filed in the chosen span', async () => {
    const user = userEvent.setup()
    renderFeed()
    await screen.findByText('জুলাইয়ের প্রথম রিপোর্ট')

    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-07-01')
    await user.type(screen.getByLabelText('শেষ তারিখ'), '2026-07-31')

    expect(screen.getByText('জুলাইয়ের প্রথম রিপোর্ট')).toBeInTheDocument()
    expect(screen.getByText('জুলাইয়ের দ্বিতীয় রিপোর্ট')).toBeInTheDocument()
    expect(screen.queryByText('জুনের পুরোনো রিপোর্ট')).toBeNull()
  })

  // The one ordering decision in this page: date -> counts -> status. If the
  // counts were taken before the window, the chips would advertise reports the
  // list has already dropped — a filter contradicting the thing it filtered.
  it('recounts the status chips against the chosen span', async () => {
    const user = userEvent.setup()
    renderFeed()
    await screen.findByText('জুলাইয়ের প্রথম রিপোর্ট')

    const chips = screen.getByRole('group', { name: 'অবস্থা' })
    expect(within(chips).getByRole('button', { name: /সমাধান হয়েছে/ })).toBeInTheDocument()

    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-07-01')

    // The only Resolved report is June's, so its chip goes with it — a chip that
    // filters to nothing is a dead end, which is why only present statuses show.
    expect(within(chips).queryByRole('button', { name: /সমাধান হয়েছে/ })).toBeNull()
    expect(within(chips).getByRole('button', { name: /সব অবস্থা/ })).toHaveTextContent('২')
  })

  // The local-day boundary, end to end. p-1 is timestamped 01:00 on 3 July in
  // Dhaka; compared against UTC midnight it would fall OUTSIDE a range starting
  // that day, and would silently vanish from the feed.
  it('keeps a report filed in the small hours of the from-day', async () => {
    const user = userEvent.setup()
    renderFeed()
    await screen.findByText('জুলাইয়ের প্রথম রিপোর্ট')

    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-07-03')
    await user.type(screen.getByLabelText('শেষ তারিখ'), '2026-07-03')

    expect(screen.getByText('জুলাইয়ের প্রথম রিপোর্ট')).toBeInTheDocument()
  })

  it('says the span is empty rather than "no reports yet"', async () => {
    const user = userEvent.setup()
    renderFeed()
    await screen.findByText('জুলাইয়ের প্রথম রিপোর্ট')

    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-01-01')
    await user.type(screen.getByLabelText('শেষ তারিখ'), '2026-01-31')

    expect(await screen.findByText('এই সময়ের মধ্যে কিছু নেই')).toBeInTheDocument()
  })

  // The date range is applied here, over rows the server already returned whole.
  // If it ever became a query param the feed would refetch — and the endpoint has
  // no from/to, so the param would be silently ignored and the filter would stop
  // working. One call, whatever the reader picks.
  it('filters without asking the server again', async () => {
    const user = userEvent.setup()
    renderFeed()
    await screen.findByText('জুলাইয়ের প্রথম রিপোর্ট')
    const before = vi.mocked(problemsApi.listProblems).mock.calls.length

    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-07-01')

    expect(vi.mocked(problemsApi.listProblems).mock.calls.length).toBe(before)
  })
})
