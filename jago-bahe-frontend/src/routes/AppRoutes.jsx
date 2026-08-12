import { Routes, Route } from 'react-router-dom'
import ProtectedRoute from './ProtectedRoute.jsx'
import Landing from '../pages/public/Landing.jsx'
import ProblemFeed from '../pages/public/ProblemFeed.jsx'
import ProblemDetail from '../pages/public/ProblemDetail.jsx'
import ReportProblem from '../pages/public/ReportProblem.jsx'
import EditProblem from '../pages/resident/EditProblem.jsx'
import MyReports from '../pages/resident/MyReports.jsx'
import Notifications from '../pages/notifications/Notifications.jsx'
import OfficialDirectory from '../pages/public/OfficialDirectory.jsx'
import SeatActivity from '../pages/public/SeatActivity.jsx'
import Scorecard from '../pages/public/Scorecard.jsx'
import Register from '../pages/auth/Register.jsx'
import RegisterOfficial from '../pages/auth/RegisterOfficial.jsx'
import Login from '../pages/auth/Login.jsx'
import OfficialDashboard from '../pages/official/OfficialDashboard.jsx'
import CaseDetail from '../pages/official/CaseDetail.jsx'
import Observation from '../pages/official/Observation.jsx'
import AdminHome from '../pages/admin/AdminHome.jsx'
import Moderation from '../pages/admin/Moderation.jsx'
import Residents from '../pages/admin/Residents.jsx'
import Queue from '../pages/admin/Queue.jsx'
import Validating from '../pages/admin/Validating.jsx'
import AssignmentPanel from '../pages/admin/AssignmentPanel.jsx'
import Forwarding from '../pages/admin/Forwarding.jsx'
import ForwardingPanel from '../pages/admin/ForwardingPanel.jsx'
import ForwardingQueue from '../pages/super/ForwardingQueue.jsx'
import ForwardPanel from '../pages/super/ForwardPanel.jsx'
import ClaimsReview from '../pages/claims/ClaimsReview.jsx'
import Oversight from '../pages/super/Oversight.jsx'

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
          "signed in, any role" — supported since F3 and first used here, because a
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
          nobody. Seniority is not a client-side fact — the backend decides who
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

      {/* super admin — a flat fourth role, NOT a superset of admin (it is refused
          every /admin route above by ProtectedRoute).
          /super stays the read-only oversight feed: the role oversees union admins
          and never overrides them. Forwarding below is not a counter-example — it
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
