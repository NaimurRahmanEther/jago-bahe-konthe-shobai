import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Observation from './Observation.jsx'
import * as observationApi from '../../lib/api/observation.js'

vi.mock('../../lib/api/observation.js', () => ({
  listObservations: vi.fn(),
  noteObservation: vi.fn(),
}))

let mockUser = { id: 'acct-1', officialId: 'off-upazila' }
vi.mock('../../auth/useAuth.js', () => ({
  useAuth: () => ({ user: mockUser, role: 'official' }),
}))

const observation = (id, level, resolvedAt = null, notes = []) => ({
  id,
  level,
  openedAt: '2026-07-18T00:00:00Z',
  resolvedAt,
  notes,
})

const row = (caseId, { silentDays = 0, observations = [], title, direct = true } = {}) => ({
  caseId,
  problemId: `prob-${caseId}`,
  problemTitle: title,
  officialId: 'off-union',
  officialName: 'করিম উদ্দিন',
  status: 'Assigned',
  deadline: '2026-07-18T00:00:00Z',
  lastActivityAt: null,
  silentDays,
  direct,
  observations,
})

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Observation />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockUser = { id: 'acct-1', officialId: 'off-upazila' }
})

describe('Observation', () => {
  // Concept §7: monitoring is continuous from assignment, so a healthy case is
  // listed too — just not as something anybody is waiting on.
  it('separates the cases nobody has answered from the ones merely being watched', async () => {
    vi.mocked(observationApi.listObservations).mockResolvedValue([
      row('silent', { silentDays: 8, observations: [observation('obs-1', 1)], title: 'ড্রেন উপচে পড়ছে' }),
      row('healthy', { title: 'রাস্তার বাতি নেই' }),
    ])
    renderPage()

    const waiting = await screen.findByRole('region', { name: 'সাড়া পাওয়া যায়নি' })
    expect(within(waiting).getByText('ড্রেন উপচে পড়ছে')).toBeInTheDocument()
    expect(within(waiting).queryByText('রাস্তার বাতি নেই')).not.toBeInTheDocument()

    const watching = screen.getByRole('region', { name: 'নজরে আছে' })
    expect(within(watching).getByText('রাস্তার বাতি নেই')).toBeInTheDocument()
  })

  // Bengali numerals, not Latin (A.5.3 rule 5).
  it('spells the silence out in days, in Bengali digits', async () => {
    vi.mocked(observationApi.listObservations).mockResolvedValue([
      row('silent', { silentDays: 8, observations: [observation('obs-1', 1)], title: 'ড্রেন উপচে পড়ছে' }),
    ])
    renderPage()
    expect(await screen.findByText('৮ দিন ধরে সাড়া নেই')).toBeInTheDocument()
    expect(screen.queryByText(/8 দিন/)).not.toBeInTheDocument()
  })

  // The monitor is not the case's owner: /official/cases/{id} answers 403 for
  // them, and everything they need is public on the problem page.
  it('links a row to the public problem page, never to the official case page', async () => {
    vi.mocked(observationApi.listObservations).mockResolvedValue([
      row('silent', { silentDays: 8, observations: [observation('obs-1', 1)], title: 'ড্রেন উপচে পড়ছে' }),
    ])
    renderPage()
    const link = await screen.findByRole('link', { name: 'ড্রেন উপচে পড়ছে' })
    expect(link).toHaveAttribute('href', '/problems/prob-silent')
  })

  it('records a note against the open observation', async () => {
    vi.mocked(observationApi.listObservations).mockResolvedValue([
      row('silent', { silentDays: 8, observations: [observation('obs-1', 1)], title: 'ড্রেন উপচে পড়ছে' }),
    ])
    vi.mocked(observationApi.noteObservation).mockResolvedValue({ id: 'obsn-1', text: 'x', createdAt: '2026-07-26T00:00:00Z' })
    renderPage()

    await screen.findByText('ড্রেন উপচে পড়ছে')
    await userEvent.type(screen.getByLabelText('আপনি কী পদক্ষেপ নিয়েছেন'), 'চেয়ারম্যানকে মনে করিয়ে দিয়েছি')
    await userEvent.click(screen.getByRole('button', { name: 'লিখে রাখুন' }))

    expect(observationApi.noteObservation).toHaveBeenCalledWith('obs-1', {
      text: 'চেয়ারম্যানকে মনে করিয়ে দিয়েছি',
    })
  })

  // A note answers an escalation. There is nothing to answer on a case nobody is
  // waiting on, and offering the box there would invite a 404.
  it('offers no note box on a case nobody is waiting on', async () => {
    vi.mocked(observationApi.listObservations).mockResolvedValue([row('healthy', { title: 'রাস্তার বাতি নেই' })])
    renderPage()
    await screen.findByText('রাস্তার বাতি নেই')
    expect(screen.queryByLabelText('আপনি কী পদক্ষেপ নিয়েছেন')).not.toBeInTheDocument()
  })

  // Decision 2: a resolved observation is history, not deletion — the pattern has
  // to stay visible after the official finally answers.
  it('keeps a resolved silence on the record', async () => {
    vi.mocked(observationApi.listObservations).mockResolvedValue([
      row('answered', {
        observations: [observation('obs-1', 1, '2026-07-20T00:00:00Z')],
        title: 'পানির লাইন ফেটেছে',
      }),
    ])
    renderPage()
    expect(await screen.findByText('এর আগে ১ বার সাড়া না পেয়ে এই কেস উপরে এসেছিল')).toBeInTheDocument()
  })

  it('renders the three states', async () => {
    vi.mocked(observationApi.listObservations).mockResolvedValue([])
    const { unmount } = renderPage()
    expect(await screen.findByText('এখন কোনো কেস আপনার পর্যবেক্ষণে নেই')).toBeInTheDocument()
    unmount()

    vi.mocked(observationApi.listObservations).mockRejectedValue(new Error('boom'))
    renderPage()
    expect(await screen.findByText('পর্যবেক্ষণের তালিকা লোড করা যায়নি।')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'আবার চেষ্টা করুন' })).toBeInTheDocument()
  })

  // An official whose claim is not yet approved has no officialId, so the query
  // never runs; an empty page would read as "nobody has gone quiet".
  it('says the claim is pending rather than showing an empty list', async () => {
    mockUser = { id: 'acct-1', officialId: null }
    renderPage()
    expect(await screen.findByText('আপনার দাবি অনুমোদনের অপেক্ষায়')).toBeInTheDocument()
    expect(observationApi.listObservations).not.toHaveBeenCalled()
  })
})
