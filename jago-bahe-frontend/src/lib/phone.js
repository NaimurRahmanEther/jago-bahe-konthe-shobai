/**
 * One phone number, many spellings.
 *
 * This mirrors the backend's `valueobject.normalize` /
 * `internal/shared/domain/valueobject/phone_number.go` — the same established
 * pattern as `ProblemFeed`'s STATUSES mirroring `PublicStatuses()` and
 * `models.js` mirroring the contract's shapes. **The backend is the authority**
 * (A.4.6): it normalizes every phone it is given, and nothing here can widen or
 * narrow what it accepts. This copy exists only so the form can answer instantly
 * instead of spending a round-trip to say "that is not a phone number".
 *
 * Why it exists at all: registration and the next login are typed by the same
 * person on different days, and the login field is `autoComplete="tel"` — so the
 * browser fills in whatever it stored, commonly `+880 1712-345678`, against a
 * number that was typed plainly as `01712345678`. Those are the same number to a
 * human and were two different accounts to the database, which is what made a
 * second login reject credentials that had just worked.
 *
 * Keep this in step with the Go file if the rule ever changes.
 */

/** The canonical form: an 11-digit Bangladeshi mobile number. */
export const PHONE_PATTERN = /^01[0-9]{9}$/

/** U+09E6 '০', the first of the ten Bengali digits. */
const BENGALI_ZERO = 0x09e6

/**
 * Reduce the ways one number gets written down to the single canonical form.
 *
 * Deliberately narrow: it removes only decoration — separators, the country
 * code, Bengali numerals — and never repairs a number that is genuinely wrong.
 * A non-number in, a non-number out; `PHONE_PATTERN` is what judges the result.
 *
 * @param {string} raw
 * @returns {string} the canonical `01XXXXXXXXX`, or the input reduced as far as
 *   these rules allow if it is not a Bangladeshi mobile number
 */
export function normalizePhone(raw) {
  let s = String(raw ?? '')
    .trim()
    .replace(/[\s\-().]/g, '')
    .replace(/[০-৯]/g, (d) => String(d.charCodeAt(0) - BENGALI_ZERO))

  if (s.startsWith('+')) s = s.slice(1)

  // Scoped to the exact country-code length so a foreign number is never
  // salvaged into a local one: a canonical number is 11 characters, so this can
  // only ever match a +880/880-prefixed one.
  if (s.length === 13 && s.startsWith('880')) s = `0${s.slice(3)}`

  return s
}
