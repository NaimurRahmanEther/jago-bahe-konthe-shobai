import { client } from './client.js'

/**
 * @param {string} problemId
 * @returns {Promise<import('../types/models.js').Suggestion[]>}
 */
export function listSuggestions(problemId) {
  return client.get(`/problems/${problemId}/suggestions`).then((res) => res.data)
}

/**
 * @param {string} problemId
 * @param {string} text
 * @returns {Promise<import('../types/models.js').Suggestion>}
 */
export function proposeSuggestion(problemId, text) {
  return client.post(`/problems/${problemId}/suggestions`, { text }).then((res) => res.data)
}

/**
 * @param {string} suggestionId
 * @returns {Promise<import('../types/models.js').Suggestion>}
 */
export function upvoteSuggestion(suggestionId) {
  return client.post(`/suggestions/${suggestionId}/upvote`).then((res) => res.data)
}
