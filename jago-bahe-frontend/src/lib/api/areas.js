import { client } from './client.js'

/**
 * The seat's whole geography, flat.
 *
 * Takes no filter: one seat is bounded by design (eleven rows today — the seat,
 * Dhamoirhat upazila, its eight unions and the pourashava), so consumers derive
 * the slice they need client-side via TanStack's `select` rather than the API
 * growing ?level= / ?parent= surface.
 *
 * Ungated — the public geography of a public seat. Register reads it before the
 * resident has an account to authenticate with.
 *
 * @returns {Promise<import('../types/models.js').Area[]>}
 */
export function listAreas() {
  return client.get('/areas').then((res) => res.data)
}
