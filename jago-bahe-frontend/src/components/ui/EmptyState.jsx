export default function EmptyState({ title, description, action }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-card border border-hairline bg-surface p-8 text-center animate-fade-in">
      <p className="font-semibold text-ink">{title}</p>
      {description && <p className="text-muted">{description}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}
