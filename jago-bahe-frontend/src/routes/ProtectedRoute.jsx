import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../auth/useAuth.js'

// Role guards are UX only — the backend is the real authority (CLAUDE.md A.5.7).
export default function ProtectedRoute({ allowedRoles, children }) {
  const { role } = useAuth()
  const location = useLocation()

  if (!role) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  if (allowedRoles && !allowedRoles.includes(role)) {
    return <Navigate to="/" replace />
  }

  return children
}
