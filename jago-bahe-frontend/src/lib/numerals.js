// Bengali numerals via Intl, which also gets the things a digit-swap would miss:
// grouping (১,২৩৪) and the decimal separator (৫.২). Constructed once — building
// an Intl formatter per render is measurably slow.
const bengali = new Intl.NumberFormat('bn')

/**
 * Render a number in Bengali numerals.
 *
 * Strings inside `bn.json` do NOT come through here: they use i18next's own
 * `{{value, number}}` formatter, which is this same Intl call under the hood.
 * Use this only where a number reaches the DOM without passing through `t()` —
 * a StatTile value, a filter count (A.5.3).
 *
 * @param {number|string} value
 * @returns {string}
 */
export function toBengaliDigits(value) {
  return typeof value === 'number' && Number.isFinite(value) ? bengali.format(value) : String(value)
}
