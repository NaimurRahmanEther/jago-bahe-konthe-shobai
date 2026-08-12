import { useId } from 'react'

/**
 * A titled region: a real heading OUTSIDE, the content in a card INSIDE.
 *
 * Extracted per A.5.8 well past the third use — `rounded-card border border-hairline
 * bg-surface p-4` is hand-rolled 38 times across 26 files, and paired with a title in
 * 8+ of those. `Card` already covers the box; what kept being rewritten is the box
 * PLUS its heading, and the headings had drifted apart: `ProblemDetail` alone carried
 * one real `<h2 className="text-title">` and three `<p className="text-sm
 * font-semibold">` pretending to be headings. A `<p>` is not a heading, so pages
 * built that way have no outline below their `<h1>` at all.
 *
 * The shape follows `AdminHome`'s local `QueueSection`, which is the house idiom and
 * was the only instance of it: heading above the card, `gap-3` within a section and
 * `gap-6` between them. Boxing the content while leaving the title outside is what
 * makes adjacent sections separable at a glance — the thing four identical bordered
 * cards in a flat column cannot do.
 *
 * `aria-labelledby` is load-bearing rather than decoration. Bangla section titles
 * routinely repeat their own rows' status badges verbatim ("অনুমোদনের অপেক্ষায়" is
 * both a heading and a badge), so without an accessible name a region is ambiguous
 * to a screen reader and unaddressable in a test (A.5.3 rule 4).
 *
 * @param {{
 *   title: string,
 *   meta?: string,
 *   action?: React.ReactNode,
 *   boxed?: boolean,
 *   className?: string,
 *   children: React.ReactNode,
 * }} props
 */
export default function Section({ title, meta, action, boxed = true, className = '', children }) {
  const headingId = useId()

  return (
    <section aria-labelledby={headingId} className={`flex flex-col gap-3 ${className}`}>
      <div className="flex items-baseline justify-between gap-3">
        <h2 id={headingId} className="text-title font-semibold text-ink">
          {title}
          {/* `meta` is a STRING the caller has already put through t(), not a number.
              That is deliberate: a bare count rendered here would come out in Latin
              digits (as AdminHome's `({data.length})` still does), and A.5.3 rule 5
              wants it formatted by i18next's {{count, number}} in bn.json instead. */}
          {meta && <span className="ml-2 text-meta font-normal text-muted">{meta}</span>}
        </h2>
        {action}
      </div>

      {boxed ? (
        <div className="rounded-card border border-hairline bg-surface p-4">{children}</div>
      ) : (
        children
      )}
    </section>
  )
}
