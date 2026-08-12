import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import DateRangeFilter from './DateRangeFilter.jsx'

// A tiny host, because the control is fully controlled: testing it against a
// frozen prop would prove the chips render and nothing about whether choosing
// one and typing a date can end up disagreeing, which is the whole risk here.
function Host({ direction }) {
  const [value, setValue] = useState({ preset: 'all', from: '', to: '' })
  return (
    <>
      <DateRangeFilter value={value} onChange={setValue} label="রিপোর্টের তারিখ" direction={direction} />
      <output data-testid="state">{`${value.preset}|${value.from}|${value.to}`}</output>
    </>
  )
}

beforeEach(() => {
  vi.useFakeTimers({ shouldAdvanceTime: true })
  vi.setSystemTime(new Date(2026, 6, 15, 12, 0, 0))
})

afterEach(() => {
  vi.useRealTimers()
})

describe('DateRangeFilter', () => {
  it('names the group with the label the page gave it', () => {
    render(<Host />)
    expect(screen.getByRole('heading', { name: 'রিপোর্টের তারিখ' })).toBeInTheDocument()
  })

  // Every label the app draws is Bangla. The browser's own picker chrome is not
  // ours to localise — which is exactly why the presets carry the common cases.
  it('offers the backward presets in Bangla by default', () => {
    render(<Host />)
    expect(screen.getAllByRole('button').map((b) => b.textContent)).toEqual([
      'সব সময়',
      'আজ',
      'গত ৭ দিন',
      'গত ৩০ দিন',
      'এই মাস',
    ])
  })

  it('offers forward presets for a list of deadlines', () => {
    render(<Host direction="future" />)
    expect(screen.getAllByRole('button').map((b) => b.textContent)).toEqual([
      'সব সময়',
      'আজ',
      'আগামী ৭ দিন',
      'আগামী ৩০ দিন',
      'এই মাস',
    ])
  })

  it('fills both date fields when a preset is chosen', async () => {
    const user = userEvent.setup()
    render(<Host />)

    await user.click(screen.getByRole('button', { name: 'গত ৭ দিন' }))

    expect(screen.getByTestId('state')).toHaveTextContent('last7|2026-07-09|2026-07-15')
    expect(screen.getByLabelText('শুরুর তারিখ')).toHaveValue('2026-07-09')
    expect(screen.getByLabelText('শেষ তারিখ')).toHaveValue('2026-07-15')
  })

  it('marks the chosen preset pressed, and only that one', async () => {
    const user = userEvent.setup()
    render(<Host />)

    await user.click(screen.getByRole('button', { name: 'আজ' }))

    expect(screen.getByRole('button', { name: 'আজ' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'সব সময়' })).toHaveAttribute('aria-pressed', 'false')
  })

  // The one way the two halves could lie to each other: a chip left highlighted
  // over dates it no longer describes. Typing a date drops the preset to custom.
  it('clears the active preset when a date is typed by hand', async () => {
    const user = userEvent.setup()
    render(<Host />)

    await user.click(screen.getByRole('button', { name: 'গত ৭ দিন' }))
    expect(screen.getByRole('button', { name: 'গত ৭ দিন' })).toHaveAttribute('aria-pressed', 'true')

    await user.clear(screen.getByLabelText('শুরুর তারিখ'))
    await user.type(screen.getByLabelText('শুরুর তারিখ'), '2026-07-01')

    expect(screen.getByRole('button', { name: 'গত ৭ দিন' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByTestId('state')).toHaveTextContent('custom|2026-07-01|2026-07-15')
  })

  it('resets to an unbounded range on "সব সময়"', async () => {
    const user = userEvent.setup()
    render(<Host />)

    await user.click(screen.getByRole('button', { name: 'গত ৩০ দিন' }))
    await user.click(screen.getByRole('button', { name: 'সব সময়' }))

    expect(screen.getByTestId('state')).toHaveTextContent('all||')
  })

  // Nothing was reported tomorrow, so a past-facing list must not offer the date.
  // A deadline list must — most deadlines are ahead of you.
  it('caps a past-facing range at today, and leaves a deadline range open', () => {
    const { unmount } = render(<Host />)
    expect(screen.getByLabelText('শেষ তারিখ')).toHaveAttribute('max', '2026-07-15')
    unmount()

    render(<Host direction="future" />)
    expect(screen.getByLabelText('শেষ তারিখ')).not.toHaveAttribute('max')
  })

  // Labels are always visible above the field — never placeholder-only (A.6).
  it('labels both date fields visibly', () => {
    render(<Host />)
    expect(screen.getByLabelText('শুরুর তারিখ').tagName).toBe('INPUT')
    expect(screen.getByLabelText('শেষ তারিখ')).toHaveAttribute('type', 'date')
  })
})
