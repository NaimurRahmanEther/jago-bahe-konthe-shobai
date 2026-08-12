import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ImageViewer from './ImageViewer.jsx'

const onClose = vi.fn()

beforeEach(() => {
  onClose.mockClear()
})

/**
 * The viewer is opened from somewhere — a photo the reader clicked. The trigger
 * is part of what is under test: closing must hand focus back to it.
 */
function Harness({ open }) {
  return (
    <>
      <button type="button">ছবিটি বড় করে দেখুন</button>
      <ImageViewer open={open} src="data:image/jpeg;base64,abc" alt="ছবি" onClose={onClose} />
    </>
  )
}

describe('ImageViewer', () => {
  it('renders nothing while closed', () => {
    render(<Harness open={false} />)
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('shows the photo uncropped in a labelled dialog', () => {
    render(<Harness open />)
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAccessibleName('ছবি')
    // object-contain, never object-cover: this is the one place the photo is
    // shown whole, which is the entire reason to open it.
    expect(screen.getByAltText('ছবি').className).toContain('object-contain')
  })

  it('closes on Escape', async () => {
    render(<Harness open />)
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })

  it('closes on the close button', async () => {
    render(<Harness open />)
    await userEvent.click(screen.getByRole('button', { name: 'বন্ধ করুন' }))
    expect(onClose).toHaveBeenCalled()
  })

  it('closes on a backdrop click but not on the photo itself', async () => {
    render(<Harness open />)

    await userEvent.click(screen.getByAltText('ছবি'))
    expect(onClose).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('dialog'))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('takes focus on open and returns it to the trigger on close', async () => {
    const { rerender } = render(<Harness open={false} />)
    const opener = screen.getByRole('button', { name: 'ছবিটি বড় করে দেখুন' })
    opener.focus()

    rerender(<Harness open />)
    expect(screen.getByRole('button', { name: 'বন্ধ করুন' })).toHaveFocus()

    rerender(<Harness open={false} />)
    expect(opener).toHaveFocus()
  })

  it('locks body scroll only while open', () => {
    const { rerender } = render(<Harness open={false} />)
    expect(document.body.style.overflow).toBe('')

    rerender(<Harness open />)
    expect(document.body.style.overflow).toBe('hidden')

    rerender(<Harness open={false} />)
    expect(document.body.style.overflow).toBe('')
  })
})
