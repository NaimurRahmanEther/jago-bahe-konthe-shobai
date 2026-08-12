import { useTranslation } from 'react-i18next'
import { usePendingResidents, useVerifyResident } from '../../hooks/useResidents.js'
import Button from '../../components/ui/Button.jsx'
import SkeletonList from '../../components/ui/Skeleton.jsx'
import EmptyState from '../../components/ui/EmptyState.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

export default function Residents() {
  const { t } = useTranslation()
  const { data: residents, isLoading, isError, refetch } = usePendingResidents()
  const verify = useVerifyResident()

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-h1 font-semibold text-ink">{t('admin.residents.title')}</h1>
        <p className="text-sm text-muted">{t('admin.residents.subtitle')}</p>
      </div>

      {isLoading && <SkeletonList count={3} />}

      {isError && <ErrorState message={t('admin.residents.error')} onRetry={refetch} />}

      {!isLoading && !isError && residents?.length === 0 && <EmptyState title={t('admin.residents.empty')} />}

      {!isLoading && !isError && residents?.length > 0 && (
        <div className="flex flex-col gap-3">
          {residents.map((resident, i) => (
            <div
              key={resident.id}
              style={{ animationDelay: `${Math.min(i, 5) * 40}ms` }}
              className="flex flex-wrap items-center justify-between gap-3 rounded-card border border-hairline bg-surface p-4 animate-rise-in"
            >
              <div className="flex flex-col gap-1">
                <p className="font-semibold text-ink">{resident.name}</p>
                <p className="text-meta text-muted">
                  {t('admin.residents.phone')}: {resident.phone}
                  {resident.nid && (
                    <span className="ml-2">
                      · {t('admin.residents.nid')}: {resident.nid}
                    </span>
                  )}
                </p>
              </div>
              <Button onClick={() => verify.mutate(resident.id)} disabled={verify.isPending}>
                {t('admin.residents.verify')}
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
