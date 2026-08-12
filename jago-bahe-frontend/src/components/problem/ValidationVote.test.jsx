import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ValidationVote from './ValidationVote.jsx'

// Shared, mutable mock state (hoisted above the vi.mock factories).
const h = vi.hoisted(() => ({
  role: 'resident',
  user: { id: 'res-1', verified: true },
  updateUser: vi.fn(),
  mutateAsync: vi.fn(),
}))
vi.mock('../../auth/useAuth.js', () => ({
  useAuth: () => ({ role: h.role, user: h.user, updateUser: h.updateUser }),
}))
vi.mock('../../hooks/useProblems.js', () => ({
  useValidateProblem: () => ({ mutateAsync: h.mutateAsync, isPending: false }),
}))

const problem = (over = {}) => ({
  id: 'p1',
  validCount: 3,
  validationThreshold: 5,
  status: 'Reported',
  myVote: null,
  ...over,
})

beforeEach(() => {
  h.role = 'resident'
  h.user = { id: 'res-1', verified: true }
  h.updateUser.mockReset()
  h.mutateAsync.mockReset().mockResolvedValue({})
})

describe('ValidationVote', () => {
  it('shows the valid/invalid controls and the public count for a resident', () => {
    render(<ValidationVote problem={problem()} />)
    expect(screen.getByRole('button', { name: 'বৈধ' })).toBeInTheDocument()
    // 'বৈধ' is a substring of 'অবৈধ'; getByRole's name matches the full
    // normalized string, so these stay distinct. Do not loosen to a regex.
    expect(screen.getByRole('button', { name: 'অবৈধ' })).toBeInTheDocument()
    // Bengali numerals — the app is Bangla-only (A.5.3).
    expect(screen.getByText('৩ জন যাচাই করেছেন')).toBeInTheDocument()
  })

  // The threshold is an admin-surface fact now (A.3.1.1); the public sees the
  // count alone. If this fails, ValidationBar has crept back onto a public screen.
  it('shows no threshold and no progress bar', () => {
    render(<ValidationVote problem={problem()} />)
    expect(screen.queryByText('৩ / ৫ জন যাচাই করেছেন')).toBeNull()
    expect(screen.queryByRole('progressbar')).toBeNull()
  })

  it('casts a vote and thanks the resident', async () => {
    const user = userEvent.setup()
    render(<ValidationVote problem={problem()} />)
    await user.click(screen.getByRole('button', { name: 'বৈধ' }))
    expect(h.mutateAsync).toHaveBeenCalledWith({ id: 'p1', vote: 'valid' })
    expect(await screen.findByText('আপনার মতামতের জন্য ধন্যবাদ।')).toBeInTheDocument()
  })

  // The regression this whole DTO field exists for: "have I voted" used to be
  // component-local, so a reload re-offered the buttons on a report the resident
  // had already validated, and the backend answered 409 to a vote the UI invited.
  it('renders an already-cast vote from the server on first paint', () => {
    render(<ValidationVote problem={problem({ myVote: 'valid' })} />)
    expect(screen.getByRole('button', { name: 'বৈধ' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'অবৈধ' })).toBeDisabled()
    expect(screen.getByText('আপনার মত: বৈধ')).toBeInTheDocument()
  })

  // The mutation used to be optimistic — votedChoice was set BEFORE the request,
  // so "ধন্যবাদ" rendered whether or not the server accepted. Thanking someone for
  // a civic act that did not happen is the kind of lie this platform cannot tell.
  it('reports a refused repeat vote instead of thanking for it', async () => {
    const user = userEvent.setup()
    h.mutateAsync.mockRejectedValue({ status: 409 })
    render(<ValidationVote problem={problem()} />)
    await user.click(screen.getByRole('button', { name: 'বৈধ' }))

    expect(await screen.findByText(/ইতোমধ্যে যাচাই করেছেন/)).toBeInTheDocument()
    expect(screen.queryByText('আপনার মতামতের জন্য ধন্যবাদ।')).toBeNull()
    // Still usable: the vote did not land, so the controls stay live.
    expect(screen.getByRole('button', { name: 'বৈধ' })).toBeEnabled()
  })

  it('reports any other failure plainly', async () => {
    const user = userEvent.setup()
    h.mutateAsync.mockRejectedValue({ status: 500 })
    render(<ValidationVote problem={problem()} />)
    await user.click(screen.getByRole('button', { name: 'বৈধ' }))
    expect(await screen.findByText(/যাচাই করা যায়নি/)).toBeInTheDocument()
  })

  it('does not offer voting to a non-resident', () => {
    h.role = 'official'
    render(<ValidationVote problem={problem()} />)
    expect(screen.queryByRole('button', { name: 'বৈধ' })).toBeNull()
    expect(screen.getByText(/শুধুমাত্র বাসিন্দারা যাচাই করতে পারবেন/)).toBeInTheDocument()
  })

  // Post-assignment voting stays open on purpose — A.3.1.1 constraint 3 makes the
  // continuing count the signal that lets an early assignment be judged.
  it('still offers voting once a report has been assigned', () => {
    render(<ValidationVote problem={problem({ status: 'InProgress' })} />)
    expect(screen.getByRole('button', { name: 'বৈধ' })).toBeInTheDocument()
  })

  it.each(['Rejected', 'Withdrawn'])('closes voting on a %s report', (status) => {
    render(<ValidationVote problem={problem({ status })} />)
    expect(screen.queryByRole('button', { name: 'বৈধ' })).toBeNull()
    expect(screen.getByText('৩ জন যাচাই করেছেন')).toBeInTheDocument()
  })

  // A resident used to see the prompt "is this problem real?" with no buttons and
  // no explanation on a closed report — which reads as the feature being broken
  // rather than the question being over. Every closed branch now says why.
  it('explains why voting is closed rather than just hiding the controls', () => {
    render(<ValidationVote problem={problem({ status: 'Rejected' })} />)
    expect(screen.getByText(/আর মতামত নেওয়া হচ্ছে না/)).toBeInTheDocument()
    // Not the logged-out copy: this resident IS signed in.
    expect(screen.queryByText(/অনুগ্রহ করে লগইন করুন/)).toBeNull()
  })

  // Every self-registered resident starts unverified, so this is the state most
  // first-time users are actually in. It used to render enabled buttons that could
  // only ever 403, with a generic "try again" that could never work.
  describe('an unverified resident', () => {
    beforeEach(() => {
      h.user = { id: 'res-1', verified: false }
    })

    it('is told their account is awaiting verification', () => {
      render(<ValidationVote problem={problem()} />)
      expect(screen.getByText(/আপনার অ্যাকাউন্ট এখনও যাচাই করা হয়নি/)).toBeInTheDocument()
    })

    // Deliberately NOT disabled: `verified` is captured at login and there is no
    // endpoint to re-read it, so it is stale-false for anyone verified mid-session.
    // Blocking on it would lock out exactly the person who just became eligible.
    it('may still try, and the stale flag heals when the vote lands', async () => {
      const user = userEvent.setup()
      render(<ValidationVote problem={problem()} />)
      const valid = screen.getByRole('button', { name: 'বৈধ' })
      expect(valid).toBeEnabled()

      await user.click(valid)
      expect(await screen.findByText('আপনার মতামতের জন্য ধন্যবাদ।')).toBeInTheDocument()
      expect(h.updateUser).toHaveBeenCalledWith({ verified: true })
    })

    it('gets the real reason when the server refuses, not a bare retry', async () => {
      const user = userEvent.setup()
      h.mutateAsync.mockRejectedValue({ status: 403, code: 'not_verified' })
      render(<ValidationVote problem={problem()} />)
      await user.click(screen.getByRole('button', { name: 'বৈধ' }))
      // Scoped to the alert: the standing advisory above carries the same words,
      // so an unscoped query matches two elements (A.5.3 rule 4).
      expect(await screen.findByRole('alert')).toHaveTextContent(/আপনার অ্যাকাউন্ট এখনও যাচাই করা হয়নি/)
      expect(screen.queryByText(/যাচাই করা যায়নি/)).toBeNull()
    })
  })

  // 403 covers two opposite refusals, so the code — not the status — has to pick
  // the advice. Telling an out-of-area resident to wait for verification would be
  // confidently wrong.
  it('distinguishes an out-of-area refusal from an unverified one', async () => {
    const user = userEvent.setup()
    h.mutateAsync.mockRejectedValue({ status: 403, code: 'not_area_resident' })
    render(<ValidationVote problem={problem()} />)
    await user.click(screen.getByRole('button', { name: 'বৈধ' }))
    expect(await screen.findByText(/শুধুমাত্র এই এলাকার বাসিন্দারা/)).toBeInTheDocument()
  })

  // The latching bug: votedChoice was useState seeded once per MOUNT, and
  // ProblemDetail renders this without a key, so the router keeps it mounted across
  // an :id change. Voting on one report left the next one's buttons disabled and
  // thanked you for a vote you never cast. State is derived from myVote now.
  it('offers its controls on a second report after voting on a first', async () => {
    const user = userEvent.setup()
    const { rerender } = render(<ValidationVote problem={problem({ id: 'p1' })} />)
    await user.click(screen.getByRole('button', { name: 'বৈধ' }))
    expect(await screen.findByText('আপনার মতামতের জন্য ধন্যবাদ।')).toBeInTheDocument()

    // Same component instance, different problem — exactly what the router does.
    rerender(<ValidationVote problem={problem({ id: 'p2', myVote: null })} />)
    expect(screen.getByRole('button', { name: 'বৈধ' })).toBeEnabled()
    expect(screen.queryByText('আপনার মতামতের জন্য ধন্যবাদ।')).toBeNull()
  })
})
