import { useQuery } from '@tanstack/react-query'
import * as areasApi from '../lib/api/areas.js'

/**
 * The seat's geography, optionally sliced to one level or one parent.
 *
 * One query, several consumers. The key is constant and the filter lives in
 * `select`, so Register, LocationPicker and ProblemFeed share a single cache
 * entry rather than fetching the same eleven rows three times — which is also why
 * the API takes no ?level= / ?parent= (CLAUDE.md A.1.1: smallest surface).
 *
 * staleTime: Infinity — a seat's unions do not change while someone is filling in
 * a form, so there is nothing for a refetch to discover.
 *
 * `level: 'union'` returns the eight union parishads AND Dhamoirhat Pourashava:
 * the municipality is seeded as a union-level node because areas.level has no
 * pourashava, and because the mayor is already union-level for routing.
 *
 * @param {{level?: import('../lib/types/models.js').AreaLevel, parentId?: string}} [filter]
 */
export function useAreas({ level, parentId } = {}) {
  return useQuery({
    queryKey: ['areas'],
    queryFn: areasApi.listAreas,
    staleTime: Infinity,
    select: (areas) =>
      areas.filter(
        (a) => (level === undefined || a.level === level) && (parentId === undefined || a.parentId === parentId),
      ),
  })
}
