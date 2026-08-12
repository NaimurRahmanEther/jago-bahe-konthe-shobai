import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import RoleNav from './RoleNav.jsx'

export default function Navbar() {
  const { t } = useTranslation()

  return (
    <header className="sticky top-0 z-10 border-b border-hairline bg-surface">
      <div className="flex min-h-14 w-full flex-wrap items-center justify-between gap-y-1 px-4 py-2">
        <Link to="/" className="font-semibold text-brand-dark">
          {t('app.name')}
        </Link>
        <RoleNav />
      </div>
    </header>
  )
}
