import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import ImagePicker from '../ui/ImagePicker.jsx'

/**
 * @param {{
 *   onSubmit: (payload: {beforeImageUrl: string, afterImageUrl: string}) => Promise<unknown>,
 *   isSubmitting: boolean,
 * }} props
 */
export default function EvidenceUpload({ onSubmit, isSubmitting }) {
  const { t } = useTranslation()
  const [before, setBefore] = useState('')
  const [after, setAfter] = useState('')
  const [error, setError] = useState('')

  function handleSubmit(e) {
    e.preventDefault()
    if (!before || !after) {
      setError(t('resolution.evidence.bothRequired'))
      return
    }
    setError('')
    onSubmit({ beforeImageUrl: before, afterImageUrl: after })
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <div className="flex flex-col gap-3 sm:flex-row">
        <div className="flex-1">
          <ImagePicker id="before-photo" label={t('resolution.evidence.before')} value={before} onChange={setBefore} />
        </div>
        <div className="flex-1">
          <ImagePicker id="after-photo" label={t('resolution.evidence.after')} value={after} onChange={setAfter} />
        </div>
      </div>

      {error && <p className="text-sm text-reopened-fg">{error}</p>}

      <Button type="submit" disabled={isSubmitting} className="self-start">
        {t('resolution.evidence.submit')}
      </Button>
    </form>
  )
}
