import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SuggestionList from './SuggestionList.jsx'

const h = vi.hoisted(() => ({ mutateAsync: vi.fn() }))
vi.mock('../../hooks/useSuggestions.js', () => ({
  useUpvoteSuggestion: () => ({ mutateAsync: h.mutateAsync, isPending: false }),
}))

const suggestion = (over = {}) => ({
  id: 's1',
  problemId: 'p1',
  authorId: 'res-1',
  text: 'কালভার্টটি নতুন করে বানাতে হবে',
  upvoteCount: 3,
  isTop: false,
  myUpvote: null,
  createdAt: '2026-07-01T09:00:00.000Z',
  ...over,
})

beforeEach(() => {
  h.mutateAsync.mockReset().mockResolvedValue({})
})

describe('SuggestionList', () => {
  it('invites a proposal when there are none', () => {
    render(<SuggestionList problemId="p1" suggestions={[]} />)
    expect(screen.getByText('এখনও কোনো প্রস্তাব নেই।')).toBeInTheDocument()
  })

  it('marks the top suggestion and says why it matters', () => {
    render(<SuggestionList problemId="p1" suggestions={[suggestion({ isTop: true })]} />)
    expect(screen.getByText('সবচেয়ে সমর্থিত')).toBeInTheDocument()
    expect(screen.getByText(/পরিকল্পনায় এই প্রস্তাবের জবাব দিতে হবে/)).toBeInTheDocument()
  })

  // Zero upvotes is never top, so a quiet problem shows no highlight at all. This
  // is correct and is also why the highlight can look broken in practice — the
  // ranking is backend-owned and must not be second-guessed here.
  it('highlights nothing when no suggestion has been upvoted', () => {
    render(<SuggestionList problemId="p1" suggestions={[suggestion({ upvoteCount: 0 })]} />)
    expect(screen.queryByText('সবচেয়ে সমর্থিত')).toBeNull()
  })

  // Ties all carry isTop, so the design must survive more than one highlighted row.
  it('handles a tie, where more than one suggestion is top', () => {
    render(
      <SuggestionList
        problemId="p1"
        suggestions={[suggestion({ id: 's1', isTop: true }), suggestion({ id: 's2', isTop: true })]}
      />,
    )
    expect(screen.getAllByText('সবচেয়ে সমর্থিত')).toHaveLength(2)
  })

  // The regression myUpvote exists for: upvote state used to be a local Set seeded
  // empty on every mount, so a reload re-offered the button on something already
  // upvoted — and pressing it silently WITHDREW the upvote.
  it('renders an upvote already cast from the server on first paint', () => {
    render(<SuggestionList problemId="p1" suggestions={[suggestion({ myUpvote: true })]} />)
    expect(screen.getByRole('button', { name: 'সমর্থন করুন' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('does not claim an upvote the reader has not cast', () => {
    render(<SuggestionList problemId="p1" suggestions={[suggestion({ myUpvote: null })]} />)
    expect(screen.getByRole('button', { name: 'সমর্থন করুন' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('shows the count in Bengali numerals', () => {
    render(<SuggestionList problemId="p1" suggestions={[suggestion({ upvoteCount: 12 })]} />)
    expect(screen.getByText('১২')).toBeInTheDocument()
    expect(screen.queryByText('12')).toBeNull()
  })

  // An unverified resident used to press upvote and watch the count not move, with
  // nothing said. The server's reason is rendered instead of a bare retry.
  it('explains a refused upvote instead of failing silently', async () => {
    const user = userEvent.setup()
    h.mutateAsync.mockRejectedValue({ status: 403, code: 'not_verified' })
    render(<SuggestionList problemId="p1" suggestions={[suggestion()]} />)

    await user.click(screen.getByRole('button', { name: 'সমর্থন করুন' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/আপনার অ্যাকাউন্ট এখনও যাচাই করা হয়নি/)
  })
})
