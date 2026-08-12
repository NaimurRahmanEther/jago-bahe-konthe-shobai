import { useCallback } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as assignmentsApi from '../lib/api/assignments.js'
import { problemKeys } from './useProblems.js'

export const assignmentKeys = {
  queue: ['admin', 'queue'],
}

/**
 * The two above-union surfaces are keyed separately because they are genuinely two
 * different sets: the admin's list is scoped by ADVICE SCOPE (their upazila, or the
 * seat) and carries their own `mySuggestion`, while the super admin's is seat-wide
 * and never does. One key shared between them would serve one role's rows to the
 * other on a cache hit.
 */
export const forwardingKeys = {
  mine: ['admin', 'forwarding'],
  queue: ['super', 'queue'],
}

/**
 * GET /api/admin/queue returns everything the acting admin may still forward to an
 * official — Reported as well as Validated. The two hooks below are two views of
 * that ONE response, split with `select` under one query key, so the admin's
 * dashboard renders both sections from a single fetch and a single cache entry
 * (the `useAreas` pattern). Splitting them into two endpoints or two keys would
 * refetch the same rows twice and let the two lists disagree mid-flight.
 */
function useQueue(select) {
  return useQuery({ queryKey: assignmentKeys.queue, queryFn: assignmentsApi.listQueue, select })
}

/** Problems the community has already endorsed (reached V). @returns {import('@tanstack/react-query').UseQueryResult<import('../lib/types/models.js').QueueItem[]>} */
export function useAdminQueue() {
  return useQueue(selectValidated)
}

/**
 * Problems still collecting validations — approved, public, not yet at V. This is
 * the window the admin used to be blind in: before B17 the queue held Validated
 * problems only, so a report was absent and then abruptly present, and there was
 * nowhere to watch it climb.
 *
 * Sorted closest-to-threshold first so the rows nearest to community endorsement
 * are the ones the admin sees. Sorting a list the server returned is presentation,
 * not authority — nothing here decides anything, and the count still gates nothing
 * (A.3.1: the admin may forward any of these at any count, including zero).
 */
export function useValidatingQueue() {
  return useQueue(selectValidating)
}

/**
 * One row of the queue by problem id, for the assign and vote panels.
 *
 * It reads the WHOLE queue response, not `useAdminQueue`'s Validated-only view:
 * since B17 a Reported problem is assignable, so the panel must find it there too
 * — filtering to Validated first would make the assign screen 404 on exactly the
 * early-forwarding case the phase exists to enable.
 *
 * @returns {import('@tanstack/react-query').UseQueryResult<import('../lib/types/models.js').QueueItem | null>}
 */
export function useQueueItem(problemId) {
  return useQueue(
    useCallback((rows) => rows.find((r) => r.problemId === problemId) ?? null, [problemId]),
  )
}

const selectValidated = (rows) => rows.filter((r) => r.status === 'Validated')

const selectValidating = (rows) =>
  rows
    .filter((r) => r.status === 'Reported')
    // Guard a zero threshold: V is config-driven, so never divide by it blind.
    .sort((a, b) => remainingToThreshold(a) - remainingToThreshold(b))

const remainingToThreshold = (row) =>
  row.validationThreshold > 0 ? row.validationThreshold - row.validCount : Number.MAX_SAFE_INTEGER

export function useAssignWithinUnion() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ problemId, ...payload }) => assignmentsApi.assignWithinUnion(problemId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: assignmentKeys.queue })
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}

/**
 * The above-union reports this admin may advise on.
 * @returns {import('@tanstack/react-query').UseQueryResult<import('../lib/types/models.js').ForwardingItem[]>}
 */
export function useMyForwarding() {
  return useQuery({ queryKey: forwardingKeys.mine, queryFn: assignmentsApi.listMyForwarding })
}

/** One advisory row by problem id, for the advise panel. */
export function useMyForwardingItem(problemId) {
  return useQuery({
    queryKey: forwardingKeys.mine,
    queryFn: assignmentsApi.listMyForwarding,
    select: useCallback((rows) => rows.find((r) => r.problemId === problemId) ?? null, [problemId]),
  })
}

/**
 * Record or replace this admin's advice. It invalidates BOTH forwarding keys: the
 * tally the admin just moved is the same tally the super admin is deciding on, and
 * a stale count there is a decision made on the wrong evidence.
 */
export function useSuggestForwarding() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ problemId, ...payload }) => assignmentsApi.suggestForwarding(problemId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: forwardingKeys.mine })
      queryClient.invalidateQueries({ queryKey: forwardingKeys.queue })
    },
  })
}

/**
 * The super admin's queue of reports awaiting a forward.
 * @returns {import('@tanstack/react-query').UseQueryResult<import('../lib/types/models.js').ForwardingItem[]>}
 */
export function useForwardingQueue() {
  return useQuery({ queryKey: forwardingKeys.queue, queryFn: assignmentsApi.listForwardingQueue })
}

/** One row of the super admin's queue by problem id, for the forward panel. */
export function useForwardingQueueItem(problemId) {
  return useQuery({
    queryKey: forwardingKeys.queue,
    queryFn: assignmentsApi.listForwardingQueue,
    select: useCallback((rows) => rows.find((r) => r.problemId === problemId) ?? null, [problemId]),
  })
}

/**
 * Forward an above-union report. The problem becomes Assigned, so it leaves both
 * forwarding surfaces and the public feed's status changes — hence all four
 * invalidations. `assignmentKeys.queue` is included because the union admin's queue
 * is a projection of the same problem rows.
 */
export function useForwardProblem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ problemId, ...payload }) => assignmentsApi.forwardProblem(problemId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: forwardingKeys.queue })
      queryClient.invalidateQueries({ queryKey: forwardingKeys.mine })
      queryClient.invalidateQueries({ queryKey: assignmentKeys.queue })
      queryClient.invalidateQueries({ queryKey: problemKeys.all })
    },
  })
}
