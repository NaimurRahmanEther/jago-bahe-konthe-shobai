import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'

/**
 * @param {{
 *   updates: import('../../lib/types/models.js').ProgressUpdate[],
 *   onPost: (text: string) => Promise<unknown>,
 *   isSubmitting: boolean,
 * }} props
 */
export default function UpdateTimeline({ updates, onPost, isSubmitting }) {
  const { t } = useTranslation()
  const [text, setText] = useState('')

  function handleSubmit(e) {
    e.preventDefault()
    if (!text.trim()) return
    onPost(text.trim())
    setText('')
  }

  return (
    <div className="flex flex-col gap-3">
      {updates.length === 0 ? (
        <p className="text-sm text-muted">{t('resolution.timeline.empty')}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {updates.map((u) => (
            <li
              key={u.id}
              className={`rounded-card border p-3 text-sm ${
                u.kind === 'obstacle' ? 'border-transparent bg-blocked-bg text-blocked-fg' : 'border-hairline bg-surface text-ink'
              }`}
            >
              <p className="text-xs font-semibold uppercase tracking-wide">
                {u.kind === 'obstacle' ? t('resolution.timeline.obstacleLabel') : t('resolution.timeline.progressLabel')}
              </p>
              <p className="mt-1">{u.text}</p>
            </li>
          ))}
        </ul>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-2">
        <label htmlFor="update-text" className="font-medium text-ink">
          {t('resolution.timeline.formLabel')}
        </label>
        <textarea
          id="update-text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          rows={2}
          className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
        />
        <Button type="submit" disabled={isSubmitting} className="self-start">
          {t('resolution.timeline.post')}
        </Button>
      </form>
    </div>
  )
}
