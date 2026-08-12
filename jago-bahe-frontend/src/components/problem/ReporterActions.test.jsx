import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import ReporterActions from './ReporterActions.jsx'

// Shared, mutable mock state (hoisted above the vi.mock factories).
const h = vi.hoisted(() => ({
  userId: 'res-1',
  withdrawMutate: vi.fn(),
  deleteMutate: vi.fn(),
  navigate: vi.fn(),
}))

vi.mock('../../auth/useAuth.js', () => ({ useAuth: () => ({ user: { id: h.userId } }) }))
vi.mock('../../hooks/useProblems.js', () => ({
  useWithdrawProblem: () => ({ mutateAsync: h.withdrawMutate, isPending: false }),
  useDeleteProblem: () => ({ mutateAsync: h.deleteMutate, isPending: false }),
}))
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal()),
  useNavigate: () => h.navigate,
}))

const problem = (over = {}) => ({
  id: 'p1',
  reporterId: 'res-1',
  status: 'Reported',
  validCount: 0,
  ...over,
})

const show = (p) =>
  render(
    <MemoryRouter>
      <ReporterActions problem={p} />
    </MemoryRouter>,
  )

// 'প্রত্যাহার করুন' is BOTH the withdraw trigger and the confirm button inside its
// modal (bn.json problem.detail.withdraw and .withdrawConfirm are the same string),
// so an unscoped query finds two elements. Reach into the dialog explicitly.
const inDialog = () => within(screen.getByRole('dialog'))

beforeEach(() => {
  h.userId = 'res-1'
  h.withdrawMutate.mockReset().mockResolvedValue({})
  h.deleteMutate.mockReset().mockResolvedValue(undefined)
  h.navigate.mockReset()
})

describe('ReporterActions', () => {
  it('renders nothing for anyone who is not the reporter', () => {
    h.userId = 'res-2'
    const { container } = show(problem())
    expect(container).toBeEmptyDOMElement()
  })

  it('renders nothing for a logged-out visitor', () => {
    h.userId = undefined
    const { container } = show(problem())
    expect(container).toBeEmptyDOMElement()
  })

  it('offers edit, withdraw and delete on the reporter’s own fresh report', () => {
    show(problem())
    expect(screen.getByRole('link', { name: 'সম্পাদনা করুন' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'প্রত্যাহার করুন' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'মুছে ফেলুন' })).toBeInTheDocument()
  })

  // The backend's Withdrawable() includes PendingApproval — a report is its
  // reporter's own until an admin acts. The UI predicate omitted it, so a report
  // stuck in the screening queue was a visible state with no exit.
  it('offers withdraw on a PendingApproval report, but not edit', () => {
    show(problem({ status: 'PendingApproval' }))
    expect(screen.getByRole('button', { name: 'প্রত্যাহার করুন' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'সম্পাদনা করুন' })).toBeNull()
  })

  it('closes the edit window once a report has been endorsed', () => {
    show(problem({ validCount: 1 }))
    expect(screen.queryByRole('link', { name: 'সম্পাদনা করুন' })).toBeNull()
    expect(screen.getByRole('button', { name: 'প্রত্যাহার করুন' })).toBeInTheDocument()
  })

  // Delete has no window: it is the one action still offered once a case exists,
  // and it erases that case with it. This is the rule most likely to look like a
  // missing guard — see CLAUDE.md A.3.3 before "fixing" it.
  it('offers delete but not withdraw or edit once the report is assigned', () => {
    show(problem({ status: 'Assigned' }))
    expect(screen.getByRole('button', { name: 'মুছে ফেলুন' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'প্রত্যাহার করুন' })).toBeNull()
    expect(screen.queryByRole('link', { name: 'সম্পাদনা করুন' })).toBeNull()
  })

  it('still offers delete on a resolved report', () => {
    show(problem({ status: 'Resolved' }))
    expect(screen.getByRole('button', { name: 'মুছে ফেলুন' })).toBeInTheDocument()
  })

  it('withdraws with the trimmed note and closes the dialog', async () => {
    const user = userEvent.setup()
    show(problem())
    await user.click(screen.getByRole('button', { name: 'প্রত্যাহার করুন' }))
    await user.type(inDialog().getByLabelText(/কারণ/), '  ভুল করে জমা দিয়েছি  ')
    await user.click(inDialog().getByRole('button', { name: 'প্রত্যাহার করুন' }))

    expect(h.withdrawMutate).toHaveBeenCalledWith('ভুল করে জমা দিয়েছি')
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('warns about what deleting destroys before confirming', async () => {
    const user = userEvent.setup()
    show(problem())
    await user.click(screen.getByRole('button', { name: 'মুছে ফেলুন' }))

    const dialog = inDialog()
    expect(dialog.getByText(/চিরতরে মুছে যাবে/)).toBeInTheDocument()
    // The costs that are not the reporter's own: other residents' validations and
    // the official's work. A confirm dialog that hid these would be the real bug.
    expect(dialog.getByText(/পুরো অডিট ট্রেইল/)).toBeInTheDocument()
    // Withdraw is offered as the reversible alternative, at the moment it matters.
    expect(dialog.getByText(/প্রত্যাহার করলে/)).toBeInTheDocument()
  })

  it('deletes on confirm and navigates away from the report', async () => {
    const user = userEvent.setup()
    show(problem())
    await user.click(screen.getByRole('button', { name: 'মুছে ফেলুন' }))
    await user.click(inDialog().getByRole('button', { name: 'স্থায়ীভাবে মুছে ফেলুন' }))

    expect(h.deleteMutate).toHaveBeenCalledTimes(1)
    expect(h.navigate).toHaveBeenCalledWith('/me')
  })

  it('does not navigate when the caller handles removal itself', async () => {
    const user = userEvent.setup()
    const onDeleted = vi.fn()
    render(
      <MemoryRouter>
        <ReporterActions problem={problem()} onDeleted={onDeleted} />
      </MemoryRouter>,
    )
    await user.click(screen.getByRole('button', { name: 'মুছে ফেলুন' }))
    await user.click(inDialog().getByRole('button', { name: 'স্থায়ীভাবে মুছে ফেলুন' }))

    expect(onDeleted).toHaveBeenCalledTimes(1)
    expect(h.navigate).not.toHaveBeenCalled()
  })

  it('surfaces a failed delete and keeps the dialog open', async () => {
    const user = userEvent.setup()
    h.deleteMutate.mockRejectedValue(new Error('boom'))
    show(problem())
    await user.click(screen.getByRole('button', { name: 'মুছে ফেলুন' }))
    await user.click(inDialog().getByRole('button', { name: 'স্থায়ীভাবে মুছে ফেলুন' }))

    expect(await screen.findByText(/মুছে ফেলা যায়নি/)).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(h.navigate).not.toHaveBeenCalled()
  })

  it('surfaces a failed withdraw and keeps the dialog open', async () => {
    const user = userEvent.setup()
    h.withdrawMutate.mockRejectedValue(new Error('409'))
    show(problem())
    await user.click(screen.getByRole('button', { name: 'প্রত্যাহার করুন' }))
    await user.click(inDialog().getByRole('button', { name: 'প্রত্যাহার করুন' }))

    expect(await screen.findByText(/প্রত্যাহার করা যায়নি/)).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
