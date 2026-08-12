import { useTranslation } from 'react-i18next'
import Button from './Button.jsx'

export default function ErrorState({ message, onRetry }) {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col items-center gap-3 rounded-card border border-hairline bg-surface p-8 text-center animate-fade-in">
      <p className="text-ink">{message}</p>
      {onRetry && (
        <Button variant="secondary" onClick={onRetry}>
          {t('common.retry')}
        </Button>
      )}
    </div>
  )
}
