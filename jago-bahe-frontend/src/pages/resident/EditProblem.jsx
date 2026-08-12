import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useProblem, useUpdateProblem } from '../../hooks/useProblems.js'
import { useAreas } from '../../hooks/useAreas.js'
import { useAuth } from '../../auth/useAuth.js'
import Input from '../../components/ui/Input.jsx'
import Button from '../../components/ui/Button.jsx'
import Card from '../../components/ui/Card.jsx'
import Spinner from '../../components/ui/Spinner.jsx'
import ErrorState from '../../components/ui/ErrorState.jsx'

// These mirror the backend's Editable() window; the server is the authority, this
// only decides whether to show the form or an explanation (A.5.7).
const isEditable = (p) => p.status === 'Reported' && p.validCount === 0

// The server's message is English and the app is Bangla-only, so the status —
// not the body — chooses the copy. A 409 means the window closed under us.
function submitErrorKey(status) {
  if (status === 401) return 'problem.edit.errors.signedOut'
  if (status === 409) return 'problem.edit.errors.locked'
  return 'problem.edit.errors.submitFailed'
}

export default function EditProblem() {
  const { t } = useTranslation()
  const { id } = useParams()
  const { user } = useAuth()
  const { data: problem, isLoading, isError, refetch } = useProblem(id)

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }
  if (isError || !problem) {
    return <ErrorState message={t('problem.detail.error')} onRetry={refetch} />
  }

  // Ownership and the window are UX gates in front of a form the backend guards
  // anyway — but a reporter should never be shown a form that cannot succeed.
  if (problem.reporterId !== user?.id) {
    return <Notice id={id} message={t('problem.edit.notOwner')} />
  }
  if (!isEditable(problem)) {
    return <Notice id={id} message={t('problem.edit.locked')} />
  }

  // Remount per problem so the form initializes cleanly from loaded data.
  return <EditForm key={problem.id} problem={problem} />
}

function Notice({ id, message }) {
  const { t } = useTranslation()
  return (
    <Card className="mx-auto max-w-160">
      <p className="text-ink">{message}</p>
      <Link to={`/problems/${id}`} className="mt-3 inline-block text-meta font-medium text-brand">
        {t('problem.edit.backToReport')}
      </Link>
    </Card>
  )
}

function Textarea({ label, id, value, onChange, error, rows = 4, hint }) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="font-medium text-ink">
        {label} {hint && <span className="font-normal text-muted">({hint})</span>}
      </label>
      <textarea
        id={id}
        value={value}
        onChange={onChange}
        rows={rows}
        className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
      />
      {error && <p className="text-meta text-reopened-fg">{error}</p>}
    </div>
  )
}

/** @param {{problem: import('../../lib/types/models.js').Problem}} props */
function EditForm({ problem }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const updateProblem = useUpdateProblem(problem.id)
  // Every area, so the fixed one can be shown by name whatever its level.
  const { data: areas } = useAreas()

  const [title, setTitle] = useState(problem.title)
  const [description, setDescription] = useState(problem.description)
  const [address, setAddress] = useState(problem.location.address ?? '')
  const [proposedSolution, setProposedSolution] = useState(problem.proposedSolution ?? '')
  const [errors, setErrors] = useState({})
  const [submitError, setSubmitError] = useState('')

  const areaName = areas?.find((a) => a.id === problem.location.areaId)?.name

  function validate() {
    const next = {}
    if (!title.trim()) next.title = t('auth.errors.required')
    if (!description.trim()) next.description = t('auth.errors.required')
    setErrors(next)
    return Object.keys(next).length === 0
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setSubmitError('')
    if (!validate()) return
    try {
      await updateProblem.mutateAsync({
        title: title.trim(),
        description: description.trim(),
        // Area is fixed at filing; keep it (and any coordinates) and edit only the
        // address text. The backend ignores areaId even if sent.
        location: { ...problem.location, address: address.trim() },
        proposedSolution: proposedSolution.trim() || undefined,
      })
      navigate(`/problems/${problem.id}`)
    } catch (err) {
      setSubmitError(t(submitErrorKey(err?.status)))
    }
  }

  return (
    <Card className="mx-auto max-w-160">
      <Link to={`/problems/${problem.id}`} className="text-meta font-medium text-brand">
        {t('problem.edit.backToReport')}
      </Link>
      <h1 className="mt-2 text-h1 font-semibold text-ink">{t('problem.edit.title')}</h1>
      <form className="mt-4 flex flex-col gap-5" onSubmit={handleSubmit} noValidate>
        <Input
          label={t('problem.report.problemTitle')}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          error={errors.title}
        />

        <Textarea
          id="description"
          label={t('problem.report.description')}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          error={errors.description}
        />

        {/* Area is fixed at filing — shown, not editable. */}
        <div className="flex flex-col gap-1">
          <span className="font-medium text-ink">{t('problem.edit.area')}</span>
          <p className="text-meta text-muted">{areaName ?? problem.location.areaId}</p>
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="address" className="font-medium text-ink">
            {t('problem.report.address')}
          </label>
          <input
            id="address"
            type="text"
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            placeholder={t('problem.report.addressPlaceholder')}
            className="min-h-11 w-full rounded-control border border-hairline bg-surface px-3 text-ink focus-visible:outline-2 focus-visible:outline-brand"
          />
        </div>

        <Textarea
          id="proposedSolution"
          label={t('problem.report.proposedSolution')}
          hint={t('problem.report.optional')}
          value={proposedSolution}
          onChange={(e) => setProposedSolution(e.target.value)}
          rows={3}
        />

        {submitError && <p className="text-meta text-reopened-fg">{submitError}</p>}

        <Button type="submit" disabled={updateProblem.isPending} className="w-full">
          {t('problem.edit.save')}
        </Button>
      </form>
    </Card>
  )
}
