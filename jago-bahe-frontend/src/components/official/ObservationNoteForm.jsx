import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'

/**
 * The monitor's one recorded action: what they did about a silence.
 *
 * It is the only control on the observation page. There is deliberately nothing
 * here that reassigns or closes the case — silence raises visibility, not
 * responsibility (Concept §7), the same reason the super admin's oversight page
 * offers no buttons at all.
 *
 * @param {{
 *   observationId: string,
 *   onSubmit: (payload: {observationId: string, text: string}) => void,
 *   isSubmitting: boolean,
 *   isError: boolean,
 * }} props
 */
export default function ObservationNoteForm({ observationId, onSubmit, isSubmitting, isError }) {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  const [error, setError] = useState('')

  function handleSubmit(e) {
    e.preventDefault()
    if (!text.trim()) {
      setError(t('auth.errors.required'))
      return
    }
    setError('')
    onSubmit({ observationId, text: text.trim() })
    setText('')
  }

  const fieldId = `observation-note-${observationId}`

  return (
    <form onSubmit={handleSubmit} className="mt-3 flex flex-col gap-2 border-t border-hairline pt-3">
      <label htmlFor={fieldId} className="text-meta font-medium text-ink">
        {t('observation.note.label')}
      </label>
      <textarea
        id={fieldId}
        rows={2}
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder={t('observation.note.placeholder')}
        className="w-full rounded-control border border-hairline bg-surface p-3 text-ink focus-visible:outline-2 focus-visible:outline-brand"
      />
      <p className="text-micro text-muted">{t('observation.note.publicHint')}</p>
      {error && <p className="text-meta text-reopened-fg">{error}</p>}
      {isError && <p className="text-meta text-reopened-fg">{t('observation.note.error')}</p>}
      <div>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? t('observation.note.submitting') : t('observation.note.submit')}
        </Button>
      </div>
    </form>
  )
}
