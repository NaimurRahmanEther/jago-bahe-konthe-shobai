import { client } from './client.js'

/** @returns {Promise<import('../types/models.js').Official[]>} */
export function listOfficials() {
  return client.get('/officials').then((res) => res.data)
}
