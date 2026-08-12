import { useId } from 'react'

/**
 * A labelled select — Input.jsx's twin for a choice instead of free text.
 *
 * The label is always visible above the field, never a placeholder: literacy
 * varies and a vanishing label leaves nothing to read (Design Guideline A.6).
 */
export default function Select({ label, id, error, className = '', children, ...props }) {
  const generatedId = useId()
  const selectId = id ?? generatedId
  const errorId = error ? `${selectId}-error` : undefined

  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={selectId} className="font-medium text-ink">
        {label}
      </label>
      <select
        id={selectId}
        aria-invalid={Boolean(error)}
        aria-describedby={errorId}
        className={`min-h-11 w-full rounded-control border bg-surface px-3 text-ink focus-visible:outline-2 focus-visible:outline-brand ${
          error ? 'border-reopened-fg' : 'border-hairline'
        } ${className}`}
        {...props}
      >
        {children}
      </select>
      {error && (
        <p id={errorId} className="text-sm text-reopened-fg">
          {error}
        </p>
      )}
    </div>
  )
}
