import { useQuery } from '@tanstack/react-query'
import * as activityApi from '../lib/api/activity.js'

/**
 * The seat's public decision record.
 *
 * Deliberately NOT keyed by the caller, unlike problemKeys.mine / the detail key
 * (A.5.5 rule 2). Those are keyed because the same request legitimately has
 * different answers for different accounts, so sharing a cache entry would serve
 * one reader another's data. This endpoint answers every reader identically, so a
 * caller key would only fragment the cache and hide that fact.
 *
 * No staleTime: unlike the geography, this changes whenever an admin acts, and a
 * stale accountability record is the one thing this page must not show.
 *
 * @param {{limit?: number}} [opts]
 */
export function useActivity({ limit } = {}) {
  return useQuery({
    queryKey: ['activity', limit ?? null],
    queryFn: () => activityApi.listActivity({ limit }),
  })
}
