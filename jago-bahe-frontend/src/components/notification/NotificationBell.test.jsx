import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import NotificationBell from './NotificationBell.jsx'

// A component test mocks the HOOK (a page test mocks the api module). The bell has
// exactly one data dependency and one identity dependency, so both are stubbed and
// the assertions are about rendering, not about fetching.
let mockUser = { id: 'acct-1' }
let mockCount = 0

vi.mock('../../auth/useAuth.js', () => ({
  useAuth: () => ({ user: mockUser, role: 'resident' }),
}))
vi.mock('../../hooks/useNotifications.js', () => ({
  useUnreadCount: () => ({ data: { count: mockCount } }),
}))

const renderBell = () =>
  render(
    <MemoryRouter>
      <NotificationBell />
    </MemoryRouter>,
  )

describe('NotificationBell', () => {
  beforeEach(() => {
    mockUser = { id: 'acct-1' }
    mockCount = 0
  })

  // Guideline §9 is explicit: "icons always with text — never icon-only (literacy +
  // clarity)". A bare bell glyph is not permitted, and this is the pin.
  it('is never icon-only — the word is always present', () => {
    renderBell()
    expect(screen.getByText('বিজ্ঞপ্তি')).toBeInTheDocument()
  })

  it('links to the notifications page', () => {
    renderBell()
    expect(screen.getByRole('link')).toHaveAttribute('href', '/notifications')
  })

  it('shows no count pill at zero', () => {
    renderBell()
    const link = screen.getByRole('link')
    expect(link.textContent).toBe('বিজ্ঞপ্তি')
  })

  it('renders the count in Bengali numerals', () => {
    mockCount = 3
    renderBell()
    expect(screen.getByText('৩')).toBeInTheDocument()
  })

  // The number reaches the DOM without passing through t(), so it converts via
  // toBengaliDigits (A.5.3 rule 5). A Latin digit here is the tell that the
  // conversion was missed — which is exactly how F18's first attempt passed the
  // whole suite while changing not one digit.
  it('renders no Latin digits anywhere', () => {
    mockCount = 12
    renderBell()
    expect(screen.getByRole('link').textContent).not.toMatch(/[0-9]/)
  })

  // Colour is never the only signal: a screen reader gets the whole sentence, with
  // the count formatted by i18next's own {{count, number}}.
  it('spells the count out for a screen reader', () => {
    mockCount = 3
    renderBell()
    expect(screen.getByRole('link', { name: /৩টি নতুন বিজ্ঞপ্তি/ })).toBeInTheDocument()
  })

  it('says so when there is nothing new', () => {
    renderBell()
    expect(screen.getByRole('link', { name: 'কোনো নতুন বিজ্ঞপ্তি নেই' })).toBeInTheDocument()
  })

  // Signed out there is nothing to show and nothing to fetch; rendering an empty
  // bell would invite a click straight into a redirect.
  it('renders nothing when signed out', () => {
    mockUser = null
    const { container } = renderBell()
    expect(container).toBeEmptyDOMElement()
  })
})
