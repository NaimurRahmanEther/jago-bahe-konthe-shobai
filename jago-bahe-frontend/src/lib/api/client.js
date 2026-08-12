import axios from 'axios'
import { API_URL } from '../../config/env.js'

export const client = axios.create({ baseURL: API_URL })

let authToken = null

/** Called by AuthContext on login/logout (F3) so the request interceptor can attach the JWT. */
export function setAuthToken(token) {
  authToken = token
}

client.interceptors.request.use((config) => {
  if (authToken) {
    config.headers.Authorization = `Bearer ${authToken}`
  }
  return config
})

client.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error.response?.status ?? 0
    // The backend's error envelope is {code, message} (pkg/httpx.ErrorBody), and
    // `code` is the stable, machine-readable half — `message` is English prose
    // written for a developer, and this app renders Bangla. Screens that want to
    // explain a refusal should switch on `code`, never on the message text and
    // rarely on the status alone: one status covers several refusals (403 is both
    // not_verified and not_area_resident, which need different advice).
    const code = error.response?.data?.code ?? ''
    const message = error.response?.data?.message ?? error.message ?? 'Network error'
    return Promise.reject({ status, code, message })
  },
)
