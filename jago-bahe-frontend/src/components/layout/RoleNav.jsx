import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/useAuth.js'
import NotificationBell from '../notification/NotificationBell.jsx'

export default function RoleNav() {
  const { t } = useTranslation()
  const { role, logout } = useAuth()

  return (
    <nav aria-label={t('nav.primary')} className="flex items-center gap-3 text-sm font-medium">
      <Link to="/problems" className="text-ink hover:text-brand">
        {t('nav.problems')}
      </Link>
      <Link to="/officials" className="text-ink hover:text-brand">
        {t('nav.directory')}
      </Link>
      {/* Outside every role branch on purpose: the decision record is public, and
          a logged-out resident is exactly who it is for. */}
      <Link to="/activity" className="text-ink hover:text-brand">
        {t('nav.activity')}
      </Link>
      {role === 'resident' && (
        <Link to="/me" className="text-ink hover:text-brand">
          {t('nav.myReports')}
        </Link>
      )}
      {role === 'official' && (
        <>
          <Link to="/official" className="text-ink hover:text-brand">
            {t('nav.officialDashboard')}
          </Link>
          {/* Shown to every official, not only senior ones: the nav cannot know
              who monitors whom, and the page is honestly empty for those who
              monitor nobody. */}
          <Link to="/official/observations" className="text-ink hover:text-brand">
            {t('nav.officialObservation')}
          </Link>
        </>
      )}
      {role === 'admin' && (
        <>
          <Link to="/admin" className="text-ink hover:text-brand">
            {t('nav.adminHome')}
          </Link>
          <Link to="/admin/moderation" className="text-ink hover:text-brand">
            {t('nav.adminModeration')}
          </Link>
          {/* The three queues follow a report's life in order: screening →
              community validation → assignment. The last two had no nav link at
              all; /admin/queue was reachable only from the dashboard. */}
          <Link to="/admin/validating" className="text-ink hover:text-brand">
            {t('nav.adminValidating')}
          </Link>
          <Link to="/admin/queue" className="text-ink hover:text-brand">
            {t('nav.adminQueue')}
          </Link>
          {/* Advice, not a decision — an above-union report is the super admin's
              to forward (A.3.8). Shown to every admin, since advice is scoped to
              the upazila rather than to their own union. */}
          <Link to="/admin/forwarding" className="text-ink hover:text-brand">
            {t('nav.adminForwarding')}
          </Link>
          <Link to="/admin/residents" className="text-ink hover:text-brand">
            {t('nav.adminResidents')}
          </Link>
          <Link to="/claims" className="text-ink hover:text-brand">
            {t('nav.claims')}
          </Link>
        </>
      )}
      {role === 'super_admin' && (
        <>
          <Link to="/super" className="text-ink hover:text-brand">
            {t('nav.oversight')}
          </Link>
          <Link to="/super/queue" className="text-ink hover:text-brand">
            {t('nav.superQueue')}
          </Link>
          <Link to="/claims" className="text-ink hover:text-brand">
            {t('nav.claims')}
          </Link>
        </>
      )}
      {role ? (
        <>
          {/* Outside every role branch, like /activity above but for the opposite
              reason: a notification is about the CALLER rather than their office,
              so all four roles have one. It renders nothing when signed out. */}
          <NotificationBell />
          <button
            type="button"
            onClick={logout}
            className="min-h-11 text-ink hover:text-brand"
          >
            {t('nav.logout')}
          </button>
        </>
      ) : (
        <>
          <Link to="/login" className="text-ink hover:text-brand">
            {t('nav.login')}
          </Link>
          <Link to="/register" className="text-ink hover:text-brand">
            {t('nav.register')}
          </Link>
        </>
      )}
    </nav>
  )
}
