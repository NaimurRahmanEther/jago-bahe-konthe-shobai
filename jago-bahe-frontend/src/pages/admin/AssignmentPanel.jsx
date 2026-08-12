import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueueItem, useAssignWithinUnion } from '../../hooks/useAssignments.js'
import { useOfficials } from '../../hooks/useOfficials.js'
import AssignForm from '../../components/assignment/AssignForm.jsx'
import ValidationBar from '../../components/ui/ValidationBar.jsx'
import Spinner from '../../components/ui/Spinner.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

export default function AssignmentPanel() {
  const { t } = useTranslation()
  const { id } = useParams()
  const navigate = useNavigate()
  // The whole queue, not the Validated-only view: since B17 a Reported problem is
  // forwardable, and this is the screen that forwards it.
  const { data: item, isLoading, isError, refetch } = useQueueItem(id)
  const { data: officials } = useOfficials()
  const assignWithinUnion = useAssignWithinUnion()

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  if (isError) return <ErrorState message={t('assignment.queue.error')} onRetry={refetch} />
  if (!item) return <ErrorState message={t('assignment.panel.notFound')} onRetry={refetch} />

  const publicChoice = officials?.find((o) => o.id === item.pointedOfficialId)

  async function handleConfirm(payload) {
    await assignWithinUnion.mutateAsync({ problemId: item.problemId, ...payload })
    navigate('/admin')
  }

  return (
    <div className="flex flex-col gap-4">
      <Link to="/admin" className="text-meta font-medium text-brand">
        {t('assignment.panel.back')}
      </Link>

      <div className="flex flex-col gap-2">
        <h1 className="text-h1 font-semibold text-ink">{item.title}</h1>
        <p className="text-meta text-muted">{item.address}</p>
        {/* The evidence the decision rests on. An admin forwarding a report below V
            is making a judgment call (A.3.1), and they should be looking at the
            count while they make it — this screen used to show the title alone. */}
        <ValidationBar count={item.validCount} threshold={item.validationThreshold} className="mt-1" />
      </div>

      <AssignForm
        publicChoice={publicChoice}
        onConfirm={handleConfirm}
        isSubmitting={assignWithinUnion.isPending}
      />
    </div>
  )
}
