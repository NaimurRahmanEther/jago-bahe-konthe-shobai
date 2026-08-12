import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ClaimsReview from './ClaimsReview.jsx'
import * as identityApi from '../../lib/api/identity.js'

// The api module is the boundary; the test stubs it (A.5.5). useClaims does
// `import * as identityApi`, so every export must be listed or a missing name is
// `undefined` at call time, not an import error.
vi.mock('../../lib/api/identity.js', () => ({
  listPendingClaims: vi.fn(),
  approveClaim: vi.fn(),
  rejectClaim: vi.fn(),
  listPendingResidents: vi.fn(),
  verifyResident: vi.fn(),
  getOversight: vi.fn(),
}))

const PENDING_CLAIMS = [
  {
    id: 'claim-1',
    accountId: 'acct-a',
    officialId: 'off-2',
    status: 'Pending',
    createdAt: '2026-07-10T09:00:00.000Z',
    officialName: 'সালমা বেগম',
    officialTier: 'women_member',
    areaId: 'union-1',
    claimantName: 'জাহিদ হাসান',
    claimantNid: '1990123456789',
  },
  {
    id: 'claim-2',
    accountId: 'acct-b',
    officialId: 'off-6',
    status: 'Pending',
    createdAt: '2026-07-11T09:00:00.000Z',
    officialName: 'নূরুল আমিন',
    officialTier: 'union_chairman',
    areaId: 'union-2',
    claimantName: 'রোকসানা বেগম',
  },
]

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ClaimsReview />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(identityApi.listPendingClaims).mockResolvedValue(PENDING_CLAIMS)
  vi.mocked(identityApi.approveClaim).mockResolvedValue({ ...PENDING_CLAIMS[0], status: 'Approved' })
  vi.mocked(identityApi.rejectClaim).mockResolvedValue({ ...PENDING_CLAIMS[0], status: 'Rejected' })
})

describe('ClaimsReview', () => {
  it('lists pending claims with the claimant, office, and tier', async () => {
    renderPage()
    expect(await screen.findByText('জাহিদ হাসান')).toBeInTheDocument()
    expect(screen.getByText('রোকসানা বেগম')).toBeInTheDocument()
    // Tier renders from problem.tier.* — the office, not a role the claimant named.
    expect(screen.getByText('নারী সদস্য')).toBeInTheDocument()
  })

  it('approves a claim by its id', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('জাহিদ হাসান')
    const approveButtons = screen.getAllByRole('button', { name: 'অনুমোদন করুন' })
    await user.click(approveButtons[0])
    expect(identityApi.approveClaim).toHaveBeenCalledWith('claim-1')
  })

  it('requires a reason before rejecting, then rejects with it', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('জাহিদ হাসান')

    const rejectButtons = screen.getAllByRole('button', { name: 'প্রত্যাখ্যান করুন' })
    await user.click(rejectButtons[0])

    // Confirm with no reason: the mutation must not fire.
    await user.click(screen.getByRole('button', { name: 'প্রত্যাখ্যান নিশ্চিত করুন' }))
    expect(screen.getByText('একটি কারণ লিখুন।')).toBeInTheDocument()
    expect(identityApi.rejectClaim).not.toHaveBeenCalled()

    await user.type(screen.getByRole('textbox'), 'দপ্তরের সাথে তথ্য মেলেনি')
    await user.click(screen.getByRole('button', { name: 'প্রত্যাখ্যান নিশ্চিত করুন' }))
    // The api takes the id positionally and the reason in the body — rejectClaim(id, {reason}).
    expect(identityApi.rejectClaim).toHaveBeenCalledWith('claim-1', { reason: 'দপ্তরের সাথে তথ্য মেলেনি' })
  })

  it('invites nothing when there are no claims to review', async () => {
    vi.mocked(identityApi.listPendingClaims).mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('পর্যালোচনার জন্য কোনো দাবি নেই')).toBeInTheDocument()
  })

  it('offers a retry when the claims cannot be loaded', async () => {
    const user = userEvent.setup()
    vi.mocked(identityApi.listPendingClaims).mockRejectedValue(new Error('down'))
    renderPage()
    expect(await screen.findByText('দাবির তালিকা লোড করা যায়নি।')).toBeInTheDocument()

    vi.mocked(identityApi.listPendingClaims).mockResolvedValue(PENDING_CLAIMS)
    await user.click(screen.getByRole('button', { name: 'আবার চেষ্টা করুন' }))
    expect(await screen.findByText('জাহিদ হাসান')).toBeInTheDocument()
  })

  it('shows a skeleton while the claims are in flight', () => {
    vi.mocked(identityApi.listPendingClaims).mockReturnValue(new Promise(() => {}))
    renderPage()
    expect(screen.getByRole('status', { name: 'লোড হচ্ছে...' })).toBeInTheDocument()
  })
})
