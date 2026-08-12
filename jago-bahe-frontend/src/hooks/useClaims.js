import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as identityApi from '../lib/api/identity.js'
import { oversightKeys } from './useOversight.js'

export const claimKeys = {
  pending: ['claims', 'pending'],
}

/** The claims the signed-in reviewer may decide (server-filtered by the office's tier). */
export function usePendingClaims() {
  return useQuery({ queryKey: claimKeys.pending, queryFn: identityApi.listPendingClaims })
}

export function useApproveClaim() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (claimId) => identityApi.approveClaim(claimId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: claimKeys.pending })
      // Oversight reads the audit log this approval just wrote to.
      queryClient.invalidateQueries({ queryKey: oversightKeys.all })
      // Approval sets accounts.official_id, so who the directory's entries resolve
      // to has changed — the officials list is now stale.
      queryClient.invalidateQueries({ queryKey: ['officials'] })
    },
  })
}

export function useRejectClaim() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ claimId, reason }) => identityApi.rejectClaim(claimId, { reason }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: claimKeys.pending })
      queryClient.invalidateQueries({ queryKey: oversightKeys.all })
    },
  })
}
