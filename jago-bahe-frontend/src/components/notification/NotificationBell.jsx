import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/useAuth.js'
import { useUnreadCount } from '../../hooks/useNotifications.js'
import { toBengaliDigits } from '../../lib/numerals.js'

/** Inline, not an icon library — the spec forbids a component kit (Guideline §11).
 *  Copied in shape from PhotoGlyph in ui/ImagePicker.jsx. */
function BellGlyph() {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-5 w-5"
    >
      <path d="M18 8a6 6 0 1 0-12 0c0 6-2 7-2 7h16s-2-1-2-7" />
      <path d="M13.7 20a2 2 0 0 1-3.4 0" />
    </svg>
  )
}

/**
 * The nav's link to /notifications, with an unread count.
 *
 * NEVER ICON-ONLY. Guideline §9 is explicit — "icons always with text, never
 * icon-only (literacy + clarity)" — so the word বিজ্ঞপ্তি sits beside the glyph
 * and is not optional. The count is a pill, not a bare colour: Guideline §2 rule
 * 1 forbids colour as the only signal, and the aria-label spells the whole thing
 * out for a screen reader.
 *
 * The pill is BRAND TEAL. Both reds are already spoken for — bright #dc2626 is
 * Reopened, muted maroon is Rejected — and an unread count is not an adverse
 * verdict; amber means "waiting on someone" and would make a routine badge read
 * like a stalled case.
 *
 * It shares its query key with the /notifications page, so opening the page costs
 * no second request — the OfficialDashboard/useObservations precedent.
 */
export default function NotificationBell() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { data } = useUnreadCount()

  // Signed out there is nothing to show and nothing to fetch; the hook is already
  // disabled, and rendering an empty bell would invite a click to a guarded route.
  if (!user) return null

  const count = data?.count ?? 0

  return (
    <Link
      to="/notifications"
      className="inline-flex min-h-11 items-center gap-1.5 text-ink hover:text-brand"
      aria-label={count > 0 ? t('notification.bell.unread', { count }) : t('notification.bell.none')}
    >
      <BellGlyph />
      <span>{t('notification.bell.label')}</span>
      {count > 0 && (
        // The number reaches the DOM without passing through t(), so it converts
        // here (A.5.3 rule 5) — i18next's formatter only reaches interpolated
        // values, and a Latin digit in a Bangla nav is the tell that it was missed.
        <span className="inline-flex min-w-5 items-center justify-center rounded-full bg-brand px-1.5 py-0.5 text-micro font-semibold text-surface">
          {toBengaliDigits(count)}
        </span>
      )}
    </Link>
  )
}
