import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as obstacleApi from '../lib/api/obstacle.js'
import { problemKeys } from './useProblems.js'

export const obstacleKeys = {
  all: ['obstacle'],
  detail: (problemId) => [...obstacleKeys.all, problemId],
}

/** The active obstacle a blocked problem is being judged on, or null. */
export function useObstacle(problemId) {
  return useQuery({
    queryKey: obstacleKeys.detail(problemId),
    queryFn: () => obstacleApi.getObstacle(problemId),
    enabled: Boolean(problemId),
  })
}

function useObstacleMutation(problemId, mutationFn) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: obstacleKeys.detail(problemId) })
    },
  })
}

/** @param {string} problemId */
export function useVoteObstacle(problemId) {
  return useObstacleMutation(problemId, (vote) => obstacleApi.voteObstacle(problemId, { vote }))
}

/** @param {string} problemId */
export function useProposeUnblockingPlan(problemId) {
  return useObstacleMutation(problemId, (text) => obstacleApi.proposeUnblockingPlan(problemId, text))
}

/** @param {string} problemId */
export function useUpvoteUnblockingPlan(problemId) {
  return useObstacleMutation(problemId, (planId) => obstacleApi.upvoteUnblockingPlan(problemId, planId))
}

/**
 * Adjudicate the problem's active obstacle (admin/moderator). A deny bounces the
 * case back to InProgress, so this also refreshes the problem detail + feed —
 * the obstacle surface disappears once the problem leaves Blocked.
 * @param {string} problemId
 */
export function useAdjudicateObstacle(problemId) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ obstacleId, decision }) => obstacleApi.adjudicateObstacle(obstacleId, decision),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: obstacleKeys.detail(problemId) })
      queryClient.invalidateQueries({ queryKey: problemKeys.detail(problemId) })
      queryClient.invalidateQueries({ queryKey: problemKeys.list() })
    },
  })
}
