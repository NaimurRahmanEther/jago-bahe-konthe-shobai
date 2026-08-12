import { useQuery } from '@tanstack/react-query'
import * as officialsApi from '../lib/api/officials.js'

// The seat's officials directory. One constant key and staleTime: Infinity, so the
// six-odd screens that resolve an official id to a name share a single cache entry
// — the directory is a near-static record of who holds which office, and refetching
// it on every mount and window focus (the default staleTime: 0) bought nothing.
// Mirrors useAreas, which caches the seat's geography the same way and for the same
// reason. Approving a claim is the one thing that genuinely changes who an entry
// resolves to, and useClaims already invalidates ['officials'] when it does — which
// is what makes an infinite staleTime safe here rather than merely convenient.
export function useOfficials() {
  return useQuery({
    queryKey: ['officials'],
    queryFn: officialsApi.listOfficials,
    staleTime: Infinity,
  })
}
