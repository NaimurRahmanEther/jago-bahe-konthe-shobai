import { useEffect } from 'react'
import { act, render, renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, expect, it, vi } from 'vitest'
import { AuthProvider } from './AuthProvider.jsx'
import { useAuth } from './useAuth.js'
import * as authApi from '../lib/api/auth.js'
import { setAuthToken } from '../lib/api/client.js'

vi.mock('../lib/api/auth.js', () => ({ login: vi.fn(), register: vi.fn() }))
vi.mock('../lib/api/client.js', () => ({ setAuthToken: vi.fn() }))

beforeEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
})

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }) => (
    <QueryClientProvider client={client}>
      <AuthProvider>{children}</AuthProvider>
    </QueryClientProvider>
  )
  return { client, wrapper }
}

it('restores the token before child request effects run', () => {
  localStorage.setItem('jago-bahe-auth', JSON.stringify({ token: 'stored-token' }))
  const observed = vi.fn()
  function Child() {
    useEffect(() => { observed(setAuthToken.mock.lastCall?.[0]) }, [])
    return null
  }
  const { wrapper } = setup()
  render(<Child />, { wrapper })
  expect(observed).toHaveBeenCalledWith('stored-token')
})

it('clears cached private data and credentials when logging out', () => {
  const { client, wrapper } = setup()
  const { result } = renderHook(useAuth, { wrapper })
  client.setQueryData(['residents', 'pending'], [{ id: 'private-resident' }])
  act(() => { result.current.logout() })
  expect(client.getQueryCache().getAll()).toHaveLength(0)
  expect(setAuthToken).toHaveBeenLastCalledWith(null)
  expect(result.current.user).toBeNull()
})

it('discards the previous account cache before exposing a new session', async () => {
  const { client, wrapper } = setup()
  const { result } = renderHook(useAuth, { wrapper })
  client.setQueryData(['assignments'], [{ id: 'previous-account' }])
  authApi.login.mockResolvedValue({ token: 'new-token', role: 'admin', user: { id: 'new-user' } })
  await act(async () => { await result.current.login('phone', 'password') })
  expect(client.getQueryCache().getAll()).toHaveLength(0)
  expect(result.current.user.id).toBe('new-user')
  expect(setAuthToken).toHaveBeenLastCalledWith('new-token')
})
