import { createContext, useCallback, useEffect, useState } from 'react'
import * as authApi from '../lib/api/auth.js'
import { setAuthToken } from '../lib/api/client.js'

export const AuthContext = createContext(null)

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
  const [session, setSession] = useState(readStoredSession)

  useEffect(() => {
    setAuthToken(session?.token ?? null)
  }, [session])

  const login = useCallback(async (phone, password) => {
    const result = await authApi.login({ phone, password })
    const next = { token: result.token, role: result.role, user: result.user }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
    setSession(next)
    return next
  }, [])

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
    setSession(null)
  }, [])

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
