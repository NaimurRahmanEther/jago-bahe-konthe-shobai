import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useAcknowledgeCase,
  useReportObstacle,
  useCase,
  useCompleteTask,
  useMarkDone,
  usePostUpdate,
  useSubmitPlan,
  useRevisePlan,
  useUploadEvidence,
  toPublicStatus,
} from '../../hooks/useCases.js'
import { useProblem } from '../../hooks/useProblems.js'
import StatusBadge from '../../components/problem/StatusBadge.jsx'
import PlanForm from '../../components/resolution/PlanForm.jsx'
import UpdateTimeline from '../../components/resolution/UpdateTimeline.jsx'
import EvidenceUpload from '../../components/resolution/EvidenceUpload.jsx'
import ObstacleForm from '../../components/resolution/ObstacleForm.jsx'
import ObstacleJudgment from '../../components/resolution/ObstacleJudgment.jsx'
import AuditTrail from '../../components/resolution/AuditTrail.jsx'
import Button from '../../components/ui/Button.jsx'
import Section from '../../components/ui/Section.jsx'
import Spinner from '../../components/ui/Spinner.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

const WORKING_STATUSES = ['Planned', 'InProgress', 'Blocked', 'Reopened']

export default function CaseDetail() {
  const { t } = useTranslation()
  const { id } = useParams()
  const { data: kase, isLoading, isError, refetch } = useCase(id)
  const { data: problem } = useProblem(kase?.problemId)

  const acknowledgeCase = useAcknowledgeCase()
  const submitPlan = useSubmitPlan()
  const revisePlan = useRevisePlan()
  const postUpdate = usePostUpdate()
  const uploadEvidence = useUploadEvidence()
  const markDone = useMarkDone()
  const reportObstacle = useReportObstacle()
  const completeTask = useCompleteTask()

  const [disputing, setDisputing] = useState(false)
  const [disputeReason, setDisputeReason] = useState('')
  const [blocking, setBlocking] = useState(false)
  const [revising, setRevising] = useState(false)

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  if (isError || !kase) {
    return <ErrorState message={t('resolution.dashboard.error')} onRetry={refetch} />
  }

  const hasEvidence = kase.evidence.length > 0
  const canMarkDone = kase.status === 'InProgress' && hasEvidence
  // Replan restarts a stalled case with a fresh plan — legal from Reopened and
  // InProgress (e.g. after an obstacle was resolved), never from Blocked, which
  // must be unblocked first. Mirrors the backend lifecycle (A.x).
  const canReplan = kase.status === 'InProgress' || kase.status === 'Reopened'

  return (
    // gap-6 between sections, matching ProblemDetail and AdminHome.
    <div className="flex flex-col gap-6">
      <Link to="/official" className="font-medium text-brand">
        {t('resolution.detail.back')}
      </Link>

      <div className="flex items-start justify-between gap-3 animate-fade-in">
        <h1 className="text-h1 font-semibold text-ink">
          {problem?.title ?? t('resolution.dashboard.caseLabel', { id: kase.id })}
        </h1>
        <StatusBadge status={toPublicStatus(kase.status)} className="shrink-0" />
      </div>
      {problem && <p className="text-sm text-muted">{problem.location.address}</p>}

      {kase.status === 'Assigned' && (
        <div className="flex flex-col gap-3 rounded-card border border-hairline bg-surface p-4">
          <p className="text-sm text-ink">{t('resolution.acknowledge.prompt')}</p>
          {!disputing ? (
            <div className="flex flex-wrap gap-3">
              <Button
                onClick={() => acknowledgeCase.mutate({ caseId: kase.id, decision: 'accept' })}
                disabled={acknowledgeCase.isPending}
              >
                {t('resolution.acknowledge.accept')}
              </Button>
              <Button variant="secondary" onClick={() => setDisputing(true)} disabled={acknowledgeCase.isPending}>
                {t('resolution.acknowledge.dispute')}
              </Button>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              <label htmlFor="dispute-reason" className="font-medium text-ink">
                {t('resolution.acknowledge.disputeReason')}
              </label>
              <textarea
                id="dispute-reason"
                value={disputeReason}
                onChange={(e) => setDisputeReason(e.target.value)}
                rows={2}
                className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
              />
              <div className="flex gap-3">
                <Button
                  onClick={() => acknowledgeCase.mutate({ caseId: kase.id, decision: 'dispute', reason: disputeReason.trim() })}
                  disabled={!disputeReason.trim() || acknowledgeCase.isPending}
                >
                  {t('resolution.acknowledge.submitDispute')}
                </Button>
                <Button variant="secondary" onClick={() => setDisputing(false)} disabled={acknowledgeCase.isPending}>
                  {t('common.cancel')}
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {kase.status === 'Disputed' && (
        <div className="rounded-card border border-hairline bg-surface p-4">
          <p className="text-sm text-ink">{t('resolution.acknowledge.disputed')}</p>
          {kase.disputeReason && <p className="mt-1 text-sm text-muted">{kase.disputeReason}</p>}
        </div>
      )}

      {kase.status === 'Acknowledged' && (
        <PlanForm
          problemId={kase.problemId}
          onSubmit={(payload) => submitPlan.mutateAsync({ caseId: kase.id, ...payload })}
          isSubmitting={submitPlan.isPending}
        />
      )}

      {WORKING_STATUSES.includes(kase.status) && (
        <>
          {kase.plan && (
            <Section title={t('resolution.plan.title')}>
              <p className="text-ink">
                <span className="font-medium">{t('resolution.plan.suggestionResponse')}: </span>
                {kase.plan.suggestionResponse}
              </p>
              <p className="mt-2 text-ink">{kase.plan.strategy}</p>
              {kase.plan.tasks?.length > 0 && (
                <ul className="mt-3 flex flex-col gap-1">
                  {kase.plan.tasks.map((task) => (
                    <li key={task.id}>
                      <label className="flex min-h-11 cursor-pointer items-center gap-3">
                        <input
                          type="checkbox"
                          checked={task.completed}
                          disabled={task.completed || completeTask.isPending}
                          onChange={() =>
                            completeTask.mutate({ caseId: kase.id, taskId: task.id, problemId: kase.problemId })
                          }
                          className="h-5 w-5 shrink-0 accent-brand"
                        />
                        <span className={task.completed ? 'text-muted line-through' : 'text-ink'}>
                          <span className="text-meta text-muted">
                            {t('resolution.plan.week', { count: task.weekNumber })}:{' '}
                          </span>
                          {task.task}
                        </span>
                      </label>
                    </li>
                  ))}
                </ul>
              )}
            </Section>
          )}

          {canReplan &&
            (!revising ? (
              <Button variant="secondary" className="self-start" onClick={() => setRevising(true)}>
                {t('resolution.replan.open')}
              </Button>
            ) : (
              <PlanForm
                problemId={kase.problemId}
                mode="revise"
                initial={kase.plan}
                onSubmit={(payload) =>
                  revisePlan
                    .mutateAsync({ caseId: kase.id, problemId: kase.problemId, ...payload })
                    .then(() => setRevising(false))
                }
                isSubmitting={revisePlan.isPending}
              />
            ))}

          {kase.status === 'Blocked' && kase.obstacles.length > 0 && (
            <ObstacleJudgment problemId={kase.problemId} />
          )}

          <Section title={t('resolution.timeline.title')}>
            <UpdateTimeline
              updates={kase.updates}
              onPost={(text) => postUpdate.mutateAsync({ caseId: kase.id, kind: 'progress', text })}
              isSubmitting={postUpdate.isPending}
            />
          </Section>

          <Section title={t('resolution.evidence.title')}>
            <EvidenceUpload
              onSubmit={(payload) => uploadEvidence.mutateAsync({ caseId: kase.id, ...payload })}
              isSubmitting={uploadEvidence.isPending}
            />
            {hasEvidence && (
              <p className="mt-2 text-meta text-validated-fg">
                {t('resolution.evidence.attached', { count: kase.evidence.length })}
              </p>
            )}
          </Section>

          {kase.status !== 'Blocked' &&
            (!blocking ? (
              <Button variant="secondary" className="self-start" onClick={() => setBlocking(true)}>
                {t('resolution.obstacle.openForm')}
              </Button>
            ) : (
              <ObstacleForm
                onSubmit={(payload) => reportObstacle.mutateAsync({ caseId: kase.id, ...payload }).then(() => setBlocking(false))}
                isSubmitting={reportObstacle.isPending}
              />
            ))}

          {kase.status !== 'Blocked' && (
            <div className="flex flex-col gap-1">
              <Button onClick={() => markDone.mutate({ caseId: kase.id })} disabled={!canMarkDone || markDone.isPending}>
                {t('resolution.markDone')}
              </Button>
              {!canMarkDone && <p className="text-xs text-muted">{t('resolution.markDoneHint')}</p>}
            </div>
          )}
        </>
      )}

      {kase.status === 'Done' && (
        <div className="rounded-card border border-transparent bg-done-bg p-4 text-done-fg">
          <p className="text-sm font-semibold">{t('resolution.done.title')}</p>
          <p className="mt-1 text-sm">{t('resolution.done.awaitingConfirmation')}</p>
        </div>
      )}

      {kase.status === 'Resolved' && (
        <div className="rounded-card border border-transparent bg-resolved-bg p-4 text-resolved-fg">
          <p className="text-sm font-semibold">{t('resolution.resolved.title')}</p>
        </div>
      )}

      <Section title={t('problem.detail.recordTitle')} boxed={false}>
        <AuditTrail entries={problem?.audit ?? []} />
      </Section>
    </div>
  )
}
