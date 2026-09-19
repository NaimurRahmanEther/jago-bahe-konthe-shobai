import { useCallback, useLayoutEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import * as authApi from '../lib/api/auth.js'
import { setAuthToken } from '../lib/api/client.js'

import { AuthContext } from './context.js'

const STORAGE_KEY = 'jago-bahe-auth'

function readStoredSession() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export function AuthProvider({ children }) {
  const queryClient = useQueryClient()
  const [session, setSession] = useState(readStoredSession)

  // Restore credentials before child query effects issue their first requests.
  useLayoutEffect(() => {
    setAuthToken(session?.token ?? null)
  }, [session])

  const login = useCallback(async (phone, password) => {
    const result = await authApi.login({ phone, password })
    const next = { token: result.token, role: result.role, user: result.user }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
    setAuthToken(next.token)
    // Some role-specific queries use constant keys. Never retain another
    // account's cached responses or in-flight queries across session changes.
    queryClient.clear()
    setSession(next)
    return next
  }, [queryClient])

  const register = useCallback((payload) => authApi.register(payload), [])

  // Merge fields into the stored user. This exists for ONE reason: `user.verified`
  // is captured at login and there is no endpoint to re-read it (no GET /api/me),
  // so an admin verifying a resident who is already signed in leaves that session
  // claiming unverified until they log out and back in — and nothing tells them to.
  //
  // The server is still the authority (A.5.7): callers use this to record what a
  // successful write already PROVED, never to grant themselves a capability. A
  // validation the backend accepted is proof the account is verified, so
  // ValidationVote heals the stale flag with it.
  const updateUser = useCallback((fields) => {
    setSession((prev) => {
      if (!prev) return prev
      const next = { ...prev, user: { ...prev.user, ...fields } }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
      return next
    })
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY)
    setAuthToken(null)
    queryClient.clear()
    setSession(null)
  }, [queryClient])

  const value = {
    user: session?.user ?? null,
    role: session?.role ?? null,
    token: session?.token ?? null,
    login,
    register,
    logout,
    updateUser,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
