import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

/**
 * A full-screen viewer for one photo.
 *
 * Why this is not `Modal.jsx`: Modal is a titled content dialog — a padded
 * `max-w-120` panel with a hardcoded `id="modal-title"` — and it has no scroll
 * lock, no focus restore and no backdrop close. A lightbox is full-bleed and
 * untitled, and the photo is the only thing in it. Teaching Modal to be both
 * would put its one existing consumer (ReporterActions) at risk for no gain, so
 * this reuses the tokens and the Escape pattern rather than the component.
 *
 * The image is `object-contain`, never `object-cover`: everywhere else in the
 * app a photo is a cropped preview in a fixed frame, and this is the one place
 * it must be shown whole. Undoing that crop is the entire reason to open it.
 *
 * @param {{open: boolean, src?: string, alt?: string, onClose: () => void}} props
 */
export default function ImageViewer({ open, src, alt = '', onClose }) {
  const { t } = useTranslation()
  const closeRef = useRef(null)
  const returnFocusRef = useRef(null)

  useEffect(() => {
    if (!open) return

    // Remember what opened us, so closing hands focus back to the photo the
    // reader clicked rather than dumping them at the top of the document.
    returnFocusRef.current = document.activeElement
    closeRef.current?.focus()

    const handleKey = (e) => {
      if (e.key === 'Escape') onClose?.()
    }
    window.addEventListener('keydown', handleKey)

    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    return () => {
      window.removeEventListener('keydown', handleKey)
      document.body.style.overflow = previousOverflow
      returnFocusRef.current?.focus?.()
    }
  }, [open, onClose])

  if (!open || !src) return null

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t('problem.detail.imageViewer')}
      onClick={onClose}
      className="fixed inset-0 z-30 flex animate-fade-in items-center justify-center bg-ink/80 p-4"
    >
      <button
        ref={closeRef}
        type="button"
        onClick={onClose}
        aria-label={t('common.close')}
        className="absolute right-4 top-4 min-h-11 min-w-11 rounded-control bg-surface/90 text-h1 text-ink hover:bg-surface"
      >
        ×
      </button>

      {/* The click that closes the viewer is the backdrop's; the photo itself
          must not swallow it or a reader who taps the photo gets nothing. */}
      <img
        src={src}
        alt={alt}
        onClick={(e) => e.stopPropagation()}
        className="max-h-[90vh] max-w-[90vw] rounded-card object-contain"
      />
    </div>
  )
}
