import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useReportProblem } from '../../hooks/useProblems.js'
import Input from '../../components/ui/Input.jsx'
import Button from '../../components/ui/Button.jsx'
import Card from '../../components/ui/Card.jsx'
import ImagePicker from '../../components/ui/ImagePicker.jsx'
import LocationPicker from '../../components/problem/LocationPicker.jsx'
import OfficialPicker from '../../components/problem/OfficialPicker.jsx'

/**
 * Map a failed submit to something the reporter can act on. The server's own
 * message is English and this app is Bangla-only (A.5.3), so the status — not the
 * message — chooses the copy.
 */
function submitErrorKey(status) {
  if (status === 401) return 'problem.report.errors.signedOut'
  if (status === 413) return 'problem.report.errors.tooLarge'
  if (status === 429) return 'problem.report.errors.tooMany'
  if (status === 0) return 'problem.report.errors.offline'
  return 'problem.report.errors.submitFailed'
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
      {error && <p className="text-sm text-reopened-fg">{error}</p>}
    </div>
  )
}

export default function ReportProblem() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const reportProblem = useReportProblem()

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [location, setLocation] = useState({ areaId: '', address: '' })
  const [officialId, setOfficialId] = useState('')
  const [proposedSolution, setProposedSolution] = useState('')
  const [imageUrl, setImageUrl] = useState('')
  const [errors, setErrors] = useState({})
  const [submitError, setSubmitError] = useState('')

  function validate() {
    const next = {}
    if (!title.trim()) next.title = t('auth.errors.required')
    if (!description.trim()) next.description = t('auth.errors.required')
    if (!location.areaId) next.location = t('problem.report.errors.locationRequired')
    if (!officialId) next.official = t('problem.report.errors.officialRequired')
    setErrors(next)
    return Object.keys(next).length === 0
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setSubmitError('')
    if (!validate()) return

    try {
      const problem = await reportProblem.mutateAsync({
        title: title.trim(),
        description: description.trim(),
        location,
        pointedOfficialId: officialId,
        proposedSolution: proposedSolution.trim() || undefined,
        imageUrl: imageUrl || undefined,
      })
      navigate(`/problems/${problem.id}`)
    } catch (err) {
      // A bare `catch {}` used to answer every failure with one sentence, so a
      // rejected photo, an expired session and a dead backend were indistinguishable
      // — the reporter was told to "try again" at the one thing that could not work.
      // The client interceptor normalizes every rejection to { status, message }.
      setSubmitError(t(submitErrorKey(err?.status)))
    }
  }

  return (
    <Card className="mx-auto max-w-160">
      <h1 className="text-h1 font-semibold text-ink">{t('problem.report.title')}</h1>
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

        <LocationPicker value={location} onChange={setLocation} />
        {errors.location && <p className="-mt-3 text-sm text-reopened-fg">{errors.location}</p>}

        <OfficialPicker value={officialId} onChange={setOfficialId} />
        {errors.official && <p className="-mt-3 text-sm text-reopened-fg">{errors.official}</p>}

        <ImagePicker
          id="problem-photo"
          label={t('problem.report.image')}
          hint={t('problem.report.optional')}
          value={imageUrl}
          onChange={setImageUrl}
        />

        <Textarea
          id="proposedSolution"
          label={t('problem.report.proposedSolution')}
          hint={t('problem.report.optional')}
          value={proposedSolution}
          onChange={(e) => setProposedSolution(e.target.value)}
          rows={3}
        />

        {submitError && <p className="text-sm text-reopened-fg">{submitError}</p>}

        <Button type="submit" disabled={reportProblem.isPending} className="w-full">
          {t('problem.report.submit')}
        </Button>
      </form>
    </Card>
  )
}
