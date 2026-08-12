import { useTranslation } from 'react-i18next'

/**
 * How close a problem is to the validity threshold V.
 *
 * Design Guideline §7 allows exactly one moving thing here — "a smooth fill on
 * the validation progress bar" — and it was specified but never built, so the
 * mechanism the whole platform turns on rendered as a grey sentence.
 *
 * DISPLAY ONLY. The flip to Validated is the backend's (golden rule 7): the bar
 * reaching its end is a picture of the count, never the thing that causes it.
 *
 * The label stays alongside the bar rather than being replaced by it — colour and
 * length alone must never carry meaning (Guideline §2).
 *
 * @param {{count: number, threshold: number, className?: string}} props
 */
export default function ValidationBar({ count, threshold, className = '' }) {
  const { t } = useTranslation()

  // A threshold of 0 would divide by zero; V is config-driven, so guard rather
  // than assume. Over-count clamps to full instead of overflowing the track.
  const pct = threshold > 0 ? Math.min(100, Math.round((count / threshold) * 100)) : 0
  const met = threshold > 0 && count >= threshold

  return (
    <div className={`flex flex-col gap-1.5 ${className}`}>
      <div
        className="h-1.5 w-full overflow-hidden rounded-full bg-brand-tint"
        role="progressbar"
        aria-valuenow={count}
        aria-valuemin={0}
        aria-valuemax={threshold}
        aria-label={t('problem.validate.progress', { count, threshold })}
      >
        <div
          className="h-full rounded-full bg-brand transition-[width] duration-200 ease-standard motion-reduce:transition-none"
          style={{ width: `${pct}%` }}
        />
      </div>
      <p className={`text-micro tabular-nums ${met ? 'font-semibold text-brand-dark' : 'text-muted'}`}>
        {t('problem.validate.progress', { count, threshold })}
      </p>
    </div>
  )
}
