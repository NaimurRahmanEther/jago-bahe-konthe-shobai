import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as identityApi from '../lib/api/identity.js'
import { oversightKeys } from './useOversight.js'

export const residentKeys = {
  pending: ['residents', 'pending'],
}

/** The admin's own union's unverified residents — the queue that makes V reachable. */
export function usePendingResidents() {
  return useQuery({ queryKey: residentKeys.pending, queryFn: identityApi.listPendingResidents })
}

export function useVerifyResident() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (residentId) => identityApi.verifyResident(residentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: residentKeys.pending })
      // Verification is an audited decision; the super admin's feed reads it.
      queryClient.invalidateQueries({ queryKey: oversightKeys.all })
    },
  })
}
