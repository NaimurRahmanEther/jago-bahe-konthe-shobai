import { describe, it, expect, vi, afterEach } from 'vitest'
import { formatDate, dayStart, dayEnd, inRange, toInputValue, presetRange, presetsFor } from './date.js'

afterEach(() => {
  vi.useRealTimers()
})

describe('inRange — the local-day boundary', () => {
  // THE test in this file. `new Date('2026-07-03')` is UTC midnight by
  // specification, which in Bangladesh (UTC+6) is 06:00 on the 3rd — so a range
  // built that way silently drops every report filed before six in the morning.
  // The list just quietly has fewer rows in it and nothing says why.
  it('includes a record timestamped early on the from-day, in local time', () => {
    const oneAmLocal = new Date(2026, 6, 3, 1, 0, 0).toISOString()
    expect(inRange(oneAmLocal, { from: '2026-07-03', to: '2026-07-03' })).toBe(true)
  })

  it('includes a record timestamped late on the to-day, in local time', () => {
    const elevenPmLocal = new Date(2026, 6, 3, 23, 30, 0).toISOString()
    expect(inRange(elevenPmLocal, { from: '2026-07-03', to: '2026-07-03' })).toBe(true)
  })

  it('excludes the last moment of the day before', () => {
    const justBefore = new Date(2026, 6, 2, 23, 59, 59).toISOString()
    expect(inRange(justBefore, { from: '2026-07-03' })).toBe(false)
  })

  it('excludes the first moment of the day after', () => {
    const justAfter = new Date(2026, 6, 4, 0, 0, 1).toISOString()
    expect(inRange(justAfter, { to: '2026-07-03' })).toBe(false)
  })
})

describe('inRange — open and empty bounds', () => {
  const iso = new Date(2026, 6, 15, 12, 0, 0).toISOString()

  it('accepts everything when neither bound is set', () => {
    expect(inRange(iso, {})).toBe(true)
    expect(inRange(iso, { from: '', to: '' })).toBe(true)
    expect(inRange(iso)).toBe(true)
  })

  it('treats a missing end as unbounded in that direction', () => {
    expect(inRange(iso, { from: '2026-07-01' })).toBe(true)
    expect(inRange(iso, { to: '2026-07-31' })).toBe(true)
    expect(inRange(iso, { from: '2026-08-01' })).toBe(false)
  })

  // A record whose timestamp cannot be read must stay VISIBLE. Hiding one because
  // we could not parse its date is the worse failure on a platform whose whole
  // promise is that nothing quietly disappears from the record.
  it('keeps a row whose date is missing or unreadable', () => {
    expect(inRange(undefined, { from: '2026-07-01', to: '2026-07-31' })).toBe(true)
    expect(inRange('not-a-date', { from: '2026-07-01', to: '2026-07-31' })).toBe(true)
  })

  it('ignores a malformed bound rather than hiding everything', () => {
    expect(inRange(iso, { from: '03/07/2026' })).toBe(true)
  })
})

describe('dayStart / dayEnd', () => {
  it('anchors to local midnight, not UTC midnight', () => {
    const start = dayStart('2026-07-03')
    expect(start.getFullYear()).toBe(2026)
    expect(start.getMonth()).toBe(6)
    expect(start.getDate()).toBe(3)
    expect(start.getHours()).toBe(0)
  })

  it('ends the day at the last millisecond, so the to-bound is inclusive', () => {
    const end = dayEnd('2026-07-03')
    expect(end.getDate()).toBe(3)
    expect(end.getHours()).toBe(23)
    expect(end.getMinutes()).toBe(59)
  })

  it('returns null for an empty or malformed value', () => {
    expect(dayStart('')).toBeNull()
    expect(dayStart(undefined)).toBeNull()
    expect(dayStart('2026-7-3')).toBeNull()
    expect(dayEnd('nonsense')).toBeNull()
  })
})

describe('toInputValue', () => {
  // toISOString().slice(0,10) would name YESTERDAY for anyone east of Greenwich
  // after 18:00 — so "আজ" would quietly filter to the wrong day every evening.
  it('names the local day, not the UTC one', () => {
    expect(toInputValue(new Date(2026, 6, 3, 23, 30))).toBe('2026-07-03')
  })

  it('zero-pads, because the date input demands YYYY-MM-DD', () => {
    expect(toInputValue(new Date(2026, 0, 5))).toBe('2026-01-05')
  })
})

describe('presetRange', () => {
  // 15 July 2026, local noon.
  const pin = () => vi.setSystemTime(new Date(2026, 6, 15, 12, 0, 0))

  it('reads "today" as exactly one day', () => {
    vi.useFakeTimers()
    pin()
    expect(presetRange('today')).toEqual({ from: '2026-07-15', to: '2026-07-15' })
  })

  // Seven days INCLUDING today, not a seven-day gap ending yesterday.
  it('counts the last seven days inclusive of today', () => {
    vi.useFakeTimers()
    pin()
    expect(presetRange('last7')).toEqual({ from: '2026-07-09', to: '2026-07-15' })
  })

  it('counts the last thirty days inclusive of today', () => {
    vi.useFakeTimers()
    pin()
    expect(presetRange('last30')).toEqual({ from: '2026-06-16', to: '2026-07-15' })
  })

  // The forward presets exist because deadlines are mostly ahead of you: an
  // official asking what falls due next week has no backward preset that answers.
  it('counts the next seven days inclusive of today', () => {
    vi.useFakeTimers()
    pin()
    expect(presetRange('next7')).toEqual({ from: '2026-07-15', to: '2026-07-21' })
  })

  it('spans the whole calendar month, including days still to come', () => {
    vi.useFakeTimers()
    pin()
    expect(presetRange('thisMonth')).toEqual({ from: '2026-07-01', to: '2026-07-31' })
  })

  it('returns an empty, unbounded range for "all" and for anything unknown', () => {
    expect(presetRange('all')).toEqual({ from: '', to: '' })
    expect(presetRange('custom')).toEqual({ from: '', to: '' })
  })
})

describe('presetsFor', () => {
  it('offers backward presets for things that happened', () => {
    expect(presetsFor('past')).toEqual(['all', 'today', 'last7', 'last30', 'thisMonth'])
    expect(presetsFor()).toEqual(['all', 'today', 'last7', 'last30', 'thisMonth'])
  })

  it('offers forward presets for deadlines', () => {
    expect(presetsFor('future')).toEqual(['all', 'today', 'next7', 'next30', 'thisMonth'])
  })
})

describe('formatDate', () => {
  // Bengali digits come from Intl, never from a hand-rolled digit swap (A.5.3
  // rule 5). If this ever renders 2026 in Latin, the locale has been lost.
  it('renders the date in Bengali numerals', () => {
    const out = formatDate('2026-07-03T09:00:00Z')
    expect(out).toMatch(/[০-৯]/)
    expect(out).not.toMatch(/[0-9]/)
  })

  it('offers three shapes, and the numeric one is the shortest', () => {
    const iso = '2026-07-03T09:00:00Z'
    expect(formatDate(iso, 'numeric').length).toBeLessThan(formatDate(iso, 'long').length)
  })

  it('returns an empty string rather than "Invalid Date" on the page', () => {
    expect(formatDate(undefined)).toBe('')
    expect(formatDate('nonsense')).toBe('')
  })
})
