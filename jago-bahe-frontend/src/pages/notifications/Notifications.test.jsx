import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Notifications from './Notifications.jsx'
import * as notificationApi from '../../lib/api/notifications.js'

// A page test mocks the API MODULE and drives the real hooks — the only way the
// three states are exercised for real. The factory must list EVERY export of the
// module: useNotifications.js does `import * as`, so an omitted name is not an
// import error, it is `undefined` at call time (A.5.5 rule 1).
vi.mock('../../lib/api/notifications.js', () => ({
  listNotifications: vi.fn(),
  unreadNotificationCount: vi.fn(),
  markNotificationRead: vi.fn(),
  markAllNotificationsRead: vi.fn(),
}))

vi.mock('../../auth/useAuth.js', () => ({
  useAuth: () => ({ user: { id: 'acct-1' }, role: 'resident' }),
}))

const row = (id, overrides = {}) => ({
  id,
  type: 'problem_approved',
  problemId: 'prob-1',
  problemTitle: 'রাস্তার বাতিগুলো জ্বলছে না',
  detail: '',
  readAt: null,
  createdAt: '2026-07-20T10:00:00.000Z',
  ...overrides,
})

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Notifications />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Notifications', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(notificationApi.unreadNotificationCount).mockResolvedValue({ count: 0 })
    vi.mocked(notificationApi.markNotificationRead).mockResolvedValue(undefined)
    vi.mocked(notificationApi.markAllNotificationsRead).mockResolvedValue({ marked: 0 })
  })

  it('shows a skeleton while loading', () => {
    vi.mocked(notificationApi.listNotifications).mockReturnValue(new Promise(() => {}))
    renderPage()
    expect(screen.getByRole('status', { name: 'লোড হচ্ছে...' })).toBeInTheDocument()
  })

  it('shows the cause and a retry when the list fails', async () => {
    vi.mocked(notificationApi.listNotifications).mockRejectedValue({ status: 500, code: '', message: 'boom' })
    renderPage()
    expect(await screen.findByText('বিজ্ঞপ্তি আনা যায়নি।')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'আবার চেষ্টা করুন' })).toBeInTheDocument()
  })

  // A.6: an empty state is an invitation to act, never a dead end.
  it('offers an action when there is nothing yet', async () => {
    vi.mocked(notificationApi.listNotifications).mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('এখনো কোনো বিজ্ঞপ্তি নেই')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'সব প্রতিবেদন দেখুন' })).toBeInTheDocument()
  })

  it('lists the caller notifications', async () => {
    vi.mocked(notificationApi.listNotifications).mockResolvedValue([row('notif-1'), row('notif-2')])
    renderPage()
    // Scoped by region: "বিজ্ঞপ্তি" is the page heading, the nav word AND the bell
    // label, so a bare getByText would find several (A.5.3 rule 4).
    const list = await screen.findByRole('region', { name: 'সাম্প্রতিক' })
    expect(within(list).getAllByRole('link')).toHaveLength(2)
  })

  // Opening the page must NOT clear the badge — someone who opens it to triage
  // would otherwise lose the record of what was new. The escape hatch is explicit.
  it('does not mark anything read merely by rendering', async () => {
    vi.mocked(notificationApi.listNotifications).mockResolvedValue([row('notif-1')])
    renderPage()
    await screen.findByRole('region', { name: 'সাম্প্রতিক' })
    expect(notificationApi.markAllNotificationsRead).not.toHaveBeenCalled()
    expect(notificationApi.markNotificationRead).not.toHaveBeenCalled()
  })

  it('offers mark-all only when something is unread', async () => {
    vi.mocked(notificationApi.listNotifications).mockResolvedValue([
      row('notif-1', { readAt: '2026-07-21T09:00:00.000Z' }),
    ])
    renderPage()
    await screen.findByRole('region', { name: 'সাম্প্রতিক' })
    expect(screen.queryByRole('button', { name: 'সব পড়া হয়েছে' })).toBeNull()
  })

  it('clears the badge on request', async () => {
    const user = userEvent.setup()
    vi.mocked(notificationApi.listNotifications).mockResolvedValue([row('notif-1')])
    renderPage()

    const button = await screen.findByRole('button', { name: 'সব পড়া হয়েছে' })
    await user.click(button)
    expect(notificationApi.markAllNotificationsRead).toHaveBeenCalledTimes(1)
  })

  it('reports a failed mark-all without losing the list', async () => {
    const user = userEvent.setup()
    vi.mocked(notificationApi.listNotifications).mockResolvedValue([row('notif-1')])
    vi.mocked(notificationApi.markAllNotificationsRead).mockRejectedValue({ status: 500, code: '', message: 'boom' })
    renderPage()

    await user.click(await screen.findByRole('button', { name: 'সব পড়া হয়েছে' }))
    expect(await screen.findByText('চিহ্নিত করা যায়নি।')).toBeInTheDocument()
    // The rows are still there — a failed secondary action must not empty the page.
    const list = screen.getByRole('region', { name: 'সাম্প্রতিক' })
    expect(within(list).getAllByRole('link')).toHaveLength(1)
  })

  it('marks a row read when it is followed', async () => {
    const user = userEvent.setup()
    vi.mocked(notificationApi.listNotifications).mockResolvedValue([row('notif-1')])
    renderPage()

    const list = await screen.findByRole('region', { name: 'সাম্প্রতিক' })
    await user.click(within(list).getAllByRole('link')[0])
    expect(notificationApi.markNotificationRead).toHaveBeenCalledWith('notif-1')
  })
})
