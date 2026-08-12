import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as observationApi from '../lib/api/observation.js'
import { useAuth } from '../auth/useAuth.js'

export const observationKeys = {
  all: ['observations'],
}

/**
 * The cases the signed-in official monitors — every one they are the direct
 * monitor of, plus any that climbed to them from further down the ladder.
 *
 * `enabled` mirrors useCases: an official whose claim is not yet approved has no
 * officialId, and an empty list would read as "nobody below you has gone quiet"
 * rather than "your claim is pending".
 */
export function useObservations() {
  const { user } = useAuth()
  return useQuery({
    queryKey: observationKeys.all,
    queryFn: () => observationApi.listObservations(),
    enabled: Boolean(user?.officialId),
  })
}

/** Record what the monitor did about a silence. */
export function useNoteObservation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ observationId, text }) => observationApi.noteObservation(observationId, { text }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: observationKeys.all })
    },
  })
}

/**
 * Whether any rung is still open on a row — i.e. somebody is waiting on this
 * official right now. Derived here rather than recomputed per component so the
 * page and the row cannot disagree about what "waiting" means.
 * @param {import('../lib/types/models.js').ObservedCase} row
 */
export function isWaiting(row) {
  return (row.observations ?? []).some((o) => o.resolvedAt === null)
}
