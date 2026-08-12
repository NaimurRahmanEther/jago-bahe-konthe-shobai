import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useModerationQueue } from '../../hooks/useModeration.js'
import { useAdminQueue, useValidatingQueue, useMyForwarding } from '../../hooks/useAssignments.js'
import { useSeatOverview } from '../../hooks/useScorecard.js'
import StatusBadge from '../../components/problem/StatusBadge.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import StatTile from '../../components/ui/StatTile.jsx'
import ValidationBar from '../../components/ui/ValidationBar.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

// The admin's three queues follow the report's life in order: screening decides
// whether a report is publishable at all, validation is the community weighing in,
// and assignment decides who fixes it. They are different decisions with different
// rules, so they get different lists.
//
// The middle one is new in B17. It exists because the admin used to be blind
// between approving a report and its reaching V — the assignment queue was
// Validated-only, so a report was absent and then abruptly present, with nowhere
// to watch it climb. The count gates nothing (CLAUDE.md A.3.1); the admin forwards
// a report when they judge it trustworthy, and these rows are that judgment's
// evidence.
function QueueSection({ id, title, linkLabel, href, query, emptyLabel, errorLabel, renderRow }) {
  const { data, isLoading, isError, refetch } = query
  return (
    // Named region: all three headings rhyme with the status badges on their own
    // rows in Bangla ("অনুমোদনের অপেক্ষায়" is both this section's title and its
    // rows' badge), so each section needs an accessible name to be addressable at
    // all — for a screen reader and for a test alike (CLAUDE.md A.5.3 rule 4).
    <section className="flex flex-col gap-3" aria-labelledby={id}>
      <div className="flex items-baseline justify-between gap-3">
        <h2 id={id} className="font-semibold text-ink">
          {title}
          {data?.length > 0 && <span className="ml-2 text-meta font-normal text-muted">({data.length})</span>}
        </h2>
        <Link to={href} className="text-meta font-medium text-brand">
          {linkLabel}
        </Link>
      </div>

      {isLoading && <SkeletonList count={2} />}
      {isError && <ErrorState message={errorLabel} onRetry={refetch} />}
      {!isLoading && !isError && data?.length === 0 && <EmptyState title={emptyLabel} />}
      {!isLoading && !isError && data?.length > 0 && (
        <div className="flex flex-col gap-3">{data.slice(0, 3).map(renderRow)}</div>
      )}
    </section>
  )
}

/**
 * One preview row. It takes plain fields rather than an object because the three
 * sections feed it two different API shapes — a problemDTO from moderation
 * (`id`, `location.address`) and a queueItemDTO from the queue (`problemId`,
 * `address`). It used to read the problem shape for both, which meant every queue
 * row threw on `location.address`; keeping the mapping at each call site makes the
 * difference visible instead of hiding it behind one guess.
 *
 * The validation bar renders only when a threshold is present, so the screening
 * section stays a plain row.
 */
function Row({ title, address, status, href, index, validCount, threshold }) {
  return (
    <Link
      to={href}
      style={{ animationDelay: `${Math.min(index, 5) * 40}ms` }}
      className="block rounded-card border border-hairline bg-surface p-4 transition duration-150 ease-standard hover:border-brand animate-rise-in"
    >
      <div className="flex items-start justify-between gap-3">
        <h3 className="text-title font-semibold leading-snug text-ink">{title}</h3>
        <StatusBadge status={status} className="shrink-0" />
      </div>
      <p className="mt-2 text-meta text-muted">{address}</p>
      {threshold > 0 && (
        <ValidationBar count={validCount} threshold={threshold} className="mt-3" />
      )}
    </Link>
  )
}

export default function AdminHome() {
  const { t } = useTranslation()
  const moderation = useModerationQueue()
  const validating = useValidatingQueue()
  const assignment = useAdminQueue()
  const forwarding = useMyForwarding()
  const { data: seat, isLoading: seatLoading, isError: seatError, refetch: refetchSeat } = useSeatOverview()

  // Both queue sections read the same fetch (one key, two selects). Every row is
  // union-routed — since B20 an above-union report is the super admin's to forward
  // and never reaches this queue, so the old `routing === 'vote'` branch to a vote
  // screen is gone with the vote (A.3.8). Those reports get their own section below.
  const queueRow = (item, i) => (
    <Row
      key={item.problemId}
      title={item.title}
      address={item.address}
      status={item.status}
      href={`/admin/problems/${item.problemId}/assign`}
      index={i}
      validCount={item.validCount}
      threshold={item.validationThreshold}
    />
  )

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-h1 font-semibold text-ink">{t('admin.home.title')}</h1>

      <section className="flex flex-col gap-3" aria-labelledby="seat-overview-title">
        <h2 id="seat-overview-title" className="font-semibold text-ink">
          {t('admin.home.statsTitle')}
        </h2>
        {seatLoading && <SkeletonList count={1} />}
        {seatError && <ErrorState message={t('admin.home.statsError')} onRetry={refetchSeat} />}
        {!seatLoading && !seatError && seat && (
          <div className="grid grid-cols-2 gap-3">
            <StatTile value={seat.problems} label={t('admin.home.stats.problems')} />
            <StatTile value={seat.resolved} label={t('admin.home.stats.resolved')} tone="resolved" />
            <StatTile value={seat.pending} label={t('admin.home.stats.pending')} />
            <StatTile value={seat.blocked} label={t('admin.home.stats.blocked')} tone="blocked" />
          </div>
        )}
      </section>

      <QueueSection
        id="admin-screening-title"
        title={t('admin.home.screeningTitle')}
        linkLabel={t('admin.home.screeningLink')}
        href="/admin/moderation"
        query={moderation}
        emptyLabel={t('moderation.empty')}
        errorLabel={t('moderation.error')}
        renderRow={(p, i) => (
          <Row
            key={p.id}
            title={p.title}
            address={p.location.address}
            status={p.status}
            href="/admin/moderation"
            index={i}
          />
        )}
      />

      <QueueSection
        id="admin-validating-title"
        title={t('admin.home.validatingTitle')}
        linkLabel={t('admin.home.validatingLink')}
        href="/admin/validating"
        query={validating}
        emptyLabel={t('validating.empty')}
        errorLabel={t('validating.error')}
        renderRow={queueRow}
      />

      <QueueSection
        id="admin-assignment-title"
        title={t('admin.home.assignmentTitle')}
        linkLabel={t('admin.home.assignmentLink')}
        href="/admin/queue"
        query={assignment}
        emptyLabel={t('assignment.queue.empty')}
        errorLabel={t('assignment.queue.error')}
        renderRow={queueRow}
      />

      {/* The fourth section is a different KIND of thing from the three above it:
          those are decisions this admin makes, this one is advice they give. An
          above-union report is the super admin's to forward, and the admin's part
          is to say where it should go (A.3.8). It sits last because it is the only
          one that does not end with this admin deciding anything. */}
      <QueueSection
        id="admin-forwarding-title"
        title={t('admin.home.forwardingTitle')}
        linkLabel={t('admin.home.forwardingLink')}
        href="/admin/forwarding"
        query={forwarding}
        emptyLabel={t('forwarding.admin.empty')}
        errorLabel={t('forwarding.admin.error')}
        renderRow={(item, i) => (
          <Row
            key={item.problemId}
            title={item.title}
            address={item.address}
            status={item.status}
            href={`/admin/forwarding/${item.problemId}`}
            index={i}
            validCount={item.validCount}
            threshold={item.validationThreshold}
          />
        )}
      />
    </div>
  )
}
