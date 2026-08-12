import { useTranslation } from 'react-i18next'

/**
 * Calm loading placeholder — shimmer bars inside a card, matching real row shape.
 * Honors reduced-motion (pulse disabled) per Design Guideline §A.6.
 */
export function SkeletonCard() {
  return (
    <div className="rounded-card border border-hairline bg-surface p-4">
      <div className="h-4 w-3/4 animate-pulse rounded bg-canvas motion-reduce:animate-none" />
      <div className="mt-3 h-3 w-1/2 animate-pulse rounded bg-canvas motion-reduce:animate-none" />
      <div className="mt-2 h-3 w-1/3 animate-pulse rounded bg-canvas motion-reduce:animate-none" />
    </div>
  )
}

/**
 * A short stack of SkeletonCards.
 *
 * Announced as a live region so the wait is perceivable to a screen reader, which
 * otherwise meets silence where a sighted user sees shimmer.
 */
export default function SkeletonList({ count = 3 }) {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col gap-3" role="status" aria-label={t('common.loading')} aria-busy="true">
      {Array.from({ length: count }).map((_, i) => (
        <SkeletonCard key={i} />
      ))}
    </div>
  )
}
