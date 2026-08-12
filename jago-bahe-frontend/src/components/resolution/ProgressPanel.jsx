import { useTranslation } from 'react-i18next'
import Spinner from '../ui/Spinner.jsx'
import ErrorState from '../ui/ErrorState.jsx'
import StatusBadge from '../problem/StatusBadge.jsx'
import { formatDate } from '../../lib/date.js'

/**
 * The public progress of one problem: what the official said they would do, the
 * community question that plan answers, the updates since, and the photographic
 * evidence.
 *
 * The answeredSuggestion is the snapshot the API returns — the top suggestion as
 * it stood when the plan was written. It is never re-ranked or recomputed here:
 * "top" moves with live upvotes, and pairing the response with today's top would
 * show the official answering a question they were never asked.
 *
 * Rendered in two places, deliberately differently framed: expanded on the public
 * problem page, where it is the point of the page, and behind a toggle on /me,
 * where it is one row among the reporter's own reports.
 *
 * @param {{
 *   progress: object|null|undefined,
 *   isLoading?: boolean,
 *   isError?: boolean,
 *   onRetry?: () => void,
 *   onZoom?: (src: string, alt: string) => void,
 * }} props
 */
export default function ProgressPanel({ progress, isLoading, isError, onRetry, onZoom }) {
  const { t } = useTranslation()

  // The panel's dates are the ones the reader dwells on, so they keep the long
  // month form the local formatter used before lib/date.js absorbed it.
  const long = (iso) => formatDate(iso, 'long')

  if (isLoading) {
    return (
      <div className="flex justify-center py-6">
        <Spinner />
      </div>
    )
  }

  if (isError) {
    return <ErrorState message={t('progress.error')} onRetry={onRetry} />
  }

  // No case at all — assigned, but the official has not opened it yet. Calm and
  // factual, not an error: nothing has gone wrong, nothing has happened.
  if (!progress) {
    return <p className="py-3 text-muted">{t('progress.none')}</p>
  }

  const { plan, updates = [], evidence = [], obstacles = [], blockedOnHigherAuthority } = progress

  return (
    <div className="mt-3 flex flex-col gap-5">
      {blockedOnHigherAuthority && (
        <p className="rounded-control bg-blocked-bg p-3 text-blocked-fg">{t('progress.fairnessNote')}</p>
      )}

      {/* Where the case stands, before any of its content. A reader's first
          question is whether the official has even opened it — and silence is
          itself a fact worth publishing, which is why an unacknowledged case says
          so rather than showing nothing. */}
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-hairline pb-4">
        {progress.status && <StatusBadge status={progress.status} />}
        <p className="text-meta text-muted">
          {progress.acknowledged
            ? t('progress.acknowledgedOn', { date: long(progress.acknowledgedAt ?? progress.createdAt) })
            : t('progress.notAcknowledged')}
        </p>
        {progress.deadline && (
          <p className="text-meta text-muted">
            {t('progress.deadline', { date: long(progress.deadline) })}
          </p>
        )}
      </div>

      {/* The plan, when there is one. A case can legitimately exist without one —
          acknowledged but not yet planned — and this used to bail out of the whole
          component in that state, hiding the updates and evidence that DID exist
          behind a flat "no plan published". */}
      {plan ? (
        <div className="flex flex-col gap-4">
          <div>
            <h3 className="text-title font-semibold text-ink">{t('progress.planTitle')}</h3>
            <p className="mt-1.5 max-w-[70ch] text-ink">{plan.strategy}</p>
          </div>

          {/* The week-by-week checklist, with each week's completion state — the
              public sees the plan being worked through, not just its length. Never
              colour alone (A.6): the tick, the strikethrough, and a screen-reader
              label all carry "done" alongside the green. Legacy plans that predate
              the checklist have no tasks and fall back to the bare week count. */}
          {plan.tasks?.length > 0 ? (
            <div>
              <p className="text-meta font-medium text-muted">{t('progress.weeklyTasks')}</p>
              <ul className="mt-2 flex flex-col gap-1.5">
                {plan.tasks.map((task) => (
                  <li key={task.id} className="flex items-start gap-2">
                    <span aria-hidden="true" className={task.completed ? 'text-validated-fg' : 'text-muted'}>
                      {task.completed ? '✓' : '○'}
                    </span>
                    <span className={task.completed ? 'text-muted line-through' : 'text-ink'}>
                      <span className="text-meta text-muted">
                        {t('resolution.plan.week', { count: task.weekNumber })}:{' '}
                      </span>
                      {task.task}
                    </span>
                    <span className="sr-only">
                      {task.completed ? t('progress.taskDone') : t('progress.taskPending')}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            plan.timelineWeeks > 0 && (
              <div className="rounded-control border border-hairline p-3 sm:max-w-xs">
                <p className="text-meta font-medium text-muted">{t('progress.timeline')}</p>
                <p className="mt-0.5 text-ink">{t('progress.timelineWeeks', { count: plan.timelineWeeks })}</p>
              </div>
            )
          )}

          {plan.obstacles && (
            <div className="rounded-control border border-hairline p-3 sm:max-w-xs">
              <p className="text-meta font-medium text-muted">{t('progress.obstacles')}</p>
              <p className="mt-0.5 text-ink">{plan.obstacles}</p>
            </div>
          )}

          {/* THE accountability pairing (Concept §4, Figure 3): the community asked
              X, the official answered Y. Stacked rather than side by side, and the
              answer indented under the question, because these are not two equal
              facts — the second is a reply to the first, and a two-column grid drew
              them as unrelated neighbours. A problem nobody upvoted has no top, so
              the answer then stands on its own. */}
          {(plan.answeredSuggestion || plan.suggestionResponse) && (
            <div className="flex flex-col gap-2 rounded-card border border-brand/20 bg-brand-tint p-4">
              {plan.answeredSuggestion && (
                <div>
                  <p className="text-meta font-medium text-brand-dark">
                    {t('progress.answeredSuggestion')}
                  </p>
                  <p className="mt-1 max-w-[70ch] text-ink">{plan.answeredSuggestion}</p>
                </div>
              )}
              {plan.suggestionResponse && (
                <div className="border-l-2 border-brand pl-3 sm:ml-2">
                  <p className="text-meta font-medium text-brand-dark">{t('progress.response')}</p>
                  <p className="mt-1 max-w-[70ch] text-ink">{plan.suggestionResponse}</p>
                </div>
              )}
            </div>
          )}
        </div>
      ) : (
        <p className="text-muted">{t('progress.noPlanYet')}</p>
      )}

      {/* What got in the way — the public "cause of not solving". Persistent: it
          stays on the record after the case leaves Blocked (obstacle denied, or a
          resident reopened it), so a reader can always see why the work stalled.
          Amber, never shaming: an obstacle is waiting, not failure (A.6). */}
      {obstacles.length > 0 && (
        <div>
          <h3 className="text-title font-semibold text-ink">{t('progress.obstaclesTitle')}</h3>
          <div className="mt-2 flex flex-col gap-3">
            {obstacles.map((o) => (
              <div key={o.id} className="rounded-card border border-transparent bg-blocked-bg p-3 text-blocked-fg">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="inline-flex rounded-full bg-surface px-2 py-0.5 text-micro font-bold text-blocked-fg">
                    {t(`obstacle.categories.${o.category}`)}
                  </span>
                  <span className="text-micro font-medium">{t(`progress.obstacleVerdict.${o.adjudication}`)}</span>
                </div>
                <p className="mt-2 max-w-[70ch] text-sm">{o.whatBlocks}</p>
                <p className="mt-1 text-meta">
                  {t('resolution.obstacle.whoUnblocksLabel')}: {o.whoUnblocks}
                </p>
                {o.proofTried && (
                  <p className="mt-1 text-meta">
                    {t('resolution.obstacle.proofTried')}: {o.proofTried}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {updates.length > 0 && (
        <div>
          <h3 className="text-title font-semibold text-ink">{t('progress.updatesTitle')}</h3>
          <ol className="mt-2 flex flex-col gap-3 border-l-2 border-hairline pl-4">
            {updates.map((u) => (
              <li key={u.id}>
                <p className="text-meta tabular-nums text-muted">{long(u.createdAt)}</p>
                <p className="mt-0.5 max-w-[70ch] text-ink">{u.text}</p>
              </li>
            ))}
          </ol>
        </div>
      )}

      {/* The before/after record. This has always been on the progress payload and
          was never rendered — the problem page draws its own gallery from
          problem.evidence, which the API still ships empty (the standing B5
          TODO(contract)), so until now the photographs an official uploaded were
          reachable nowhere at all. */}
      {evidence.length > 0 && (
        <div>
          <h3 className="text-title font-semibold text-ink">{t('progress.evidenceTitle')}</h3>
          <div className="mt-2 flex flex-col gap-4">
            {evidence.map((e) => (
              <div key={e.id} className="grid gap-3 sm:grid-cols-2">
                {[
                  { src: e.beforeImageUrl, label: t('resolution.evidence.before') },
                  { src: e.afterImageUrl, label: t('resolution.evidence.after') },
                ]
                  .filter((photo) => photo.src)
                  .map((photo) => (
                    <figure key={photo.label} className="flex flex-col gap-1">
                      {onZoom ? (
                        <button
                          type="button"
                          onClick={() => onZoom(photo.src, photo.label)}
                          className="block w-full cursor-zoom-in rounded-control"
                        >
                          <EvidenceImage src={photo.src} alt={photo.label} />
                        </button>
                      ) : (
                        <EvidenceImage src={photo.src} alt={photo.label} />
                      )}
                      <figcaption className="text-micro text-muted">{photo.label}</figcaption>
                    </figure>
                  ))}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

/** A fixed aspect ratio so the page does not jump as the photo decodes (A.6). */
function EvidenceImage({ src, alt }) {
  return (
    <img
      src={src}
      alt={alt}
      loading="lazy"
      className="aspect-[4/3] w-full rounded-control border border-hairline object-cover"
    />
  )
}
