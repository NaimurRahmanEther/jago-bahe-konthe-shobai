import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useMyForwardingItem, useSuggestForwarding } from '../../hooks/useAssignments.js'
import { useOfficials } from '../../hooks/useOfficials.js'
import AdviceTally from '../../components/assignment/AdviceTally.jsx'
import SuggestForm from '../../components/assignment/SuggestForm.jsx'
import ValidationBar from '../../components/ui/ValidationBar.jsx'
import Spinner from '../../components/ui/Spinner.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

/**
 * Where a union admin advises on one above-union report.
 *
 * The screen shows the two things the decision will be weighed against — what the
 * REPORTER asked for and what the other admins ADVISE — before it offers the form,
 * because advice given without seeing the rest is just a first guess repeated.
 */
export default function ForwardingPanel() {
  const { t } = useTranslation()
  const { id } = useParams()
  const navigate = useNavigate()
  const { data: item, isLoading, isError, refetch } = useMyForwardingItem(id)
  const { data: officials } = useOfficials()
  const suggest = useSuggestForwarding()

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  if (isError) return <ErrorState message={t('forwarding.admin.error')} onRetry={refetch} />
  if (!item) return <ErrorState message={t('forwarding.notFound')} onRetry={refetch} />

  const pointed = officials?.find((o) => o.id === item.pointedOfficialId)

  async function handleSubmit(payload) {
    await suggest.mutateAsync({ problemId: item.problemId, ...payload })
    navigate('/admin/forwarding')
  }

  return (
    <div className="flex flex-col gap-4">
      <Link to="/admin/forwarding" className="text-meta font-medium text-brand">
        {t('forwarding.back')}
      </Link>

      <div className="flex flex-col gap-2">
        <h1 className="text-h1 font-semibold text-ink">{item.title}</h1>
        <p className="text-meta text-muted">{item.address}</p>
        <ValidationBar count={item.validCount} threshold={item.validationThreshold} className="mt-1" />
        {/* The public record is one click away: the title and address alone are not
            enough to advise on, and the report itself is already public. */}
        <Link to={`/problems/${item.problemId}`} className="text-meta font-medium text-brand">
          {t('forwarding.viewReport')}
        </Link>
      </div>

      <section className="flex flex-col gap-2 rounded-card border border-hairline bg-surface p-4">
        <h2 className="font-semibold text-ink">{t('forwarding.pointed')}</h2>
        <p className="text-ink">
          {pointed ? (
            <>
              {pointed.name}{' '}
              <span className="text-muted">({t(`problem.tier.${pointed.tier}`)})</span>
            </>
          ) : (
            <span className="text-muted">{t('forwarding.pointedUnknown')}</span>
          )}
        </p>
      </section>

      <section className="flex flex-col gap-2">
        <h2 className="font-semibold text-ink">{t('forwarding.tallyTitle')}</h2>
        <AdviceTally suggestions={item.suggestions} topOfficialId={item.topOfficialId} />
      </section>

      <SuggestForm
        mySuggestion={item.mySuggestion}
        onSubmit={handleSubmit}
        isSubmitting={suggest.isPending}
        error={suggest.isError ? t('forwarding.advise.failed') : ''}
      />
    </div>
  )
}
