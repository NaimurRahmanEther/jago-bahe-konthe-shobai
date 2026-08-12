import { useId } from 'react'

export default function Input({ label, id, error, className = '', ...props }) {
  const generatedId = useId()
  const inputId = id ?? generatedId
  const errorId = error ? `${inputId}-error` : undefined

  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={inputId} className="font-medium text-ink">
        {label}
      </label>
      <input
        id={inputId}
        aria-invalid={Boolean(error)}
        aria-describedby={errorId}
        className={`min-h-11 w-full rounded-control border bg-surface px-3 text-ink focus-visible:outline-2 focus-visible:outline-brand ${
          error ? 'border-reopened-fg' : 'border-hairline'
        } ${className}`}
        {...props}
      />
      {error && (
        <p id={errorId} className="text-sm text-reopened-fg">
          {error}
        </p>
      )}
    </div>
  )
}
