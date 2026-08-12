import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Residents from './Residents.jsx'
import * as identityApi from '../../lib/api/identity.js'

vi.mock('../../lib/api/identity.js', () => ({
  listPendingClaims: vi.fn(),
  approveClaim: vi.fn(),
  rejectClaim: vi.fn(),
  listPendingResidents: vi.fn(),
  verifyResident: vi.fn(),
  getOversight: vi.fn(),
}))

const PENDING_RESIDENTS = [
  { id: 'acct-1', name: 'রহিমা খাতুন', phone: '01810000021', role: 'resident', nid: '1234567890111', verified: false },
  { id: 'acct-2', name: 'সেলিম মিয়া', phone: '01810000022', role: 'resident', nid: '1234567890222', verified: false },
]

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Residents />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(identityApi.listPendingResidents).mockResolvedValue(PENDING_RESIDENTS)
  vi.mocked(identityApi.verifyResident).mockResolvedValue({ verified: true })
})

describe('Residents', () => {
  it('lists the union’s unverified residents', async () => {
    renderPage()
    expect(await screen.findByText('রহিমা খাতুন')).toBeInTheDocument()
    expect(screen.getByText('সেলিম মিয়া')).toBeInTheDocument()
  })

  it('verifies a resident by their account id', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('রহিমা খাতুন')
    const verifyButtons = screen.getAllByRole('button', { name: 'যাচাই করুন' })
    await user.click(verifyButtons[0])
    expect(identityApi.verifyResident).toHaveBeenCalledWith('acct-1')
  })

  it('shows an empty state when no one is awaiting verification', async () => {
    vi.mocked(identityApi.listPendingResidents).mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('যাচাইয়ের অপেক্ষায় কোনো বাসিন্দা নেই')).toBeInTheDocument()
  })

  it('offers a retry when the list cannot be loaded', async () => {
    const user = userEvent.setup()
    vi.mocked(identityApi.listPendingResidents).mockRejectedValue(new Error('down'))
    renderPage()
    expect(await screen.findByText('বাসিন্দাদের তালিকা লোড করা যায়নি।')).toBeInTheDocument()

    vi.mocked(identityApi.listPendingResidents).mockResolvedValue(PENDING_RESIDENTS)
    await user.click(screen.getByRole('button', { name: 'আবার চেষ্টা করুন' }))
    expect(await screen.findByText('রহিমা খাতুন')).toBeInTheDocument()
  })

  it('shows a skeleton while the list is in flight', () => {
    vi.mocked(identityApi.listPendingResidents).mockReturnValue(new Promise(() => {}))
    renderPage()
    expect(screen.getByRole('status', { name: 'লোড হচ্ছে...' })).toBeInTheDocument()
  })
})
