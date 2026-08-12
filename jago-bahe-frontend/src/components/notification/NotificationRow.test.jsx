import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import NotificationRow from './NotificationRow.jsx'

// The fixture mirrors notificationDTO field-for-field. `readAt` is present-and-null
// on an unread row, never absent — the QueueItem lesson at the field level: a shape
// the component reads and the API does not send is invisible until something
// renders it.
const ROW = {
  id: 'notif-1',
  type: 'problem_approved',
  problemId: 'prob-1',
  problemTitle: 'রাস্তার বাতিগুলো জ্বলছে না',
  detail: '',
  readAt: null,
  createdAt: '2026-07-20T10:00:00.000Z',
}

const renderRow = (overrides = {}, onRead = vi.fn()) => {
  const utils = render(
    <MemoryRouter>
      <NotificationRow notification={{ ...ROW, ...overrides }} onRead={onRead} />
    </MemoryRouter>,
  )
  return { ...utils, onRead }
}

describe('NotificationRow', () => {
  it('renders the snapshotted title and links to the public problem page', () => {
    renderRow()
    expect(screen.getByText('রাস্তার বাতিগুলো জ্বলছে না')).toBeInTheDocument()
    expect(screen.getByRole('link')).toHaveAttribute('href', '/problems/prob-1')
  })

  // A.3.2.1 rule 1: a <button> inside an <a> is invalid HTML and breaks keyboard
  // navigation, which is exactly why the whole row is one link and mark-read fires
  // from its onClick. If a control is ever wanted here, that anchor is the thing to
  // solve first, not work around.
  it('is a single link and contains no button', () => {
    renderRow()
    expect(screen.getAllByRole('link')).toHaveLength(1)
    expect(screen.queryByRole('button')).toBeNull()
  })

  it('falls back to a placeholder when the snapshotted title is empty', () => {
    renderRow({ problemTitle: '' })
    expect(screen.getByText('শিরোনামহীন প্রতিবেদন')).toBeInTheDocument()
  })

  // Colour is never the only signal (Guideline §2 rule 1): the unread marker is a
  // dot AND the word.
  it('marks an unread row with a word, not colour alone', () => {
    renderRow()
    expect(screen.getByText('নতুন')).toBeInTheDocument()
  })

  it('does not mark a read row as new', () => {
    renderRow({ readAt: '2026-07-21T09:00:00.000Z' })
    expect(screen.queryByText('নতুন')).toBeNull()
  })

  it('marks an unread row read when it is followed', async () => {
    const user = userEvent.setup()
    const { onRead } = renderRow()
    await user.click(screen.getByRole('link'))
    expect(onRead).toHaveBeenCalledWith('notif-1')
  })

  it('does not re-mark a row that is already read', async () => {
    const user = userEvent.setup()
    const { onRead } = renderRow({ readAt: '2026-07-21T09:00:00.000Z' })
    await user.click(screen.getByRole('link'))
    expect(onRead).not.toHaveBeenCalled()
  })

  // One case per type. `detail` is type-specific and is only ever read inside the
  // component's switch, so each branch has to be exercised — a wrong branch would
  // render an unrelated sentence with a plausible-looking payload in it.
  describe('the eight sentences', () => {
    it('problem_approved', () => {
      renderRow({ type: 'problem_approved' })
      expect(screen.getByText(/অনুমোদিত হয়েছে/)).toBeInTheDocument()
    })

    it('problem_rejected renders the ground through the shared Bangla labels', () => {
      renderRow({ type: 'problem_rejected', detail: 'spam' })
      // Reusing problem.rejectionReason.* rather than a second set of labels is what
      // stops the moderation screen and this row disagreeing about what "spam" says.
      expect(screen.getByText(/স্প্যাম/)).toBeInTheDocument()
    })

    it('problem_assigned names the official', () => {
      renderRow({ type: 'problem_assigned', detail: 'মো. শাহাদত হোসেন' })
      expect(screen.getByText(/মো. শাহাদত হোসেন/)).toBeInTheDocument()
    })

    it('confirmation_requested asks the reporter to confirm', () => {
      renderRow({ type: 'confirmation_requested' })
      expect(screen.getByText(/নিশ্চিত করুন/)).toBeInTheDocument()
    })

    it('case_assigned renders the deadline as a date, not a raw timestamp', () => {
      renderRow({ type: 'case_assigned', detail: '2026-08-01T00:00:00Z' })
      expect(screen.queryByText(/2026-08-01T00:00:00Z/)).toBeNull()
      expect(screen.getByText(/সময়সীমা/)).toBeInTheDocument()
    })

    it('case_reopened', () => {
      renderRow({ type: 'case_reopened' })
      expect(screen.getByText(/আবার চালু হয়েছে/)).toBeInTheDocument()
    })

    it('obstacle_declared carries the free-text whoUnblocks', () => {
      renderRow({ type: 'obstacle_declared', detail: 'উপজেলা প্রকৌশলী' })
      expect(screen.getByText(/উপজেলা প্রকৌশলী/)).toBeInTheDocument()
    })

    // One type, two OPPOSITE outcomes — confirmed protects the official, denied
    // bounces the work back to them. Getting the branch wrong would tell an official
    // the reverse of what happened, which is the worst failure in the set.
    it('obstacle_adjudicated distinguishes confirmed from denied', () => {
      const { unmount } = renderRow({ type: 'obstacle_adjudicated', detail: 'confirmed' })
      expect(screen.getByText(/সত্য বলে স্বীকৃত/)).toBeInTheDocument()
      unmount()

      renderRow({ type: 'obstacle_adjudicated', detail: 'denied' })
      expect(screen.getByText(/নাকচ হয়েছে/)).toBeInTheDocument()
    })
  })

  // The backend whitelists eight in a CHECK constraint, so a ninth reaching here
  // means the halves have drifted. Saying less is honest; rendering a raw i18n key
  // at a resident is not.
  it('says nothing rather than shouting a raw key on an unknown type', () => {
    renderRow({ type: 'something_new' })
    expect(screen.queryByText(/notification\.type/)).toBeNull()
    expect(screen.getByText('রাস্তার বাতিগুলো জ্বলছে না')).toBeInTheDocument()
  })
})
