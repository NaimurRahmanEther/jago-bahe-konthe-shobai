import { lazy } from 'react'
import { Routes, Route } from 'react-router-dom'
import ProtectedRoute from './ProtectedRoute.jsx'
const Landing = lazy(() => import('../pages/public/Landing.jsx'))
const ProblemFeed = lazy(() => import('../pages/public/ProblemFeed.jsx'))
const ProblemDetail = lazy(() => import('../pages/public/ProblemDetail.jsx'))
const ReportProblem = lazy(() => import('../pages/public/ReportProblem.jsx'))
const EditProblem = lazy(() => import('../pages/resident/EditProblem.jsx'))
const MyReports = lazy(() => import('../pages/resident/MyReports.jsx'))
const Notifications = lazy(() => import('../pages/notifications/Notifications.jsx'))
const OfficialDirectory = lazy(() => import('../pages/public/OfficialDirectory.jsx'))
const SeatActivity = lazy(() => import('../pages/public/SeatActivity.jsx'))
const Scorecard = lazy(() => import('../pages/public/Scorecard.jsx'))
const Register = lazy(() => import('../pages/auth/Register.jsx'))
const RegisterOfficial = lazy(() => import('../pages/auth/RegisterOfficial.jsx'))
const Login = lazy(() => import('../pages/auth/Login.jsx'))
const OfficialDashboard = lazy(() => import('../pages/official/OfficialDashboard.jsx'))
const CaseDetail = lazy(() => import('../pages/official/CaseDetail.jsx'))
const Observation = lazy(() => import('../pages/official/Observation.jsx'))
const AdminHome = lazy(() => import('../pages/admin/AdminHome.jsx'))
const Moderation = lazy(() => import('../pages/admin/Moderation.jsx'))
const Residents = lazy(() => import('../pages/admin/Residents.jsx'))
const Queue = lazy(() => import('../pages/admin/Queue.jsx'))
const Validating = lazy(() => import('../pages/admin/Validating.jsx'))
const AssignmentPanel = lazy(() => import('../pages/admin/AssignmentPanel.jsx'))
const Forwarding = lazy(() => import('../pages/admin/Forwarding.jsx'))
const ForwardingPanel = lazy(() => import('../pages/admin/ForwardingPanel.jsx'))
const ForwardingQueue = lazy(() => import('../pages/super/ForwardingQueue.jsx'))
const ForwardPanel = lazy(() => import('../pages/super/ForwardPanel.jsx'))
const ClaimsReview = lazy(() => import('../pages/claims/ClaimsReview.jsx'))
const Oversight = lazy(() => import('../pages/super/Oversight.jsx'))

export default function AppRoutes() {
  return (
    <Routes>
      {/* public */}
      <Route path="/" element={<Landing />} />
      <Route path="/problems" element={<ProblemFeed />} />
      <Route path="/problems/:id" element={<ProblemDetail />} />
      <Route path="/officials" element={<OfficialDirectory />} />
      {/* Public by design: the seat's decision record is for residents to read,
          not for one appointed watcher (the super admin's /super keeps the
          person-actions half). No ProtectedRoute. */}
      <Route path="/activity" element={<SeatActivity />} />
      <Route path="/officials/:id/scorecard" element={<Scorecard />} />
      <Route path="/register" element={<Register />} />
      <Route path="/register/official" element={<RegisterOfficial />} />
      <Route path="/login" element={<Login />} />

      {/* any signed-in role (B21). ProtectedRoute with NO allowedRoles means
          "signed in, any role" â€” supported since F3 and first used here, because a
          notification is about the CALLER rather than about their office. Roles
          stay flat: this is not a role widening, it is the absence of a role
          question. */}
      <Route
        path="/notifications"
        element={
          <ProtectedRoute>
            <Notifications />
          </ProtectedRoute>
        }
      />

      {/* resident */}
      <Route
        path="/me"
        element={
          <ProtectedRoute allowedRoles={['resident']}>
            <MyReports />
          </ProtectedRoute>
        }
      />
      <Route
        path="/report"
        element={
          <ProtectedRoute allowedRoles={['resident']}>
            <ReportProblem />
          </ProtectedRoute>
        }
      />
      <Route
        path="/problems/:id/edit"
        element={
          <ProtectedRoute allowedRoles={['resident']}>
            <EditProblem />
          </ProtectedRoute>
        }
      />

      {/* official */}
      <Route
        path="/official"
        element={
          <ProtectedRoute allowedRoles={['official']}>
            <OfficialDashboard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/official/cases/:id"
        element={
          <ProtectedRoute allowedRoles={['official']}>
            <CaseDetail />
          </ProtectedRoute>
        }
      />
      {/* Every official may open this; the list is empty for one who monitors
          nobody. Seniority is not a client-side fact â€” the backend decides who
          appears here, from the ladder (A.5.7). */}
      <Route
        path="/official/observations"
        element={
          <ProtectedRoute allowedRoles={['official']}>
            <Observation />
          </ProtectedRoute>
        }
      />

      {/* admin */}
      <Route
        path="/admin"
        element={
          <ProtectedRoute allowedRoles={['admin']}>
            <AdminHome />
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/moderation"
        element={
          <ProtectedRoute allowedRoles={['admin']}>
            <Moderation />
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/residents"
        element={
          <ProtectedRoute allowedRoles={['admin']}>
            <Residents />
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/validating"
        element={
          <ProtectedRoute allowedRoles={['admin']}>
            <Validating />
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/queue"
        element={
          <ProtectedRoute allowedRoles={['admin']}>
            <Queue />
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/problems/:id/assign"
        element={
          <ProtectedRoute allowedRoles={['admin']}>
            <AssignmentPanel />
          </ProtectedRoute>
        }
      />
      {/* Advice, not a decision. An above-union report is the super admin's to
          forward; the union admin's part is to say where it should go, so these
          two routes replace the admin-vote screens B4 had here (A.3.8). */}
      <Route
        path="/admin/forwarding"
        element={
          <ProtectedRoute allowedRoles={['admin']}>
            <Forwarding />
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/forwarding/:id"
        element={
          <ProtectedRoute allowedRoles={['admin']}>
            <ForwardingPanel />
          </ProtectedRoute>
        }
      />

      {/* Claim review is ONE route for both reviewer roles. The backend gates it
          with RequireClaimReviewer (admin OR super_admin) and routes each claim by
          the office's tier inside the use case; splitting it into /admin/claims and
          /super/claims would re-encode that rule in URLs and let the two drift. */}
      <Route
        path="/claims"
        element={
          <ProtectedRoute allowedRoles={['admin', 'super_admin']}>
            <ClaimsReview />
          </ProtectedRoute>
        }
      />

      {/* super admin â€” a flat fourth role, NOT a superset of admin (it is refused
          every /admin route above by ProtectedRoute).
          /super stays the read-only oversight feed: the role oversees union admins
          and never overrides them. Forwarding below is not a counter-example â€” it
          is a FRESH decision on a report no admin was entitled to decide, not the
          reversal of one that was made (A.3.8). It gets its own route so the page
          whose whole point is having no controls keeps having none. */}
      <Route
        path="/super"
        element={
          <ProtectedRoute allowedRoles={['super_admin']}>
            <Oversight />
          </ProtectedRoute>
        }
      />
      <Route
        path="/super/queue"
        element={
          <ProtectedRoute allowedRoles={['super_admin']}>
            <ForwardingQueue />
          </ProtectedRoute>
        }
      />
      <Route
        path="/super/problems/:id/forward"
        element={
          <ProtectedRoute allowedRoles={['super_admin']}>
            <ForwardPanel />
          </ProtectedRoute>
        }
      />
    </Routes>
  )
}
