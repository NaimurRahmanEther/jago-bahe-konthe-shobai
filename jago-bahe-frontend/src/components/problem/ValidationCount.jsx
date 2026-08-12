import { useTranslation } from 'react-i18next'

/**
 * How many neighbours have validated a problem — the public count, with no target.
 *
 * The public used to see "X / V" under a bar filling toward V. That stopped being
 * true at B17: reaching V no longer unlocks anything, an admin may forward a report
 * to an official at any count, and V is the community's endorsement rather than a
 * gate (CLAUDE.md A.3.1.1). A denominator advertised a mechanism that is not there,
 * and a bar drew a finish line the report does not have to cross.
 *
 * `ValidationBar` still shows "X / V" on ADMIN surfaces, where V is exactly the
 * thing being weighed. Do not reunify these two — they answer different questions
 * for different readers.
 *
 * Numbers render in Bengali numerals via i18next's built-in `number` formatter
 * (A.5.3 rule 5) — never hand-roll a digit swap.
 *
 * @param {{count: number, className?: string}} props
 */
export default function ValidationCount({ count, className = '' }) {
  const { t } = useTranslation()

  // "০ জন যাচাই করেছেন" reads as a statistic about nobody; the zero case wants its
  // own sentence, not a number.
  return (
    <p className={`text-micro tabular-nums text-muted ${className}`}>
      {count > 0 ? t('problem.validate.count', { count }) : t('problem.validate.countZero')}
    </p>
  )
}
