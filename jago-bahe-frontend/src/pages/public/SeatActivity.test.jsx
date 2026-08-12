import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import SeatActivity from './SeatActivity.jsx'
import * as activityApi from '../../lib/api/activity.js'

vi.mock('../../lib/api/activity.js', () => ({ listActivity: vi.fn() }))

const entry = (over = {}) => ({
  id: 'audit-1',
  action: 'approved',
  actorId: 'acct-admin-aranagar',
  actorName: 'প্রশাসক — আড়ানগর ইউনিয়ন',
  targetType: 'problem',
  targetId: 'prob-1',
  problemTitle: 'ফতেপুর বাজার সড়ক উন্নয়ন',
  createdAt: '2026-07-19T00:05:41Z',
  ...over,
})

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <SeatActivity />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  activityApi.listActivity.mockResolvedValue([entry()])
})

describe('SeatActivity', () => {
  // The point of the page: a resident, signed in or not, can see who decided what.
  it('names the acting admin and links to the report', async () => {
    renderPage()
    expect(await screen.findByText('প্রশাসক — আড়ানগর ইউনিয়ন')).toBeInTheDocument()
    const link = screen.getByRole('link', { name: 'ফতেপুর বাজার সড়ক উন্নয়ন' })
    expect(link).toHaveAttribute('href', '/problems/prob-1')
  })

  // Publishing the record widens who oversees; it does not add a power to override
  // (Scaffold §2). If a control ever appears over the record, that rule has been
  // broken.
  //
  // This used to assert the whole PAGE held no button, and the date filter's
  // preset chips are buttons. The assertion was narrowed rather than deleted, and
  // narrowed to the thing it was actually protecting: the record itself. A filter
  // changes what this reader sees and decides nothing about any report — so it
  // lives outside the region, and the exemption is named below rather than left
  // as a hole a real control could later slip through.
  it('offers no control over the record', async () => {
    renderPage()
    await screen.findByText('প্রশাসক — আড়ানগর ইউনিয়ন')
    const region = screen.getByRole('region', { name: /সাম্প্রতিক সিদ্ধান্ত/ })
    expect(within(region).queryAllByRole('button')).toHaveLength(0)
  })

  it('offers only the date filter, and nothing else, anywhere on the page', async () => {
    renderPage()
    await screen.findByText('প্রশাসক — আড়ানগর ইউনিয়ন')
    expect(screen.queryAllByRole('button').map((b) => b.textContent)).toEqual([
      'সব সময়',
      'আজ',
      'গত ৭ দিন',
      'গত ৩০ দিন',
      'এই মাস',
    ])
  })

  // The server withholds the person-actions, but if it ever regressed the UI must
  // not be the thing that makes the leak legible. This asserts the page renders
  // what it is given without inventing person-facing labels for it.
  it('renders an unresolved actor id rather than dropping the row', async () => {
    activityApi.listActivity.mockResolvedValue([
      entry({ actorName: undefined, actorId: 'acct-unknown' }),
    ])
    renderPage()
    expect(await screen.findByText('acct-unknown')).toBeInTheDocument()
  })

  it('labels a change the platform made by rule', async () => {
    activityApi.listActivity.mockResolvedValue([
      entry({ actorId: 'system', actorName: undefined }),
    ])
    renderPage()
    expect(await screen.findByText('স্বয়ংক্রিয়ভাবে')).toBeInTheDocument()
  })

  // A row about a problem the reader may not see arrives without a title (the
  // repository resolves public problems only). It must still render, without a
  // dead link to something they cannot open.
  it('renders a row with no resolvable problem, and no link', async () => {
    activityApi.listActivity.mockResolvedValue([entry({ problemTitle: undefined })])
    renderPage()
    await screen.findByText('প্রশাসক — আড়ানগর ইউনিয়ন')
    expect(screen.queryByRole('link', { name: /ফতেপুর/ })).toBeNull()
  })

  it('shows a rejection’s ground, because a takedown is accountable', async () => {
    activityApi.listActivity.mockResolvedValue([entry({ action: 'rejected', reason: 'spam' })])
    renderPage()
    expect(await screen.findByText('spam')).toBeInTheDocument()
  })

  // All three states (A.5.4).
  it('renders an empty record as a fact, not a failure', async () => {
    activityApi.listActivity.mockResolvedValue([])
    renderPage()
    expect(await screen.findByText(/এখনও কোনো সিদ্ধান্ত নেওয়া হয়নি/)).toBeInTheDocument()
  })

  it('offers a retry when the record cannot be loaded', async () => {
    activityApi.listActivity.mockRejectedValue({ status: 500, message: 'boom' })
    renderPage()
    expect(await screen.findByText(/রেকর্ড লোড করা যায়নি/)).toBeInTheDocument()
  })

  it('names the region so it can be scoped to', async () => {
    renderPage()
    await screen.findByText('প্রশাসক — আড়ানগর ইউনিয়ন')
    const region = screen.getByRole('region', { name: /সাম্প্রতিক সিদ্ধান্ত/ })
    expect(within(region).getByRole('link', { name: 'ফতেপুর বাজার সড়ক উন্নয়ন' })).toBeInTheDocument()
  })
})
