import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

export default function Modal({ open, onClose, title, children }) {
  const { t } = useTranslation()

  useEffect(() => {
    if (!open) return
    const handleKey = (e) => {
      if (e.key === 'Escape') onClose?.()
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-20 flex items-center justify-center bg-ink/40 p-4 animate-fade-in">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? 'modal-title' : undefined}
        className="w-full max-w-120 rounded-card bg-surface p-6 shadow-lg animate-rise-in"
      >
        <div className="flex items-start justify-between gap-4">
          {title && (
            <h2 id="modal-title" className="text-h2 font-semibold text-ink">
              {title}
            </h2>
          )}
          <button
            type="button"
            onClick={onClose}
            aria-label={t('common.close')}
            className="min-h-11 min-w-11 rounded-control text-muted hover:text-ink"
          >
            ×
          </button>
        </div>
        <div className="mt-4">{children}</div>
      </div>
    </div>
  )
}
