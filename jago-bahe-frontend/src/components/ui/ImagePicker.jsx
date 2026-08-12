import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from './Button.jsx'
import { compressImage } from '../../lib/image.js'

/**
 * Cap on the file a reporter may pick. This is not the size that gets posted:
 * compressImage downscales and re-encodes first, so a 5MB phone photo lands well
 * inside the server's 1 MiB request-body cap. Anything above this is refused up
 * front rather than spending seconds decoding a file we would reject anyway.
 */
const MAX_BYTES = 5 * 1024 * 1024

/** Inline, not an icon library — the spec forbids a component kit. */
function PhotoGlyph({ className = '' }) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <circle cx="8.5" cy="9.5" r="1.5" />
      <path d="m21 15-4.5-4.5L7 20" />
    </svg>
  )
}

/**
 * A labelled photo input in the shape of a post composer: an attach box until a
 * photo is chosen, then a small thumbnail with change/remove beside it.
 *
 * The thumbnail confirms *which* photo is attached; it is deliberately not a
 * crop preview of how the feed will render it, which cost the form more height
 * than an optional field is worth.
 *
 * Emits the chosen image as a compressed JPEG data URL, posted inline with the
 * report (there is no upload endpoint — the photo is part of the request body,
 * which is why it must be compressed to fit; see lib/image.js).
 * @param {{
 *   id: string,
 *   label: string,
 *   hint?: string,
 *   value: string,
 *   onChange: (dataUrl: string) => void,
 * }} props
 */
export default function ImagePicker({ id, label, hint, value, onChange }) {
  const { t } = useTranslation()
  const inputRef = useRef(null)
  const [error, setError] = useState('')
  const [dragging, setDragging] = useState(false)
  const [busy, setBusy] = useState(false)

  const labelId = `${id}-label`
  const hintId = `${id}-hint`
  const errorId = `${id}-error`

  async function acceptFile(file) {
    if (!file) return
    if (!file.type.startsWith('image/')) return
    if (file.size > MAX_BYTES) {
      setError(t('common.photoTooLarge'))
      return
    }
    setError('')
    setBusy(true)
    try {
      // Compressing a large photo takes a beat on a phone; the busy state is why
      // that beat does not read as a dead button.
      onChange(await compressImage(file))
    } catch {
      // Never emit on failure: a photo we could not shrink would fail the whole
      // report at the server with a message about bytes, on a form about a road.
      setError(t('common.photoFailed'))
    } finally {
      setBusy(false)
    }
  }

  async function handleFile(e) {
    const file = e.target.files?.[0]
    // Let the same file be re-picked after a remove: without this the input
    // holds the old value and selecting it again fires no change event.
    e.target.value = ''
    await acceptFile(file)
  }

  async function handleDrop(e) {
    e.preventDefault()
    setDragging(false)
    await acceptFile(e.dataTransfer.files?.[0])
  }

  function handleRemove() {
    setError('')
    onChange('')
  }

  return (
    <div className="flex flex-col gap-2">
      {/* A span, not a <label>: the attach box below is the <label htmlFor>, and
          a second one would concatenate into the input's accessible name. */}
      <span id={labelId} className="font-medium text-ink">
        {label} {hint && <span className="font-normal text-muted">({hint})</span>}
      </span>

      {/* sr-only, not hidden — it must stay focusable. The visible focus ring is
          handed to the next sibling via peer-focus-visible. */}
      <input
        ref={inputRef}
        id={id}
        type="file"
        accept="image/*"
        onChange={handleFile}
        aria-labelledby={labelId}
        aria-describedby={error ? errorId : hintId}
        className="peer sr-only"
      />

      {value ? (
        // The ring rides the row, not the thumbnail: peer-focus-visible is a
        // sibling selector, and only this wrapper is a sibling of the input.
        <div className="flex w-fit items-center gap-3 rounded-control peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-brand">
          <img
            src={value}
            alt={label}
            loading="lazy"
            className="h-28 w-28 shrink-0 rounded-control border border-hairline object-cover"
          />
          <div className="flex flex-col gap-2">
            <Button variant="secondary" onClick={() => inputRef.current?.click()}>
              {t('common.changePhoto')}
            </Button>
            <Button variant="ghost" onClick={handleRemove}>
              {t('common.removePhoto')}
            </Button>
          </div>
        </div>
      ) : (
        <label
          htmlFor={id}
          onDragOver={(e) => {
            e.preventDefault()
            setDragging(true)
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={handleDrop}
          className={`flex min-h-28 cursor-pointer flex-col items-center justify-center gap-1 rounded-card border px-4 py-3 text-center transition duration-150 ease-standard peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-brand ${
            dragging ? 'border-brand bg-brand-tint' : 'border-hairline bg-canvas hover:bg-brand-tint/40'
          }`}
        >
          <span className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-tint text-brand">
            <PhotoGlyph className="h-5 w-5" />
          </span>
          <span className="font-medium text-ink">
            {busy ? t('common.photoProcessing') : t('common.choosePhoto')}
          </span>
          {/* Dragging is a desktop gesture; don't promise it on a phone. */}
          <span className="hidden text-sm text-muted sm:block">{t('common.photoDragHint')}</span>
          <span id={hintId} className="text-sm text-muted">
            {t('common.photoHint')}
          </span>
        </label>
      )}

      {error && (
        <p id={errorId} className="text-sm text-reopened-fg">
          {error}
        </p>
      )}
    </div>
  )
}
