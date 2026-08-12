import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import Login from './Login.jsx'

// The page reads `login` from auth; stub it so these tests are about what the
// form submits and what it says when the server refuses — not about the session.
const login = vi.fn()
vi.mock('../../auth/useAuth.js', () => ({ useAuth: () => ({ login }) }))

// "লগইন করুন" is BOTH the heading and the submit button (auth.login.title and
// auth.login.submit), so every query here is scoped by role — A.5.3 rule 4.
const submit = () => screen.getByRole('button', { name: 'লগইন করুন' })
const phoneField = () => screen.getByLabelText('মোবাইল নম্বর')
const passwordField = () => screen.getByLabelText('পাসওয়ার্ড')

function renderLogin() {
  return render(
    <MemoryRouter>
      <Login />
    </MemoryRouter>,
  )
}

beforeEach(() => {
  login.mockReset()
  login.mockResolvedValue({ role: 'resident', user: { id: 'res-1' } })
})

describe('Login', () => {
  // The reported bug: registration stores 01810000001, then autofill returns the
  // same number spelled "+880 1810-000001" and the account becomes unreachable.
  it.each([
    ['+880 1810-000001', 'autofill formatting'],
    ['+8801810000001', 'country code'],
    ['01810-000001', 'dashes'],
    ['  01810000001  ', 'stray whitespace'],
    ['০১৮১০০০০০০১', 'Bengali numerals'],
  ])('submits %s (%s) as the canonical number', async (typed) => {
    const user = userEvent.setup()
    renderLogin()

    await user.type(phoneField(), typed)
    await user.type(passwordField(), 'resident123')
    await user.click(submit())

    expect(login).toHaveBeenCalledWith('01810000001', 'resident123')
  })

  it('rejects a malformed phone inline, without a round-trip', async () => {
    const user = userEvent.setup()
    renderLogin()

    await user.type(phoneField(), '0181000')
    await user.type(passwordField(), 'resident123')
    await user.click(submit())

    expect(await screen.findByText('সঠিক ১১ ডিজিটের মোবাইল নম্বর দিন')).toBeInTheDocument()
    expect(login).not.toHaveBeenCalled()
  })

  // Each status must name its own cause. A bare catch used to show "phone or
  // password is not valid" for all of these, which is what made a rate limit and
  // a dead server look like a wrong password.
  it.each([
    [401, 'মোবাইল নম্বর অথবা পাসওয়ার্ড সঠিক নয়।'],
    [429, 'অনেকবার চেষ্টা করা হয়েছে। এক মিনিট পরে আবার চেষ্টা করুন।'],
    [500, 'সার্ভারে সংযোগ করা যায়নি। আবার চেষ্টা করুন।'],
    [0, 'সার্ভারে সংযোগ করা যায়নি। আবার চেষ্টা করুন।'],
  ])('reports a %i as its own cause', async (status, expected) => {
    login.mockRejectedValue({ status, message: 'nope' })
    const user = userEvent.setup()
    renderLogin()

    await user.type(phoneField(), '01810000001')
    await user.type(passwordField(), 'wrong-password')
    await user.click(submit())

    expect(await screen.findByText(expected)).toBeInTheDocument()
  })

  it('shows a 400 as a phone field error, not a credential failure', async () => {
    login.mockRejectedValue({ status: 400, message: 'invalid_phone' })
    const user = userEvent.setup()
    renderLogin()

    await user.type(phoneField(), '01810000001')
    await user.type(passwordField(), 'resident123')
    await user.click(submit())

    expect(await screen.findByText('সঠিক ১১ ডিজিটের মোবাইল নম্বর দিন')).toBeInTheDocument()
    expect(screen.queryByText('মোবাইল নম্বর অথবা পাসওয়ার্ড সঠিক নয়।')).not.toBeInTheDocument()
  })
})
