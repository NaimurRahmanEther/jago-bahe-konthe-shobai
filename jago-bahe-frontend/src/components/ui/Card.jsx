export default function Card({ interactive = false, className = '', children, ...props }) {
  const interactiveClasses = interactive
    ? 'transition-shadow duration-150 ease-standard hover:shadow-md hover:border-brand/40'
    : ''
  return (
    <div
      className={`rounded-card border border-hairline bg-surface p-4 ${interactiveClasses} ${className}`}
      {...props}
    >
      {children}
    </div>
  )
}
