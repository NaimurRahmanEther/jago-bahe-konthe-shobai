import { useQuery } from '@tanstack/react-query'
import * as identityApi from '../lib/api/identity.js'

// Oversight is the projection every identity mutation writes into: approving or
// rejecting a claim, and verifying a resident, all append to the audit log this
// feed reads. So each of those hooks invalidates oversightKeys.all — the same way
// useModeration invalidates problemKeys.all after a takedown.
export const oversightKeys = {
  all: ['oversight'],
}

/** The super admin's read-only feed of recent moderator decisions across the seat. */
export function useOversight() {
  return useQuery({ queryKey: oversightKeys.all, queryFn: identityApi.getOversight })
}
