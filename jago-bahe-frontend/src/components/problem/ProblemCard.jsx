import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import StatusBadge from './StatusBadge.jsx'
import ValidationCount from './ValidationCount.jsx'
import AssignedOfficial from './AssignedOfficial.jsx'
import { useOfficials } from '../../hooks/useOfficials.js'

function daysAgo(isoDate) {
  return Math.max(0, Math.floor((Date.now() - new Date(isoDate).getTime()) / 86400000))
}

/**
 * One problem in a list.
 *
 * The card's job is to be readable at a glance, so it is built as a ladder: the
 * status is what you look for first and gets its own line, the title is the
 * content and wins the card at 21px, and where/who is meta at 14px. It used to
 * be a 17px title over two identical lines of grey — the address, the official,
 * the date and the validation count all rendered the same, so nothing told the
 * eye what mattered.
 *
 * The card shows the validation RESULT — how many neighbours have said this is
 * real — and nothing to act on. Casting the vote lives on the detail page, where
 * the reader has the description in front of them: "is this real?" deserves to be
 * answered after reading the report, not from a title and an address.
 *
 * That is also what lets the whole card stay one link. Vote buttons were tried
 * here and taken back out; they forced the anchor to narrow to the title, which
 * cost the card its full-size tap target for a control that was asking people to
 * judge a report they had not opened.
 *
 * @param {{problem: import('../../lib/types/models.js').Problem, index?: number}} props
 */
export default function ProblemCard({ problem, index = 0 }) {
  const { t } = useTranslation()
  const { data: officials } = useOfficials()
  const official = officials?.find((o) => o.id === problem.pointedOfficialId)
  const days = daysAgo(problem.createdAt)

  return (
    <Link
      to={`/problems/${problem.id}`}
      style={{ animationDelay: `${Math.min(index, 5) * 40}ms` }}
      className="group flex animate-rise-in flex-col gap-2.5 rounded-card border border-hairline bg-surface p-4 transition duration-150 ease-standard hover:border-brand hover:shadow-md"
    >
      <div className="flex items-center justify-between gap-2">
        <StatusBadge status={problem.status} className="shrink-0" />
        <span className="shrink-0 text-micro tabular-nums text-muted">
          {days === 0 ? t('problem.card.today') : t('problem.card.daysAgo', { count: days })}
        </span>
      </div>

      <div className="flex gap-3">
        <div className="flex min-w-0 flex-1 flex-col gap-2.5">
          <h3 className="text-balance text-title font-semibold text-ink transition-colors duration-150 ease-standard group-hover:text-brand-dark">
            {problem.title}
          </h3>

          <p className="text-meta text-muted">
            {problem.location.address}
            {official && <> · {official.name}</>}
          </p>

          {/* Who is actually working on it, once someone is. Rendered as its own
              line rather than folded into the meta above, because "an official has
              taken this up" is a different order of fact from where the problem is
              — and it is the thing the platform exists to make visible. Absent
              until the problem is assigned, which is most rows on a busy feed. */}
          <AssignedOfficial problem={problem} compact />

          <ValidationCount count={problem.validCount} />
        </div>

        {/* A thumbnail, not the photo: the card's job is "there is a photo, roughly
            of this", so a square crop is right and the full image is one tap away.
            It used to render at up to 384px tall, which turned a feed of photo
            reports into a column of pictures you had to scroll past to read. */}
        {problem.imageUrl && (
          <img
            src={problem.imageUrl}
            alt=""
            loading="lazy"
            className="size-24 shrink-0 rounded-control border border-hairline object-cover"
          />
        )}
      </div>

      {/* The whole card has always been the link; this is the first thing that
          says so. It MUST stay a <span> — an <a> or <button> here would be an
          interactive element nested inside an anchor, which is invalid HTML and
          breaks keyboard navigation. It is an affordance for existing behaviour,
          not a second control. This is exactly the wall the feed vote buttons hit,
          and the reason they went back to the detail page rather than the anchor
          being broken up to fit them. */}
      <span className="-mb-1 mt-0.5 flex min-h-11 items-center gap-1 border-t border-hairline pt-2 text-meta font-medium text-brand">
        {t('problem.card.details')} <span aria-hidden="true">›</span>
      </span>
    </Link>
  )
}
