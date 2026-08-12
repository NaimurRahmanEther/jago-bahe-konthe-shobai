import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import Spinner from '../ui/Spinner.jsx'
import UpvoteButton from '../suggestion/UpvoteButton.jsx'
import AdjudicateObstacle from './AdjudicateObstacle.jsx'
import { useAuth } from '../../auth/useAuth.js'
import {
  useObstacle,
  useVoteObstacle,
  useProposeUnblockingPlan,
  useUpvoteUnblockingPlan,
  useAdjudicateObstacle,
} from '../../hooks/useObstacle.js'

/**
 * The public judgment surface for a blocked problem's obstacle (Concept §8):
 * the named obstacle, an advisory "is this real?" vote, and community
 * unblocking plans. The vote is pressure only — it never sets status; the named
 * authority/moderator adjudicates the scorecard consequence.
 * @param {{problemId: string}} props
 */
export default function ObstacleJudgment({ problemId }) {
  const { t } = useTranslation()
  const { role } = useAuth()
  const { data: obstacle, isLoading } = useObstacle(problemId)
  const voteObstacle = useVoteObstacle(problemId)
  const proposePlan = useProposeUnblockingPlan(problemId)
  const upvotePlan = useUpvoteUnblockingPlan(problemId)
  const adjudicate = useAdjudicateObstacle(problemId)

  const [votedChoice, setVotedChoice] = useState(null)
  const [planText, setPlanText] = useState('')
  const [upvotedIds, setUpvotedIds] = useState(() => new Set())

  if (isLoading) {
    return (
      <div className="flex justify-center rounded-card border border-hairline bg-surface p-4">
        <Spinner />
      </div>
    )
  }
  if (!obstacle) return null

  const isResident = role === 'resident'
  const isAdmin = role === 'admin'
  const plans = obstacle.unblockingPlans ?? []

  function castVote(vote) {
    if (votedChoice || voteObstacle.isPending) return
    setVotedChoice(vote)
    voteObstacle.mutate(vote)
  }

  function toggleUpvote(id) {
    setUpvotedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
    upvotePlan.mutate(id)
  }

  function submitPlan(e) {
    e.preventDefault()
    if (!planText.trim()) return
    proposePlan.mutate(planText.trim())
    setPlanText('')
  }

  return (
    <div className="flex flex-col gap-4 rounded-card border border-hairline bg-surface p-4">
      <p className="text-sm font-semibold text-ink">{t('obstacle.title')}</p>

      {/* The obstacle itself — amber: waiting, not failure (Design Guideline §2). */}
      <div className="rounded-card border border-transparent bg-blocked-bg p-3 text-blocked-fg">
        <span className="inline-flex rounded-full bg-surface px-2 py-0.5 text-[11px] font-bold text-blocked-fg">
          {t(`obstacle.categories.${obstacle.category}`)}
        </span>
        <p className="mt-2 text-sm">{obstacle.whatBlocks}</p>
        <p className="mt-1 text-xs">
          {t('resolution.obstacle.whoUnblocksLabel')}: {obstacle.whoUnblocks}
        </p>
        <p className="mt-1 text-xs">
          {t('resolution.obstacle.proofTried')}: {obstacle.proofTried}
        </p>
      </div>

      {/* Advisory "is this obstacle real?" vote. */}
      <div>
        <p className="text-sm font-medium text-ink">{t('obstacle.vote.prompt')}</p>
        {isResident && (
          <div className="mt-3 flex gap-3">
            <Button
              variant={votedChoice === 'real' ? 'primary' : 'secondary'}
              className="min-h-13 flex-1 text-base"
              onClick={() => castVote('real')}
              disabled={Boolean(votedChoice) || voteObstacle.isPending}
            >
              {t('obstacle.vote.real')}
            </Button>
            <Button
              variant={votedChoice === 'not_convinced' ? 'primary' : 'secondary'}
              className="min-h-13 flex-1 text-base"
              onClick={() => castVote('not_convinced')}
              disabled={Boolean(votedChoice) || voteObstacle.isPending}
            >
              {t('obstacle.vote.notConvinced')}
            </Button>
          </div>
        )}
        <p className="mt-2 text-sm text-muted">
          {t('obstacle.vote.tally', { real: obstacle.realCount, notConvinced: obstacle.notConvincedCount })}
        </p>
        <p className="mt-1 text-xs text-muted">{t('obstacle.vote.advisoryNote')}</p>
        {votedChoice && <p className="mt-1 text-sm text-blocked-fg">{t('obstacle.vote.thanks')}</p>}
      </div>

      {/* Community ways to unblock it — the suggestion engine, re-pointed. */}
      <div className="flex flex-col gap-3">
        <p className="text-sm font-medium text-ink">{t('obstacle.plans.title')}</p>
        {plans.length === 0 ? (
          <p className="text-sm text-muted">{t('obstacle.plans.empty')}</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {plans.map((p) => (
              <li
                key={p.id}
                className="flex items-start justify-between gap-3 rounded-card border border-hairline bg-surface p-3"
              >
                <p className="text-sm text-ink">{p.text}</p>
                <UpvoteButton
                  count={p.upvoteCount}
                  active={upvotedIds.has(p.id)}
                  onToggle={() => toggleUpvote(p.id)}
                  disabled={upvotePlan.isPending}
                />
              </li>
            ))}
          </ul>
        )}

        {isResident ? (
          <form onSubmit={submitPlan} className="flex flex-col gap-2">
            <label htmlFor="unblocking-plan" className="font-medium text-ink">
              {t('obstacle.plans.formLabel')}
            </label>
            <textarea
              id="unblocking-plan"
              value={planText}
              onChange={(e) => setPlanText(e.target.value)}
              rows={2}
              className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
            />
            <Button type="submit" disabled={proposePlan.isPending} className="self-start">
              {t('obstacle.plans.submit')}
            </Button>
          </form>
        ) : (
          <p className="text-sm text-muted">{t('obstacle.plans.residentOnly')}</p>
        )}
      </div>

      {/* Admin/moderator authority: the binding verdict that sets the scorecard
          consequence (after reading the public's advisory tally above). */}
      {isAdmin && (
        <AdjudicateObstacle
          obstacle={obstacle}
          onAdjudicate={(decision) => adjudicate.mutate({ obstacleId: obstacle.id, decision })}
          isSubmitting={adjudicate.isPending}
          isError={adjudicate.isError}
        />
      )}
    </div>
  )
}
