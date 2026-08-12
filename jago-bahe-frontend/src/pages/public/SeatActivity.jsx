import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useActivity } from '../../hooks/useActivity.js'
import { useDateRange } from '../../hooks/useDateRange.js'
import ActivityFeed from '../../components/audit/ActivityFeed.jsx'
import Section from '../../components/ui/Section.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'

// This is the ONE list in the app the server truncates (the endpoint defaults to
// the most recent 100 decisions). Every other page filters a complete set, so a
// date range there is exactly a server-side range; here it is a range over a
// window, and asking for an old date could show "nothing" when the record simply
// stops short. So a range asks for a deeper window — ?limit= is already part of
// the contract — and if the answer still comes back full, the page SAYS so.
// Silence would be this feed lying by omission, which is the one thing a public
// accountability record must never do.
const RANGE_LIMIT = 500

/**
 * The seat's public decision record — every approval, rejection, assignment, vote
 * and adjudication the union admins have made, for anyone to read.
 *
 * This is the public half of what used to be the super admin's private oversight
 * feed. The decisions were always visible one report at a time; what was missing
 * was the seat-wide view, which is the only one where a pattern shows — a union
 * admin quietly rejecting everything is invisible report by report and obvious
 * here. Keeping that view behind a single appointed account was backwards for a
 * transparency platform: the people who most need it are residents.
 *
 * There is deliberately NO action on this page, exactly as on /super. Publishing
 * the record widens who does the overseeing; it does not add a power to override
 * (Scaffold §2). The remedy for an admin abusing their position is removing them
 * in public — which is what being able to see this makes possible.
 */
export default function SeatActivity() {
  const { t } = useTranslation()
  const dates = useDateRange()
  const { data: entries, isLoading, isError, refetch } = useActivity(
    dates.isActive ? { limit: RANGE_LIMIT } : {},
  )

  const visible = useMemo(
    () => (entries ?? []).filter((e) => dates.inRange(e.createdAt)),
    [entries, dates],
  )
  const truncated = dates.isActive && entries?.length === RANGE_LIMIT

  return (
    <div className="flex w-full flex-col gap-6">
      <header className="flex flex-col gap-1">
        <h1 className="text-display font-semibold text-ink">{t('activity.title')}</h1>
        <p className="max-w-[70ch] text-muted">{t('activity.subtitle')}</p>
      </header>

      {/* Outside the record, deliberately. The filter narrows what this reader is
          looking at and decides nothing about any report — but the region below
          is the one that must stay free of controls (Scaffold §2, and the test
          that pins it), so the two are kept apart rather than argued about. */}
      <DateRangeFilter value={dates.range} onChange={dates.setRange} label={t('filter.date.decisionDate')} />

      {truncated && (
        <p className="rounded-control border border-hairline bg-canvas px-3 py-2 text-meta text-muted">
          {t('filter.date.truncated', { count: RANGE_LIMIT })}
        </p>
      )}

      <Section
        title={t('activity.sectionTitle')}
        meta={visible.length ? t('activity.count', { count: visible.length }) : undefined}
      >
        {isLoading && <SkeletonList count={4} />}

        {isError && <ErrorState message={t('activity.error')} onRetry={refetch} />}

        {/* An empty record is not a failure — it is a seat where no admin has yet
            had to decide anything. Say that, rather than showing a broken-looking
            blank. A chosen span that happens to be quiet is a different fact and
            gets its own line, so neither is mistaken for the other. */}
        {!isLoading && !isError && visible.length === 0 && (
          <EmptyState
            title={dates.isActive ? t('filter.date.emptyRange') : t('activity.empty')}
            description={dates.isActive ? undefined : t('activity.emptyBody')}
          />
        )}

        {!isLoading && !isError && visible.length > 0 && <ActivityFeed entries={visible} />}
      </Section>
    </div>
  )
}
