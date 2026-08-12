import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as suggestionsApi from '../lib/api/suggestions.js'
import { useAuth } from '../auth/useAuth.js'

export const suggestionKeys = {
  all: ['suggestions'],
  list: (problemId) => [...suggestionKeys.all, 'list', problemId],
}

/** @param {string} problemId */
export function useSuggestions(problemId) {
  const { user } = useAuth()
  return useQuery({
    // Keyed by viewer as well as problem, though the endpoint takes no caller: the
    // rows now carry myUpvote, which is the reader's OWN upvote, so one account's
    // answer must not survive a logout and be served to the next person on the same
    // tab. Cache isolation, exactly as problemKeys.mine and the caller-keyed problem
    // detail key are — see CLAUDE.md A.5.5 rule 2. The request itself carries no
    // caller; the JWT does.
    queryKey: [...suggestionKeys.list(problemId), user?.id ?? ''],
    queryFn: () => suggestionsApi.listSuggestions(problemId),
    enabled: Boolean(problemId),
  })
}

/** @param {string} problemId */
export function useProposeSuggestion(problemId) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (text) => suggestionsApi.proposeSuggestion(problemId, text),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: suggestionKeys.list(problemId) })
    },
  })
}

/** @param {string} problemId */
export function useUpvoteSuggestion(problemId) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (suggestionId) => suggestionsApi.upvoteSuggestion(suggestionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: suggestionKeys.list(problemId) })
    },
  })
}
