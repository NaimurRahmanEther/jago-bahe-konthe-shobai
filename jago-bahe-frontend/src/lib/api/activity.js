import { client } from './client.js'

/**
 * The seat's public decision record: what the union admins have done, in the open.
 *
 * Ungated and caller-blind — the feed is identical for everyone, signed in or not.
 * A per-viewer variation would make it a different record for different readers,
 * which is the opposite of a public record.
 *
 * It carries the five PROBLEM actions only (approved, rejected, assigned,
 * vote_opened, adjudicated). The three actions about PEOPLE — verified,
 * claim_approved, claim_rejected — stay in the super admin's oversight feed:
 * publishing them would index who is a verified resident of which union, and
 * broadcast refused identity claims. See getOversight in ./identity.js.
 *
 * @param {{limit?: number}} [opts]
 * @returns {Promise<import('../types/models.js').ActivityEntry[]>}
 */
export function listActivity({ limit } = {}) {
  return client
    .get('/seat/activity', { params: limit ? { limit } : undefined })
    .then((res) => res.data)
}
