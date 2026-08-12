import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Oversight from './Oversight.jsx'
import * as identityApi from '../../lib/api/identity.js'
import * as assignmentsApi from '../../lib/api/assignments.js'

vi.mock('../../lib/api/identity.js', () => ({
  listPendingClaims: vi.fn(),
  approveClaim: vi.fn(),
  rejectClaim: vi.fn(),
  listPendingResidents: vi.fn(),
  verifyResident: vi.fn(),
  getOversight: vi.fn(),
}))

// The page reads the forwarding queue too, for the banner. Stub it or the suite
// makes a real HTTP call that quietly fails — which would leave the banner absent
// for the wrong reason and make the assertions below prove nothing (A.5.5).
vi.mock('../../lib/api/assignments.js', () => ({
  listQueue: vi.fn(),
  assignWithinUnion: vi.fn(),
  listMyForwarding: vi.fn(),
  suggestForwarding: vi.fn(),
  listForwardingQueue: vi.fn(),
  forwardProblem: vi.fn(),
}))

const OVERSIGHT = [
  {
    id: 'a1',
    targetType: 'claim',
    targetId: 'claim-1',
    actor: 'acct-seed-admin-1',
    action: 'claim_approved',
    reason: null,
    createdAt: '2026-07-10T09:00:00.000Z',
  },
  {
    id: 'a2',
    targetType: 'account',
    targetId: 'acct-9',
    actor: 'acct-seed-admin-1',
    action: 'verified',
    reason: null,
    createdAt: '2026-07-11T09:00:00.000Z',
  },
]

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Oversight />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(identityApi.getOversight).mockResolvedValue(OVERSIGHT)
  vi.mocked(assignmentsApi.listForwardingQueue).mockResolvedValue([])
})

describe('Oversight', () => {
  it('renders decisions and offers no control to change any of them', async () => {
    renderPage()
    expect(await screen.findByText('কর্মকর্তার দাবি অনুমোদিত হয়েছে')).toBeInTheDocument()
    expect(screen.getByText('বাসিন্দা যাচাই করা হয়েছে')).toBeInTheDocument()
    // Oversight, not override: the whole point of the role is that it acts on
    // nothing. With decisions rendered there must be no button at all — no
    // approve, no reject, no reverse.
    expect(screen.queryByRole('button')).toBeNull()
  })

  // B20 gave the super admin its first action over problems. It lives on its own
  // route, and what appears here is a LINK — so the page whose whole argument is
  // that it holds no controls keeps holding none. If this ever becomes a button
  // that forwards in place, the assertion above stops being true (A.3.8).
  it('links to the forwarding queue without putting a control on the oversight feed', async () => {
    vi.mocked(assignmentsApi.listForwardingQueue).mockResolvedValue([{ problemId: 'prob-9' }, { problemId: 'prob-10' }])
    renderPage()

    const banner = await screen.findByRole('link', { name: /ফরওয়ার্ডের অপেক্ষায়/ })
    expect(banner).toHaveAttribute('href', '/super/queue')
    expect(banner).toHaveTextContent('২টি প্রতিবেদন ফরওয়ার্ডের অপেক্ষায়')
    expect(screen.queryByRole('button')).toBeNull()
  })

  it('shows no forwarding banner when nothing is waiting', async () => {
    renderPage()
    await screen.findByText('কর্মকর্তার দাবি অনুমোদিত হয়েছে')
    expect(screen.queryByText(/ফরওয়ার্ডের অপেক্ষায়/)).not.toBeInTheDocument()
  })

  it('shows an empty state when there are no decisions yet', async () => {
    vi.mocked(identityApi.getOversight).mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('এখনও কোনো সিদ্ধান্ত নেই')).toBeInTheDocument()
  })

  it('offers a retry when the feed cannot be loaded', async () => {
    const user = userEvent.setup()
    vi.mocked(identityApi.getOversight).mockRejectedValue(new Error('down'))
    renderPage()
    expect(await screen.findByText('নজরদারির তালিকা লোড করা যায়নি।')).toBeInTheDocument()

    vi.mocked(identityApi.getOversight).mockResolvedValue(OVERSIGHT)
    await user.click(screen.getByRole('button', { name: 'আবার চেষ্টা করুন' }))
    expect(await screen.findByText('কর্মকর্তার দাবি অনুমোদিত হয়েছে')).toBeInTheDocument()
  })

  it('shows a skeleton while the feed is in flight', () => {
    vi.mocked(identityApi.getOversight).mockReturnValue(new Promise(() => {}))
    renderPage()
    expect(screen.getByRole('status', { name: 'লোড হচ্ছে...' })).toBeInTheDocument()
  })
})
