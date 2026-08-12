import { client } from './client.js'

/**
 * @param {string} officialId
 * @returns {Promise<import('../types/models.js').OfficialStats>}
 */
export function getOfficialScorecard(officialId) {
  return client.get(`/officials/${officialId}/scorecard`).then((res) => res.data)
}

/**
 * Seat-wide totals. Every count is over publicly visible problems only — an
 * unscreened report must not be countable, or the number leaks how many exist.
 * @returns {Promise<import('../types/models.js').SeatOverview>}
 */
export function getSeatOverview() {
  return client.get('/seat/overview').then((res) => res.data)
}
