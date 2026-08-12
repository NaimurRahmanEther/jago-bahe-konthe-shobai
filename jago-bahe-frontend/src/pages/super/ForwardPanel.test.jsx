import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ForwardPanel from './ForwardPanel.jsx'
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
  useOfficials: () => ({
    data: [
      { id: 'off-upz-chair', name: 'আজহার আলী', tier: 'upazila_chairman', areaId: 'upazila-1' },
      { id: 'off-upz-vice', name: 'সোহেল রানা', tier: 'upazila_vice_chairman', areaId: 'upazila-1' },
      { id: 'off-mp', name: 'শহীদুজ্জামান সরকার', tier: 'mp', areaId: 'seat-1' },
      // A union-level office, to prove it is not offered as a forwarding target.
      { id: 'off-chair', name: 'শাহাদত হোসেন', tier: 'union_chairman', areaId: 'union-1' },
    ],
  }),
}))

const suggestion = (id, officialId) => ({
  id,
  adminAccountId: `admin-${id}`,
  officialId,
  createdAt: '2026-07-20T09:00:00.000Z',
})

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
  suggestions: [suggestion('s1', 'off-upz-chair'), suggestion('s2', 'off-upz-chair')],
  topOfficialId: 'off-upz-chair',
  topCount: 2,
  mySuggestion: null,
}

function renderPanel(overrides = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  vi.mocked(assignmentsApi.listForwardingQueue).mockResolvedValue([{ ...ROW, ...overrides }])
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/super/problems/prob-9/forward']}>
        <Routes>
          <Route path="/super/problems/:id/forward" element={<ForwardPanel />} />
          <Route path="/super/queue" element={<p>কিউ</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** Pick an official through the searchable picker. */
async function choose(user, name) {
  await user.click(await screen.findByRole('button', { name: new RegExp(name) }))
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(assignmentsApi.forwardProblem).mockResolvedValue({ id: 'asg-1', problemId: 'prob-9' })
})

describe('ForwardPanel', () => {
  // The pairing IS the decision: the resident asked for X, the seat's admins
  // advise Y, and the super admin answers in public.
  it('shows the reporter’s choice and the advisers’ leading advice side by side', async () => {
    renderPanel()
    expect(await screen.findByText('উপজেলা সড়কের সেতুটি ভেঙে পড়েছে')).toBeInTheDocument()

    const pointed = screen.getByRole('heading', { name: 'প্রতিবেদক যাঁকে দেখিয়েছেন' }).closest('section')
    expect(within(pointed).getByText(/আজহার আলী/)).toBeInTheDocument()

    const advice = screen.getByRole('heading', { name: 'প্রশাসকদের পরামর্শ' }).closest('section')
    expect(within(advice).getByText(/২ জন প্রশাসকের পরামর্শ/)).toBeInTheDocument()
  })

  // Forwarding INTO a union would take a report that union's own admin should have
  // decided and route it through the seat's appointed watcher instead. The backend
  // refuses it either way; the picker simply does not offer it.
  it('offers only above-union offices as targets', async () => {
    renderPanel()
    await screen.findByText('উপজেলা সড়কের সেতুটি ভেঙে পড়েছে')
    expect(screen.getByRole('button', { name: /আজহার আলী/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /শাহাদত হোসেন/ })).not.toBeInTheDocument()
  })

  // Agreeing with BOTH the reporter and the advisers overrides nothing, so no
  // public reason is owed.
  it('forwards without a reason when the choice matches both the reporter and the advisers', async () => {
    const user = userEvent.setup()
    renderPanel()
    await screen.findByText('উপজেলা সড়কের সেতুটি ভেঙে পড়েছে')

    await choose(user, 'আজহার আলী')
    expect(screen.getByText(/\(ঐচ্ছিক\)/)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'ফরওয়ার্ড করুন' }))
    expect(assignmentsApi.forwardProblem).toHaveBeenCalledWith(
      'prob-9',
      expect.objectContaining({ officialId: 'off-upz-chair', reason: '' }),
    )
  })

  // THE LOAD-BEARING CASE. Departing from the advisers' top OR the reporter's
  // choice owes the public a written reason, and the UI must not send the request
  // without one. This mirrors domain.RequiresReason — it does not own the rule
  // (the backend answers 400 regardless), but a screen that lets the request go
  // teaches the super admin the reason is optional.
  it('requires a public reason when the choice departs from both', async () => {
    const user = userEvent.setup()
    renderPanel()
    await screen.findByText('উপজেলা সড়কের সেতুটি ভেঙে পড়েছে')

    await choose(user, 'শহীদুজ্জামান সরকার')
    expect(screen.getByText(/\(আবশ্যক\)/)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'ফরওয়ার্ড করুন' }))
    expect(screen.getByText('এই সিদ্ধান্তের জন্য একটি কারণ লিখুন')).toBeInTheDocument()
    expect(assignmentsApi.forwardProblem).not.toHaveBeenCalled()

    await user.type(screen.getByLabelText(/জনসমক্ষে কারণ/), 'সেতুটি জাতীয় সড়কের অংশ')
    await user.click(screen.getByRole('button', { name: 'ফরওয়ার্ড করুন' }))
    expect(assignmentsApi.forwardProblem).toHaveBeenCalledWith(
      'prob-9',
      expect.objectContaining({ officialId: 'off-mp', reason: 'সেতুটি জাতীয় সড়কের অংশ' }),
    )
  })

  // When the advisers TIE there is no top, which removes that trigger — but the
  // reporter's choice still stands on its own, so departing from it alone is
  // enough. The two triggers are independent.
  it('still requires a reason on a tie when the choice departs from the reporter', async () => {
    const user = userEvent.setup()
    renderPanel({
      suggestions: [suggestion('s1', 'off-upz-chair'), suggestion('s2', 'off-upz-vice')],
      topOfficialId: '',
      topCount: 0,
    })
    await screen.findByText('উপজেলা সড়কের সেতুটি ভেঙে পড়েছে')
    expect(screen.getByText('প্রশাসকেরা একমত হননি')).toBeInTheDocument()

    await choose(user, 'সোহেল রানা')
    expect(screen.getByText(/\(আবশ্যক\)/)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'ফরওয়ার্ড করুন' }))
    expect(assignmentsApi.forwardProblem).not.toHaveBeenCalled()
  })

  // The backend is the authority (A.5.7). If the mirror above ever drifts from
  // domain.RequiresReason, its 400 must still land on the field rather than as a
  // generic failure — that drift is precisely what would otherwise be invisible.
  it('renders the backend’s reason_required refusal as a field error', async () => {
    const user = userEvent.setup()
    vi.mocked(assignmentsApi.forwardProblem).mockRejectedValue({ status: 400, code: 'reason_required' })
    renderPanel()
    await screen.findByText('উপজেলা সড়কের সেতুটি ভেঙে পড়েছে')

    await choose(user, 'আজহার আলী')
    await user.click(screen.getByRole('button', { name: 'ফরওয়ার্ড করুন' }))

    expect(await screen.findByText('এই সিদ্ধান্তের জন্য একটি কারণ লিখুন')).toBeInTheDocument()
  })
})
