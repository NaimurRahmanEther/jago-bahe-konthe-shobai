import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import ProtectedRoute from './ProtectedRoute.jsx'

const h = vi.hoisted(() => ({ role: null }))
vi.mock('../auth/useAuth.js', () => ({ useAuth: () => ({ role: h.role }) }))

// Render the guard at /secret with sibling routes so a redirect is observable.
function renderGuard(allowedRoles) {
  return render(
    <MemoryRouter initialEntries={['/secret']}>
      <Routes>
        <Route
          path="/secret"
          element={
            <ProtectedRoute allowedRoles={allowedRoles}>
              <div>secret content</div>
            </ProtectedRoute>
          }
        />
        <Route path="/login" element={<div>login page</div>} />
        <Route path="/" element={<div>home page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  h.role = null
})

describe('ProtectedRoute', () => {
  it('redirects an unauthenticated user to login', () => {
    h.role = null
    renderGuard(['admin'])
    expect(screen.getByText('login page')).toBeInTheDocument()
    expect(screen.queryByText('secret content')).toBeNull()
  })

  it('redirects a signed-in user without an allowed role home', () => {
    h.role = 'resident'
    renderGuard(['admin'])
    expect(screen.getByText('home page')).toBeInTheDocument()
    expect(screen.queryByText('secret content')).toBeNull()
  })

  it('renders the children when the role is allowed', () => {
    h.role = 'admin'
    renderGuard(['admin'])
    expect(screen.getByText('secret content')).toBeInTheDocument()
  })

  // The client-side pin for "no supersets": the super admin is a FOURTH role, not
  // a rung above admin, so it must be REFUSED every /admin route — the same rule
  // the backend's RequireAdmin enforces (a super_admin fails it by design). If this
  // ever "helpfully" started admitting super_admin, it would teach a hierarchy the
  // platform deliberately does not have (CLAUDE.md F13, A.3.1).
  it('refuses a super_admin an admin-only route', () => {
    h.role = 'super_admin'
    renderGuard(['admin'])
    expect(screen.getByText('home page')).toBeInTheDocument()
    expect(screen.queryByText('secret content')).toBeNull()
  })

  it('admits a super_admin to a route that allows the role', () => {
    h.role = 'super_admin'
    renderGuard(['admin', 'super_admin'])
    expect(screen.getByText('secret content')).toBeInTheDocument()
  })

  it('allows any signed-in role when allowedRoles is omitted', () => {
    h.role = 'official'
    renderGuard(undefined)
    expect(screen.getByText('secret content')).toBeInTheDocument()
  })
})
