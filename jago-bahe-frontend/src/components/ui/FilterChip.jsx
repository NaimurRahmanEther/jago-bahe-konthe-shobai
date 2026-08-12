/**
 * One filter chip — the pill that turns a filter on and off.
 *
 * Extracted from ProblemFeed, which carried this six-line class string twice
 * (the "all" chip and the per-status chips) before the date filter would have
 * made it four (A.5 rule 8: a repeated Tailwind class string becomes a
 * component). Keeping one copy is also what keeps the date chips visually
 * identical to the status chips they sit beside.
 *
 * `aria-pressed` rather than a role of its own: this is a toggle button, and
 * the pressed state is what a screen reader needs to hear. Colour is never the
 * only signal — the active chip changes border, weight and background together.
 *
 * @param {{
 *   active: boolean,
 *   onClick: () => void,
 *   children: import('react').ReactNode,
 * }} props
 */
export default function FilterChip({ active, onClick, children }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex min-h-9 items-center gap-1.5 rounded-full border px-3 text-[13px] transition duration-150 ease-standard ${
        active
          ? 'border-brand bg-brand-tint font-semibold text-brand-dark'
          : 'border-hairline bg-surface text-muted hover:border-brand hover:text-brand-dark'
      }`}
    >
      {children}
    </button>
  )
}
