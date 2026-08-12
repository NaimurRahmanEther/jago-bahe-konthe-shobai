// Dates: the one formatter, and the maths behind the date-range filter.
//
// Formatting was duplicated in seven components before this file existed — three
// local `formatDate` closures and four inline `toLocaleDateString(i18n.language)`
// calls, in three different shapes. They now all come through `formatDate`.
//
// The locale is hardcoded 'bn', exactly as lib/numerals.js hardcodes it: there is
// one language and no fallback (A.5.3), so reading it back off i18n only made
// seven components depend on the i18n instance to print a date. Bengali digits
// come free from Intl here — never hand-roll a digit swap (A.5.3 rule 5).
//
// Formatters are constructed once. Building an Intl formatter per render is
// measurably slow, and these run per row on every list in the app.
const FORMATTERS = {
  // The compact one, for the quiet audit/activity trails (Guideline §6: "actor ·
  // action · date"). A named month there would out-shout the actor.
  numeric: new Intl.DateTimeFormat('bn'),
  // The default: readable on a row that is about the date.
  short: new Intl.DateTimeFormat('bn', { year: 'numeric', month: 'short', day: 'numeric' }),
  // For a heading-adjacent date the reader is meant to dwell on.
  long: new Intl.DateTimeFormat('bn', { year: 'numeric', month: 'long', day: 'numeric' }),
}

/**
 * Render an ISO timestamp as a Bangla date.
 *
 * @param {string} iso
 * @param {'numeric'|'short'|'long'} [style]
 * @returns {string}
 */
export function formatDate(iso, style = 'short') {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return (FORMATTERS[style] ?? FORMATTERS.short).format(d)
}

/**
 * Local midnight at the start of a `YYYY-MM-DD` day.
 *
 * Parsed field by field, NOT via `new Date('2026-07-03')` — that form is defined
 * to be UTC midnight, which in Bangladesh (UTC+6) is 06:00 on the 3rd. A range
 * built that way silently drops every report filed before six in the morning,
 * and the bug is invisible: the list just quietly has fewer rows in it. This is
 * why date.test.js leads with the boundary case.
 *
 * @param {string} value `YYYY-MM-DD`
 * @returns {Date|null} null when the value is empty or unparseable
 */
export function dayStart(value) {
  const parts = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value ?? '')
  if (!parts) return null
  const d = new Date(Number(parts[1]), Number(parts[2]) - 1, Number(parts[3]), 0, 0, 0, 0)
  return Number.isNaN(d.getTime()) ? null : d
}

/**
 * The last instant of a `YYYY-MM-DD` day, local. Both ends of a range are
 * inclusive — a reader asking for "the 3rd" means all of it.
 *
 * @param {string} value `YYYY-MM-DD`
 * @returns {Date|null}
 */
export function dayEnd(value) {
  const start = dayStart(value)
  if (!start) return null
  return new Date(start.getFullYear(), start.getMonth(), start.getDate(), 23, 59, 59, 999)
}

/**
 * Is this timestamp inside the range? An empty bound is unbounded, so an empty
 * range accepts everything.
 *
 * A row whose date is missing or unparseable stays VISIBLE. Hiding a record
 * because we could not read its timestamp is the worse failure on a platform
 * whose point is that nothing quietly disappears from the record.
 *
 * @param {string} iso
 * @param {{from?: string, to?: string}} range
 * @returns {boolean}
 */
export function inRange(iso, range = {}) {
  const from = dayStart(range.from)
  const to = dayEnd(range.to)
  if (!from && !to) return true

  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return true

  if (from && t < from.getTime()) return false
  if (to && t > to.getTime()) return false
  return true
}

/**
 * A `YYYY-MM-DD` string for a Date, in LOCAL time — `toISOString().slice(0,10)`
 * would name yesterday for anyone east of Greenwich after 18:00.
 *
 * @param {Date} d
 * @returns {string}
 */
export function toInputValue(d) {
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** Today as `YYYY-MM-DD`. */
export function todayValue() {
  return toInputValue(new Date())
}

/**
 * Which way a list's dates point.
 *
 * 'past' for the dates something HAPPENED on — a report filed, a decision taken.
 * Nothing was reported tomorrow, so those fields are capped at today.
 * 'future' for a DEADLINE, which is mostly ahead of you: an official asking what
 * is due next week must be able to say so, and backward-only presets would offer
 * them nothing but the overdue.
 *
 * @typedef {'past'|'future'} DateDirection
 */

/** @typedef {'all'|'today'|'last7'|'last30'|'next7'|'next30'|'thisMonth'|'custom'} DatePreset */

/**
 * The presets offered as chips, in order. `custom` is a state, never a chip.
 *
 * @param {DateDirection} [direction]
 * @returns {DatePreset[]}
 */
export function presetsFor(direction = 'past') {
  return direction === 'future'
    ? ['all', 'today', 'next7', 'next30', 'thisMonth']
    : ['all', 'today', 'last7', 'last30', 'thisMonth']
}

/**
 * The `{from, to}` a preset stands for. Every range INCLUDES today, so "গত ৭ দিন"
 * is today and the six days before it — the reading a person expects, not a
 * seven-day gap ending yesterday.
 *
 * @param {DatePreset} preset
 * @returns {{from: string, to: string}}
 */
export function presetRange(preset) {
  const now = new Date()
  const today = toInputValue(now)
  const shift = (n) => toInputValue(new Date(now.getFullYear(), now.getMonth(), now.getDate() + n))

  switch (preset) {
    case 'today':
      return { from: today, to: today }
    case 'last7':
      return { from: shift(-6), to: today }
    case 'last30':
      return { from: shift(-29), to: today }
    case 'next7':
      return { from: today, to: shift(6) }
    case 'next30':
      return { from: today, to: shift(29) }
    // "This month" is the whole calendar month either way. For deadlines that
    // includes days still to come, which is the question being asked.
    case 'thisMonth':
      return {
        from: toInputValue(new Date(now.getFullYear(), now.getMonth(), 1)),
        to: toInputValue(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
      }
    default:
      return { from: '', to: '' }
  }
}
