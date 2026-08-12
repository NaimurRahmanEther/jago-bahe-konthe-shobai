import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as moderationApi from '../lib/api/moderation.js'
import { problemKeys } from './useProblems.js'

export const moderationKeys = {
  queue: ['admin', 'moderation'],
}

export function useModerationQueue() {
  return useQuery({ queryKey: moderationKeys.queue, queryFn: moderationApi.listModeration })
}

// Both mutations invalidate the problem keys as well as the screening queue: a
// screening decision moves the report into the feed (approve) or publicly marks
// it Rejected, so every problem view is stale afterwards, not just this list.
export function useApproveProblem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (problemId) => moderationApi.approveProblem(problemId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: moderationKeys.queue })
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}

export function useRejectProblem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ problemId, ...payload }) => moderationApi.rejectProblem(problemId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: moderationKeys.queue })
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}
