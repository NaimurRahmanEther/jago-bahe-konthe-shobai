import { useTranslation } from 'react-i18next'
import { useOfficials } from '../../hooks/useOfficials.js'

/**
 * Who is actually working on this report — and, when the admin passed over the
 * public's nominee, the reason they gave for it.
 *
 * This is the platform's central claim made visible: a report goes to a named
 * person holding a named office, in public. Until this existed the feed showed
 * what was reported and never who took it up, which left the accountable half of
 * the record private.
 *
 * Factual and unranked (Concept §10). It states who and why; it never grades the
 * choice, never calls an override wrong, and never editorialises about the gap.
 * Renders nothing at all when the problem has not been assigned — an empty
 * "unassigned" strip on every new report would be noise, and the status badge
 * already says where the report stands.
 *
 * @param {{problem: import('../../lib/types/models.js').Problem, compact?: boolean}} props
 */
export default function AssignedOfficial({ problem, compact = false }) {
  const { t } = useTranslation()
  const { data: officials } = useOfficials()

  const { assignedOfficialId, pointedOfficialId, overrideReason } = problem
  if (!assignedOfficialId) return null

  const assigned = officials?.find((o) => o.id === assignedOfficialId)
  // The directory is still loading, or the office has no directory entry. Say the
  // true thing — someone is on it — rather than rendering a blank strip that reads
  // as "nobody assigned", which is the opposite of the fact.
  const name = assigned ? assigned.name : t('problem.assigned.unknownOfficial')
  const tier = assigned ? t(`problem.tier.${assigned.tier}`) : null

  // An override is the assignment differing from what the public asked for. The
  // reason is the admin's own words and is required by the backend, so its absence
  // here means there was no override to explain, not that one went unjustified.
  const isOverride = Boolean(pointedOfficialId) && pointedOfficialId !== assignedOfficialId
  const pointed = officials?.find((o) => o.id === pointedOfficialId)

  if (compact) {
    return (
      <p className="mt-1.5 text-meta text-brand-dark">
        {t('problem.assigned.short', { name })}
      </p>
    )
  }

  // Deliberately NOT bg-brand-tint, which it used to be. The tint had come to mean
  // three different things on one page — this card, the top suggestion, and the
  // official's answered-suggestion echo — and that repetition was a large part of why
  // the page would not separate visually. It now means exactly one thing: the
  // COMMUNITY'S VOICE. That leaves the tint marking the top suggestion and the echo of
  // it in the official's plan, which are the same suggestion snapshotted — so the
  // colour becomes a thread the eye follows from what the community asked to what the
  // official answered, which is the accountability (Concept §4, Figure 3).
  //
  // Who was assigned is a fact about the office, not an endorsement by the public, so
  // it reads on plain surface with a brand-dark eyebrow.
  return (
    <section
      aria-label={t('problem.assigned.title')}
      className="rounded-card border border-hairline bg-surface p-4"
    >
      <p className="text-meta font-medium text-brand-dark">{t('problem.assigned.title')}</p>
      <p className="mt-1 text-title font-semibold text-ink">{name}</p>
      {tier && <p className="text-meta text-muted">{tier}</p>}

      {/* The override, shown as what it is: the public asked for one person, the
          admin chose another, and here is why. Publishing the gap while withholding
          the reason would invite the reader to assume the worst of a decision that
          may well be routine (a matter of jurisdiction, most often). */}
      {isOverride && (
        <div className="mt-3 border-t border-hairline pt-3">
          <p className="text-meta text-muted">
            {t('problem.assigned.publicChoice', {
              name: pointed ? pointed.name : t('problem.assigned.unknownOfficial'),
            })}
          </p>
          {overrideReason && (
            <p className="mt-1.5 text-ink">
              <span className="font-medium">{t('problem.assigned.overrideReason')}: </span>
              {overrideReason}
            </p>
          )}
        </div>
      )}
    </section>
  )
}
