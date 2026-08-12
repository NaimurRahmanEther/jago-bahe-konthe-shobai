import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useScorecard } from '../../hooks/useScorecard.js'
import { useOfficials } from '../../hooks/useOfficials.js'
import Spinner from '../../components/ui/Spinner.jsx'
import StatTile from '../../components/ui/StatTile.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

export default function Scorecard() {
  const { t } = useTranslation()
  const { id } = useParams()
  const { data: officials } = useOfficials()
  const { data: stats, isLoading, isError, refetch } = useScorecard(id)
  const official = officials?.find((o) => o.id === id)

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  if (isError || !stats) {
    return <ErrorState message={t('scorecard.error')} onRetry={refetch} />
  }

  return (
    <div className="flex w-full flex-col gap-4">
      <Link to="/officials" className="text-sm font-medium text-brand">
        {t('scorecard.back')}
      </Link>
      <h1 className="text-h1 font-semibold text-ink">
        {official ? official.name : t('scorecard.title')}
        {official && (
          <span className="ml-2 text-sm font-normal text-muted">({t(`problem.tier.${official.tier}`)})</span>
        )}
      </h1>

      <div className="grid grid-cols-2 gap-3">
        <StatTile value={stats.resolved} label={t('scorecard.resolved')} tone="resolved" />
        <StatTile value={stats.pending} label={t('scorecard.pending')} />
        <StatTile value={stats.blocked} label={t('scorecard.blocked')} tone="blocked" />
        <StatTile value={stats.avgResponseDays} label={t('scorecard.avgResponse')} />
      </div>

      {stats.blocked > 0 && <div className="rounded-card bg-blocked-bg p-3 text-sm text-blocked-fg">{t('scorecard.fairnessNote')}</div>}
    </div>
  )
}
