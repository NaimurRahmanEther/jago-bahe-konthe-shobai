import { useCallback, useMemo, useState } from 'react'
import { inRange, presetRange } from '../lib/date.js'

/**
 * The date-range filter's state, shared by every list that offers one.
 *
 * Nine pages hold this filter; without a hook each would re-implement the same
 * small state machine, and they would drift on the one part that is easy to get
 * wrong — keeping the chips and the two date fields from disagreeing. There is
 * ONE source of truth here: choosing a preset fills `from`/`to`, and typing a
 * date drops the preset to `custom`.
 *
 * The filtering is client-side, and that is sound rather than a shortcut: no
 * list endpoint in this app paginates, so the rows a page holds ARE every row
 * the server would have matched. Narrowing them here is exactly equivalent to
 * narrowing them there — and it is presentation, not authority (golden rule 7),
 * the same standing that lets ProblemFeed filter status client-side.
 *
 * @returns {{
 *   range: {preset: string, from: string, to: string},
 *   setRange: (next: {preset: string, from: string, to: string}) => void,
 *   inRange: (iso: string) => boolean,
 *   isActive: boolean,
 * }}
 */
export function useDateRange() {
  const [range, setRange] = useState({ preset: 'all', from: '', to: '' })

  const isActive = Boolean(range.from || range.to)

  // Memoized on the bounds alone, so a page can list it in a useMemo dependency
  // array without recomputing its rows on every render.
  const test = useCallback((iso) => inRange(iso, range), [range])

  return useMemo(
    () => ({ range, setRange, inRange: test, isActive }),
    [range, test, isActive],
  )
}

/**
 * `useDateRange` plus the filtering, for the pages that are just "a list, narrowed".
 *
 * Seven pages hold the identical three lines — a range, a memoized filter over
 * one date field, and a count of what was hidden. Keeping them here is what stops
 * one of those pages quietly filtering on the wrong field, or forgetting the memo
 * and re-filtering the list on every keystroke in the date input.
 *
 * `dateKey` is the field the ROW ITSELF SHOWS, never merely the one that sorts it.
 * A filter over a date the reader cannot see on the row is unfalsifiable to them:
 * the queues filter `createdAt`, which their rows print, and the official's pages
 * filter `deadline`, which theirs do.
 *
 * @template T
 * @param {T[]|undefined} rows
 * @param {string} dateKey
 * @returns {{
 *   range: {preset: string, from: string, to: string},
 *   setRange: (next: {preset: string, from: string, to: string}) => void,
 *   isActive: boolean,
 *   visible: T[],
 *   total: number,
 * }}
 */
export function useDateFiltered(rows, dateKey) {
  const { range, setRange, inRange: test, isActive } = useDateRange()

  const visible = useMemo(() => (rows ?? []).filter((r) => test(r?.[dateKey])), [rows, dateKey, test])

  return { range, setRange, isActive, visible, total: rows?.length ?? 0 }
}

export { presetRange }
