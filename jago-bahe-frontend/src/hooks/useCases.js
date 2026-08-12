import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as resolutionApi from '../lib/api/resolution.js'
import { useAuth } from '../auth/useAuth.js'

export const caseKeys = {
  all: ['cases'],
  list: ['cases', 'list'],
  detail: (id) => ['cases', 'detail', id],
}

/** Cases assigned to the signed-in official. */
export function useCases() {
  const { user } = useAuth()
  return useQuery({
    queryKey: caseKeys.list,
    // The backend infers the official from the JWT.
    queryFn: () => resolutionApi.listCases(),
    // Not a mock-parameter guard: an official has no officialId until their
    // claim is approved (B11 sets accounts.official_id on approval), and asking
    // for cases before then would answer an empty list that reads as "you have
    // no work" rather than "your claim is pending".
    enabled: Boolean(user?.officialId),
  })
}

/**
 * A problem's public progress. Fetched lazily — pass `enabled` so a list of rows
 * does not fire one query per row on mount.
 *
 * Resolves `null` (not an error) when there is no case behind the problem.
 * @param {string} problemId
 * @param {{enabled?: boolean}} [options]
 * @returns {import('@tanstack/react-query').UseQueryResult<import('../lib/types/models.js').Progress|null>}
 */
export function useProgress(problemId, { enabled = true } = {}) {
  return useQuery({
    queryKey: ['progress', problemId],
    queryFn: () => resolutionApi.getProgress(problemId),
    enabled: Boolean(problemId) && enabled,
  })
}

/** @param {string} id */
export function useCase(id) {
  return useQuery({
    queryKey: caseKeys.detail(id),
    queryFn: () => resolutionApi.getCase(id),
    enabled: Boolean(id),
  })
}

function useCaseMutation(mutationFn) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn,
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: caseKeys.detail(variables.caseId) })
      queryClient.invalidateQueries({ queryKey: caseKeys.list })
    },
  })
}

export function useAcknowledgeCase() {
  return useCaseMutation(({ caseId, ...payload }) => resolutionApi.acknowledgeCase(caseId, payload))
}

export function useSubmitPlan() {
  return useCaseMutation(({ caseId, ...payload }) => resolutionApi.submitPlan(caseId, payload))
}

export function usePostUpdate() {
  return useCaseMutation(({ caseId, ...payload }) => resolutionApi.postUpdate(caseId, payload))
}

export function useUploadEvidence() {
  return useCaseMutation(({ caseId, ...payload }) => resolutionApi.uploadEvidence(caseId, payload))
}

export function useMarkDone() {
  return useCaseMutation(({ caseId }) => resolutionApi.markDone(caseId))
}

export function useReportObstacle() {
  return useCaseMutation(({ caseId, ...payload }) => resolutionApi.reportObstacle(caseId, payload))
}

/**
 * Posts a replacement plan to restart a stalled case. Beyond the case detail +
 * list every case mutation refreshes, it also invalidates the public
 * `['progress', problemId]` key so the public/reporter panel shows the new plan —
 * pass `problemId` in the variables (the `reason` and plan fields go to the API).
 */
export function useRevisePlan() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ caseId, problemId: _problemId, ...payload }) => resolutionApi.revisePlan(caseId, payload),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: caseKeys.detail(variables.caseId) })
      queryClient.invalidateQueries({ queryKey: caseKeys.list })
      if (variables.problemId) {
        queryClient.invalidateQueries({ queryKey: ['progress', variables.problemId] })
      }
    },
  })
}

/**
 * Marks one weekly plan task complete. Beyond the case detail + list that every
 * case mutation refreshes, it also invalidates the public `['progress', problemId]`
 * key so the public panel reflects the check — pass `problemId` in the variables.
 */
export function useCompleteTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ caseId, taskId }) => resolutionApi.completeTask(caseId, taskId),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: caseKeys.detail(variables.caseId) })
      queryClient.invalidateQueries({ queryKey: caseKeys.list })
      if (variables.problemId) {
        queryClient.invalidateQueries({ queryKey: ['progress', variables.problemId] })
      }
    },
  })
}

/** Collapses the finer-grained case lifecycle to the public 8-state StatusBadge vocabulary. */
export function toPublicStatus(status) {
  if (status === 'Acknowledged' || status === 'Planned' || status === 'Assigned') return 'Assigned'
  if (status === 'Disputed') return 'Reopened'
  return status
}
