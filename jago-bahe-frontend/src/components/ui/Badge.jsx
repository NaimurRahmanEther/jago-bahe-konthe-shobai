const VARIANTS = {
  brand: 'bg-brand-tint text-brand-dark',
  neutral: 'bg-reported-bg text-reported-fg',
}

export default function Badge({ variant = 'neutral', children, className = '' }) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-sm font-medium ${VARIANTS[variant]} ${className}`}
    >
      {children}
    </span>
  )
}
