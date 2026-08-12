import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import { useAuth } from '../../auth/useAuth.js'

/** @param {{onSubmit: (text: string) => Promise<unknown>}} props */
export default function SuggestionForm({ onSubmit }) {
  const { t } = useTranslation()
  const { role } = useAuth()
  const [text, setText] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  if (role !== 'resident') {
    return <p className="text-sm text-muted">{t('suggestion.residentOnly')}</p>
  }

  async function handleSubmit(e) {
    e.preventDefault()
    if (!text.trim()) {
      setError(t('auth.errors.required'))
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await onSubmit(text.trim())
      setText('')
    } catch {
      setError(t('suggestion.submitFailed'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-2">
      <label htmlFor="suggestion-text" className="font-medium text-ink">
        {t('suggestion.formLabel')}
      </label>
      <textarea
        id="suggestion-text"
        value={text}
        onChange={(e) => setText(e.target.value)}
        rows={2}
        className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
      />
      {error && <p className="text-sm text-reopened-fg">{error}</p>}
      <Button type="submit" disabled={isSubmitting} className="self-start">
        {t('suggestion.submit')}
      </Button>
    </form>
  )
}
