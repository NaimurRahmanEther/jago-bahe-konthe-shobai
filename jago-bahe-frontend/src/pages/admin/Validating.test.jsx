import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Validating from './Validating.jsx'
import * as assignmentsApi from '../../lib/api/assignments.js'

vi.mock('../../lib/api/assignments.js', () => ({
  listQueue: vi.fn(),
  assignWithinUnion: vi.fn(),
  listMyForwarding: vi.fn(),
  suggestForwarding: vi.fn(),
  listForwardingQueue: vi.fn(),
  forwardProblem: vi.fn(),
}))

vi.mock('../../hooks/useOfficials.js', () => ({
  useOfficials: () => ({ data: [{ id: 'off-1', name: 'করিম উদ্দিন', tier: 'ward_member', areaId: 'ward-1' }] }),
}))

const row = (problemId, status, validCount, title) => ({
  problemId,
  title,
  status,
  address: 'ওয়ার্ড ২',
  pointedOfficialId: 'off-1',
  areaId: 'ward-2',
  routing: 'union',
  validCount,
  validationThreshold: 5,
})

// One response holding both stages — the page selects its own half out of it.
const QUEUE = [
  row('far', 'Reported', 1, 'ড্রেন উপচে পড়ছে'),
  row('validated', 'Validated', 6, 'পানির লাইন ফেটেছে'),
  row('near', 'Reported', 4, 'রাস্তার বাতি নেই'),
]

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Validating />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => vi.clearAllMocks())

describe('Validating', () => {
  it('lists only the reports still collecting validations', async () => {
    vi.mocked(assignmentsApi.listQueue).mockResolvedValue(QUEUE)
    renderPage()
    expect(await screen.findByText('রাস্তার বাতি নেই')).toBeInTheDocument()
    expect(screen.getByText('ড্রেন উপচে পড়ছে')).toBeInTheDocument()
    // An endorsed problem belongs to the assignment queue, not this one.
    expect(screen.queryByText('পানির লাইন ফেটেছে')).not.toBeInTheDocument()
  })

  // Closest-to-threshold first, so the rows nearest to community endorsement are
  // the ones the admin sees. Ordering a list the server returned is presentation;
  // it decides nothing (A.5.7).
  it('orders rows closest to the threshold first', async () => {
    vi.mocked(assignmentsApi.listQueue).mockResolvedValue(QUEUE)
    renderPage()
    await screen.findByText('রাস্তার বাতি নেই')
    const titles = screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent)
    expect(titles).toEqual(['রাস্তার বাতি নেই', 'ড্রেন উপচে পড়ছে'])
  })

  it('renders a skeleton while loading', () => {
    vi.mocked(assignmentsApi.listQueue).mockReturnValue(new Promise(() => {}))
    const { container } = renderPage()
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument()
  })

  it('renders an empty state when nothing is awaiting validation', async () => {
    vi.mocked(assignmentsApi.listQueue).mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('এই মুহূর্তে যাচাইয়ের অপেক্ষায় কোনো প্রতিবেদন নেই')).toBeInTheDocument()
  })

  it('renders an error state with retry', async () => {
    vi.mocked(assignmentsApi.listQueue).mockRejectedValue(new Error('boom'))
    renderPage()
    expect(await screen.findByText('তালিকা লোড করা যায়নি।')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /আবার/ })).toBeInTheDocument()
  })
})
