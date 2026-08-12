import { describe, it, expect, beforeEach, vi } from 'vitest'
// waitFor from RTL, not vi.waitFor: compressing is async, so the emit and the
// busy-state reset land after an await. RTL's waitFor wraps them in act(); the
// vitest one does not, and warns.
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import ImagePicker from './ImagePicker.jsx'
import { compressImage } from '../../lib/image.js'

// Stubbed because compressImage is canvas + Image work that jsdom does not
// implement — and because these tests are about the picker's behaviour (accept,
// refuse, preview, remove), not about JPEG encoding. The compression itself is
// browser work, verified against a real photo in a real browser.
vi.mock('../../lib/image.js', () => ({
  compressImage: vi.fn(async () => 'data:image/jpeg;base64,compressed'),
  MAX_IMAGE_BYTES: 900 * 1024,
}))

const onChange = vi.fn()

function makeFile(name, sizeBytes) {
  const file = new File(['x'], name, { type: 'image/jpeg' })
  // A real 6MB buffer would slow the suite down; File.size is read-only, so
  // redefine it rather than allocating.
  Object.defineProperty(file, 'size', { value: sizeBytes })
  return file
}

function setup(value = '') {
  return render(<ImagePicker id="photo" label="ছবি" hint="ঐচ্ছিক" value={value} onChange={onChange} />)
}

beforeEach(() => {
  onChange.mockClear()
  vi.mocked(compressImage).mockClear()
  vi.mocked(compressImage).mockResolvedValue('data:image/jpeg;base64,compressed')
})

describe('ImagePicker', () => {
  it('shows the attach box and its size hint when no photo is chosen', () => {
    setup()
    expect(screen.getByText('ছবি যোগ করুন')).toBeInTheDocument()
    expect(screen.getByText('JPG / PNG · সর্বোচ্চ ৫MB')).toBeInTheDocument()
    expect(screen.queryByRole('img')).toBeNull()
  })

  it('emits a data URL for a file within the size cap', async () => {
    const user = userEvent.setup()
    setup()
    await user.upload(screen.getByLabelText(/ছবি/), makeFile('ok.jpg', 1024))
    await waitFor(() => expect(onChange).toHaveBeenCalledTimes(1))
    expect(onChange.mock.calls[0][0]).toMatch(/^data:image\/jpeg;base64,/)
  })

  it('refuses a file over 5MB and never emits it', async () => {
    const user = userEvent.setup()
    setup()
    await user.upload(screen.getByLabelText(/ছবি/), makeFile('huge.jpg', 6 * 1024 * 1024))
    expect(await screen.findByText('ছবিটি খুব বড় · সর্বোচ্চ ৫MB')).toBeInTheDocument()
    expect(onChange).not.toHaveBeenCalled()
  })

  it('previews a chosen photo and clears it on remove', async () => {
    const user = userEvent.setup()
    setup('data:image/jpeg;base64,abc')
    // The preview carries a real alt, and the attach box is gone.
    expect(screen.getByRole('img', { name: 'ছবি' })).toBeInTheDocument()
    expect(screen.queryByText('ছবি যোগ করুন')).toBeNull()

    await user.click(screen.getByRole('button', { name: 'সরিয়ে ফেলুন' }))
    expect(onChange).toHaveBeenCalledWith('')
  })

  it('accepts a photo dropped onto the attach box', async () => {
    setup()
    const file = makeFile('dropped.jpg', 2048)
    fireEvent.drop(screen.getByText('ছবি যোগ করুন').closest('label'), {
      dataTransfer: { files: [file] },
    })
    await waitFor(() => expect(onChange).toHaveBeenCalledTimes(1))
    expect(onChange.mock.calls[0][0]).toMatch(/^data:image\/jpeg;base64,/)
  })

  // The bug this component shipped with: the photo was emitted raw, and a phone
  // photo's data URL blew the server's 1 MiB body cap — so every report WITH a
  // photo 413'd, and the form blamed the report.
  it('compresses the photo rather than emitting the raw file', async () => {
    const user = userEvent.setup()
    setup()
    const file = makeFile('phone-photo.jpg', 4 * 1024 * 1024)
    await user.upload(screen.getByLabelText(/ছবি/), file)

    await waitFor(() => expect(onChange).toHaveBeenCalledTimes(1))
    expect(compressImage).toHaveBeenCalledWith(file)
    expect(onChange).toHaveBeenCalledWith('data:image/jpeg;base64,compressed')
  })

  it('never emits a photo it could not compress, and says so', async () => {
    const user = userEvent.setup()
    vi.mocked(compressImage).mockRejectedValue(new Error('image too large after compression'))
    setup()
    await user.upload(screen.getByLabelText(/ছবি/), makeFile('stubborn.jpg', 4 * 1024 * 1024))

    expect(await screen.findByText('ছবিটি ব্যবহার করা গেল না · অন্য একটি ছবি দিন')).toBeInTheDocument()
    // Emitting anyway would post a body the server rejects, failing the whole
    // report over an optional field.
    expect(onChange).not.toHaveBeenCalled()
  })

  it('ignores a dropped non-image', async () => {
    setup()
    const pdf = new File(['x'], 'notes.pdf', { type: 'application/pdf' })
    fireEvent.drop(screen.getByText('ছবি যোগ করুন').closest('label'), {
      dataTransfer: { files: [pdf] },
    })
    // Nothing emitted, and no error shouted at someone who dropped by accident.
    await new Promise((r) => setTimeout(r, 50))
    expect(onChange).not.toHaveBeenCalled()
    expect(screen.queryByText('ছবিটি খুব বড় · সর্বোচ্চ ৫MB')).toBeNull()
  })
})
