import { client } from './client.js'

/**
 * The cases the signed-in official monitors (B19). Takes no parameter: the
 * monitor is the caller, read from the JWT. An officialId here would let any
 * official read any other official's supervision list.
 * @returns {Promise<import('../types/models.js').ObservedCase[]>}
 */
export function listObservations() {
  return client.get('/official/observations').then((res) => res.data)
}

/**
 * Record what the monitor did about a silence. The note is appended (never
 * overwritten) and lands on the problem's public audit trail.
 *
 * This is the ONLY write an observer has. There is deliberately no call here
 * that reassigns, takes over or closes the case: silence moves visibility up the
 * ladder and leaves responsibility below (Concept §7).
 * @param {string} observationId
 * @param {{text: string}} payload
 * @returns {Promise<import('../types/models.js').ObservationNote>}
 */
export function noteObservation(observationId, payload) {
  return client.post(`/official/observations/${observationId}/note`, payload).then((res) => res.data)
}
