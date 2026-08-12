import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'

/** @param {{onConfirm: (outcome: 'solved' | 'not_solved') => Promise<unknown>, isSubmitting: boolean}} props */
export default function ConfirmBox({ onConfirm, isSubmitting }) {
  const { t } = useTranslation()
  const [answered, setAnswered] = useState(false)

  if (answered) {
    return <p className="text-sm text-validated-fg">{t('confirm.thanks')}</p>
  }

  return (
    <div className="rounded-card border border-hairline bg-surface p-4">
      <p className="text-sm font-medium text-ink">{t('confirm.prompt')}</p>
      <div className="mt-3 flex gap-3">
        <Button className="min-h-13 flex-1" onClick={() => onConfirm('solved').then(() => setAnswered(true))} disabled={isSubmitting}>
          {t('confirm.solved')}
        </Button>
        <Button
          variant="secondary"
          className="min-h-13 flex-1"
          onClick={() => onConfirm('not_solved').then(() => setAnswered(true))}
          disabled={isSubmitting}
        >
          {t('confirm.notSolved')}
        </Button>
      </div>
    </div>
  )
}
