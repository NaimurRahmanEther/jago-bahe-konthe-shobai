import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useProblems } from '../../hooks/useProblems.js'
import { useOfficials } from '../../hooks/useOfficials.js'
import { useAreas } from '../../hooks/useAreas.js'
import { useDateRange } from '../../hooks/useDateRange.js'
import { toBengaliDigits } from '../../lib/numerals.js'
import ProblemCard from '../../components/problem/ProblemCard.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'
import DateRangeFilter from '../../components/ui/DateRangeFilter.jsx'
import FilterChip from '../../components/ui/FilterChip.jsx'

// The frontend's copy of the backend's PublicStatuses() — every public state.
// PendingApproval is deliberately absent: a report awaiting screening is not
// public, so it is never a feed filter (it mirrors the backend's exclusion).
// Rejected and Withdrawn are included: a report taken down for spam, or retracted
// by its reporter, stays public with its reason, and being able to find one is the
// point. There is no GET /statuses and inventing one to avoid a 10-element array
// would be a contract violation for no gain; integration_test.go proves the mirror.
const STATUSES = [
  'Reported', 'Validated', 'Assigned', 'InProgress',
  'Blocked', 'Done', 'Resolved', 'Reopened', 'Rejected', 'Withdrawn',
]

// Where a problem stands in the lifecycle, which is what a reader actually scans
// for: is anyone looking at this yet, is someone working on it, is it over. The
// grouping is a view of Status — never a judgement about it — and every status
// appears in exactly one group, so nothing can silently fall out of the feed.
const GROUPS = [
  { id: 'atCommunity', statuses: ['Reported', 'Validated'] },
  { id: 'withOfficial', statuses: ['Assigned', 'InProgress', 'Blocked', 'Reopened', 'Done'] },
  { id: 'settled', statuses: ['Resolved', 'Rejected', 'Withdrawn'] },
]

export default function ProblemFeed() {
  const { t } = useTranslation()
  const [filters, setFilters] = useState({ area: '', status: '', official: '' })
  const dates = useDateRange()
  const { data: officials } = useOfficials()
  // The eight unions plus the pourashava — the level a problem's area resolves to.
  const { data: areaOptions, isLoading: areasLoading, isError: areasError } = useAreas({ level: 'union' })

  // Area and official narrow server-side; status does not. The chips need a count
  // per status, which is unknowable from a list the server already filtered — so
  // the status filter is applied here, over rows the server returned. Counting
  // states is presentation, not authority (the rule MyReports.jsx:26 follows);
  // nothing here computes anything verdict-shaped.
  const activeFilters = {
    area: filters.area || undefined,
    official: filters.official || undefined,
  }
  const { data: problems, isLoading, isError, refetch } = useProblems(activeFilters)

  // The date window is applied BEFORE the status counts, so the chips describe
  // the span the reader chose rather than contradicting it. Order through this
  // file: server rows -> date -> counts -> status -> groups.
  const dated = useMemo(() => (problems ?? []).filter((p) => dates.inRange(p.createdAt)), [problems, dates])

  const counts = useMemo(() => {
    const acc = {}
    for (const p of dated) acc[p.status] = (acc[p.status] ?? 0) + 1
    return acc
  }, [dated])

  const visible = useMemo(
    () => (filters.status ? dated.filter((p) => p.status === filters.status) : dated),
    [dated, filters.status],
  )

  // A single chosen status is already one group; grouping it would print a lone
  // heading over the whole list and say nothing.
  const groups = useMemo(() => {
    if (filters.status) return [{ id: null, rows: visible }]
    return GROUPS.map((g) => ({ id: g.id, rows: visible.filter((p) => g.statuses.includes(p.status)) })).filter(
      (g) => g.rows.length > 0,
    )
  }, [visible, filters.status])

  function updateFilter(key) {
    return (e) => setFilters((f) => ({ ...f, [key]: e.target.value }))
  }

  const total = dated.length

  return (
    <div className="flex w-full flex-col gap-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-h1 font-semibold text-ink">{t('problem.feed.title')}</h1>
          {!isLoading && !isError && (
            <p className="mt-0.5 text-meta text-muted">{t('problem.feed.count', { count: total })}</p>
          )}
        </div>
        <Link
          to="/report"
          className="inline-flex min-h-11 items-center rounded-control bg-brand px-4 text-sm font-medium text-white transition duration-150 ease-standard hover:bg-brand-dark active:scale-[0.98] motion-reduce:active:scale-100"
        >
          {t('problem.feed.report')}
        </Link>
      </div>

      {/* The status filter is chips rather than a select: it is the one filter whose
          shape is worth seeing at a glance, and it takes one tap instead of three.
          Only statuses actually present are offered — a chip that filters to nothing
          is a dead end. */}
      {!isLoading && !isError && total > 0 && (
        <div className="flex flex-wrap gap-2" role="group" aria-label={t('problem.feed.filterStatus')}>
          <FilterChip active={filters.status === ''} onClick={() => setFilters((f) => ({ ...f, status: '' }))}>
            {t('problem.feed.allStatuses')}
            <span className="tabular-nums opacity-75">{toBengaliDigits(total)}</span>
          </FilterChip>

          {STATUSES.filter((s) => counts[s]).map((s) => (
            <FilterChip
              key={s}
              active={filters.status === s}
              onClick={() => setFilters((f) => ({ ...f, status: f.status === s ? '' : s }))}
            >
              {t(`problem.status.${s}`)}
              <span className="tabular-nums opacity-75">{toBengaliDigits(counts[s])}</span>
            </FilterChip>
          ))}
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        {/* A filter, not a data view: its loading and error states live in the option
            list, because the feed below stays perfectly usable unfiltered either way. */}
        <select
          value={filters.area}
          onChange={updateFilter('area')}
          aria-label={t('problem.feed.filterArea')}
          disabled={areasLoading || areasError}
          className="min-h-11 rounded-control border border-hairline bg-surface px-3 text-sm text-ink transition-colors duration-150 ease-standard hover:border-brand disabled:text-muted disabled:hover:border-hairline"
        >
          {areasLoading && <option value="">{t('problem.feed.areasLoading')}</option>}
          {areasError && <option value="">{t('problem.feed.areasUnavailable')}</option>}
          {!areasLoading && !areasError && (
            <>
              <option value="">{t('problem.feed.allAreas')}</option>
              {(areaOptions ?? []).map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </>
          )}
        </select>

        <select
          value={filters.official}
          onChange={updateFilter('official')}
          aria-label={t('problem.feed.filterOfficial')}
          className="min-h-11 rounded-control border border-hairline bg-surface px-3 text-sm text-ink transition-colors duration-150 ease-standard hover:border-brand"
        >
          <option value="">{t('problem.feed.allOfficials')}</option>
          {(officials ?? []).map((o) => (
            <option key={o.id} value={o.id}>
              {o.name}
            </option>
          ))}
        </select>
      </div>

      {/* Narrowing by date is presentation, not authority: the endpoint returns
          every public report (nothing here paginates), so filtering the rows the
          page already holds is exactly what a server-side range would return. */}
      <DateRangeFilter value={dates.range} onChange={dates.setRange} label={t('filter.date.reportDate')} />

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('problem.feed.error')} onRetry={refetch} />}

      {!isLoading && !isError && visible.length === 0 && (
        <EmptyState
          // "Nothing in this span" and "nothing at all" are different facts, and
          // saying the second when the first is true reads as a broken feed.
          title={dates.isActive ? t('filter.date.emptyRange') : t('problem.feed.emptyTitle')}
          action={
            <Link
              to="/report"
              className="inline-flex min-h-11 items-center rounded-control bg-brand px-6 font-medium text-white transition duration-150 ease-standard hover:bg-brand-dark"
            >
              {t('problem.feed.report')}
            </Link>
          }
        />
      )}

      {!isLoading && !isError && visible.length > 0 && (
        <div className="flex flex-col gap-5">
          {groups.map((group) => (
            <section key={group.id ?? 'all'} className="flex flex-col gap-3">
              {group.id && (
                <h2 className="flex items-center gap-2 text-micro font-semibold uppercase tracking-wider text-muted after:h-px after:flex-1 after:bg-hairline after:content-['']">
                  {t(`problem.feed.groups.${group.id}`)}
                </h2>
              )}
              <div className="flex flex-col gap-3">
                {group.rows.map((p, i) => (
                  <ProblemCard key={p.id} problem={p} index={i} />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  )
}
