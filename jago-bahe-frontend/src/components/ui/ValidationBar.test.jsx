import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import ValidationBar from './ValidationBar.jsx'

describe('ValidationBar', () => {
  // Bengali numerals are the reason this file exists. The first attempt wired an
  // `interpolation.format` hook that i18next 26 never calls, so every digit stayed
  // Latin and the whole suite passed anyway — nothing asserted the digits. This
  // test is what makes the numerals real rather than intended (A.5.2).
  it('renders the count in Bengali numerals, not Latin', () => {
    render(<ValidationBar count={3} threshold={5} />)
    expect(screen.getByText('৩ / ৫ জন যাচাই করেছেন')).toBeInTheDocument()
    expect(screen.queryByText('3 / 5 জন যাচাই করেছেন')).not.toBeInTheDocument()
  })

  it('exposes progress to assistive tech with the real numbers', () => {
    render(<ValidationBar count={3} threshold={5} />)
    const bar = screen.getByRole('progressbar')
    expect(bar).toHaveAttribute('aria-valuenow', '3')
    expect(bar).toHaveAttribute('aria-valuemax', '5')
  })

  it('marks the threshold as met once the count reaches it', () => {
    render(<ValidationBar count={5} threshold={5} />)
    expect(screen.getByText('৫ / ৫ জন যাচাই করেছেন')).toHaveClass('text-brand-dark')
  })

  // V is config-driven, so a zero threshold is a deployment away rather than
  // impossible; it must not divide by zero or overflow the track.
  it('survives a zero threshold and clamps an over-count', () => {
    const { rerender } = render(<ValidationBar count={2} threshold={0} />)
    expect(screen.getByRole('progressbar')).toBeInTheDocument()

    rerender(<ValidationBar count={9} threshold={5} />)
    const fill = screen.getByRole('progressbar').firstChild
    expect(fill).toHaveStyle({ width: '100%' })
  })
})
