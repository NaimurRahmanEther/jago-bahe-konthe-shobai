import { useQuery } from '@tanstack/react-query'
import * as scorecardApi from '../lib/api/scorecard.js'

/** @param {string} officialId */
export function useScorecard(officialId) {
  return useQuery({
    queryKey: ['scorecard', officialId],
    queryFn: () => scorecardApi.getOfficialScorecard(officialId),
    enabled: Boolean(officialId),
  })
}

/** Seat-wide totals for the admin home. */
export function useSeatOverview() {
  return useQuery({ queryKey: ['scorecard', 'seat'], queryFn: scorecardApi.getSeatOverview })
}
