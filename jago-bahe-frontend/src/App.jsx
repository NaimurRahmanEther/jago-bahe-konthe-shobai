import { Suspense } from 'react'
import { useTranslation } from 'react-i18next'
import AppShell from './components/layout/AppShell.jsx'
import AppRoutes from './routes/AppRoutes.jsx'

function App() {
  const { t } = useTranslation()
  return (
    <AppShell>
      <Suspense fallback={<p role="status">{t('common.loading')}</p>}>
        <AppRoutes />
      </Suspense>
    </AppShell>
  )
}

export default App
