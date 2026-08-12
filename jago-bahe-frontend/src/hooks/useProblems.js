import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as problemsApi from '../lib/api/problems.js'
import * as resolutionApi from '../lib/api/resolution.js'
import { useAuth } from '../auth/useAuth.js'

export const problemKeys = {
  all: ['problems'],
  list: (filters) => [...problemKeys.all, 'list', filters ?? {}],
  // Keyed by reporter even though the endpoint takes no parameter: without it,
  // one account's reports would stay cached across a logout and reappear for the
  // next person to sign in on the same tab. This is cache isolation, not a
  // leftover of the deleted mock layer — see CLAUDE.md A.5.5.
  mine: (reporterId) => [...problemKeys.all, 'mine', reporterId ?? ''],
  detail: (id) => [...problemKeys.all, 'detail', id],
}

/** @param {{area?: string, status?: string, official?: string}} [filters] */
export function useProblems(filters) {
  return useQuery({
    queryKey: problemKeys.list(filters),
    queryFn: () => problemsApi.listProblems(filters),
  })
}

/** The signed-in resident's own reports, at every status. */
export function useMyProblems() {
  const { user } = useAuth()
  return useQuery({
    queryKey: problemKeys.mine(user?.id),
    queryFn: () => problemsApi.listMyProblems(),
    enabled: Boolean(user?.id),
  })
}

/** @param {string} id */
export function useProblem(id) {
  const { user } = useAuth()
  return useQuery({
    // Keyed by caller: a pending problem is readable by its reporter and nobody
    // else, so the same id legitimately has two answers (the problem, or a 404).
    // Sharing one cache entry between them would serve one reader the other's.
    // The request itself carries no caller — the JWT does.
    queryKey: [...problemKeys.detail(id), user?.id ?? ''],
    queryFn: () => problemsApi.getProblem(id),
    enabled: Boolean(id),
  })
}

export function useReportProblem() {
  const queryClient = useQueryClient()
  return useMutation({
    // The backend takes the reporter from the JWT.
    mutationFn: (payload) => problemsApi.reportProblem(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}

export function useValidateProblem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, vote }) => problemsApi.validateProblem(id, vote),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: problemKeys.detail(variables.id) })
      // problemKeys.all, not .list(): a bare .list() is the key ['problems','list',{}]
      // and matched other lists only by prefix accident — it would never match the
      // reporter's own view, leaving /me stale after a validate or a confirm.
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}

/** Edit the reporter's own report. @param {string} id */
export function useUpdateProblem(id) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload) => problemsApi.updateProblem(id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: problemKeys.detail(id) })
      // .all, not .list(): a bare .list() never matches the reporter's own /me view.
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}

/** Withdraw (soft-delete) the reporter's own report. @param {string} id */
export function useWithdrawProblem(id) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (note) => problemsApi.withdrawProblem(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: problemKeys.detail(id) })
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}

/**
 * Permanently delete the reporter's own report. Erases it and everything attached;
 * see lib/api/problems.js deleteProblem for what that includes.
 *
 * @param {string} id
 */
export function useDeleteProblem(id) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => problemsApi.deleteProblem(id),
    onSuccess: () => {
      // removeQueries, not invalidateQueries — the difference matters here and
      // nowhere else on this page. Invalidating would refetch a URL that now 404s
      // and drop the caller into an error state on a report they meant to delete;
      // there is no fresh copy to go and get. Drop the entry instead.
      //
      // useProblem keys detail as [...detail(id), userId], so removing on the
      // detail(id) prefix clears every caller's copy, which is what we want.
      queryClient.removeQueries({ queryKey: problemKeys.detail(id) })
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}

/** @param {string} problemId */
export function useConfirmProblem(problemId) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (outcome) => resolutionApi.confirmProblem(problemId, { outcome }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: problemKeys.detail(problemId) })
      // problemKeys.all, not .list(): a bare .list() is the key ['problems','list',{}]
      // and matched other lists only by prefix accident — it would never match the
      // reporter's own view, leaving /me stale after a validate or a confirm.
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}
