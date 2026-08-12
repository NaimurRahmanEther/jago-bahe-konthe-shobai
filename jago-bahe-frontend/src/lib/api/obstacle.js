import { client } from './client.js'

/**
 * @param {string} problemId
 * @returns {Promise<import('../types/models.js').Obstacle | null>}
 */
export function getObstacle(problemId) {
  return client.get(`/problems/${problemId}/obstacle`).then((res) => res.data)
}

/**
 * @param {string} problemId
 * @param {{vote: 'real' | 'not_convinced'}} payload
 * @returns {Promise<import('../types/models.js').Obstacle>}
 */
export function voteObstacle(problemId, payload) {
  return client.post(`/problems/${problemId}/obstacle/vote`, payload).then((res) => res.data)
}

/**
 * @param {string} problemId
 * @param {string} text
 * @returns {Promise<import('../types/models.js').UnblockingPlan>}
 */
export function proposeUnblockingPlan(problemId, text) {
  return client.post(`/problems/${problemId}/obstacle/plans`, { text }).then((res) => res.data)
}

/**
 * @param {string} problemId
 * @param {string} planId
 * @returns {Promise<import('../types/models.js').UnblockingPlan>}
 */
export function upvoteUnblockingPlan(problemId, planId) {
  return client.post(`/obstacle/plans/${planId}/upvote`).then((res) => res.data)
}

// TODO(contract): POST /admin/obstacles/{id}/adjudicate is still not in Scaffold
// Spec §5's table. §5 says the adjudication endpoint is backend-owned and "not
// yet in this contract — mark them TODO(contract) and define them in B5/B6"; B6
// defined it (resolution/interfaces/http/b6_handler.go carries the matching
// TODO) but §5 was never amended. The four obstacle routes above ARE contracted
// (§5) — this note covers adjudicate alone.

/**
 * Adjudicate a obstacle (admin/moderator authority). This — not the advisory
 * public vote — sets the scorecard consequence (Concept §8/§9). Backend gate:
 * RequireAdmin; the adjudicator identity comes from the JWT, not the body.
 * @param {string} obstacleId
 * @param {'confirm' | 'deny'} decision
 * @returns {Promise<import('../types/models.js').Obstacle>}
 */
export function adjudicateObstacle(obstacleId, decision) {
  return client.post(`/admin/obstacles/${obstacleId}/adjudicate`, { decision }).then((res) => res.data)
}
