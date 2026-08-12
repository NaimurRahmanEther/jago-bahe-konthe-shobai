import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'
import { useSuggestions } from '../../hooks/useSuggestions.js'

/**
 * Collects a case plan. In the default `create` mode it is the first plan; in
 * `revise` mode it restarts a stalled case — prefilled from the current plan and
 * carrying a required `reason` (the public "what changed"), which the payload adds.
 *
 * @param {{
 *   problemId: string,
 *   onSubmit: (payload: {strategy: string, tasks: string[], obstacles: string, suggestionResponse: string, reason?: string}) => Promise<unknown>,
 *   isSubmitting: boolean,
 *   mode?: 'create' | 'revise',
 *   initial?: {strategy?: string, tasks?: {task: string}[], obstacles?: string, suggestionResponse?: string} | null,
 * }} props
 */
export default function PlanForm({ problemId, onSubmit, isSubmitting, mode = 'create', initial = null }) {
  const { t } = useTranslation()
  const { data: suggestions } = useSuggestions(problemId)
  const topSuggestion = suggestions?.find((s) => s.isTop) ?? null
  const isRevise = mode === 'revise'

  const [strategy, setStrategy] = useState(initial?.strategy ?? '')
  // The plan is a week-by-week checklist: one task per row. In revise mode it is
  // seeded from the current plan; otherwise it starts with one empty week.
  const [tasks, setTasks] = useState(() =>
    initial?.tasks?.length ? initial.tasks.map((task) => task.task) : [''],
  )
  const [obstacles, setObstacles] = useState(initial?.obstacles ?? '')
  const [suggestionResponse, setSuggestionResponse] = useState(initial?.suggestionResponse ?? '')
  const [reason, setReason] = useState('')
  const [errors, setErrors] = useState({})

  function setTaskAt(i, value) {
    setTasks((prev) => prev.map((task, idx) => (idx === i ? value : task)))
  }
  function addWeek() {
    setTasks((prev) => [...prev, ''])
  }
  function removeWeek(i) {
    setTasks((prev) => (prev.length === 1 ? prev : prev.filter((_, idx) => idx !== i)))
  }

  function validate(cleanedTasks) {
    const next = {}
    if (!strategy.trim()) next.strategy = t('auth.errors.required')
    if (!suggestionResponse.trim()) next.suggestionResponse = t('resolution.errors.suggestionResponseRequired')
    if (cleanedTasks.length === 0) next.tasks = t('resolution.errors.tasksRequired')
    if (isRevise && !reason.trim()) next.reason = t('auth.errors.required')
    setErrors(next)
    return Object.keys(next).length === 0
  }

  function handleSubmit(e) {
    e.preventDefault()
    const cleanedTasks = tasks.map((task) => task.trim()).filter(Boolean)
    if (!validate(cleanedTasks)) return
    onSubmit({
      strategy: strategy.trim(),
      tasks: cleanedTasks,
      obstacles: obstacles.trim(),
      suggestionResponse: suggestionResponse.trim(),
      ...(isRevise ? { reason: reason.trim() } : {}),
    })
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 rounded-card border border-hairline bg-surface p-4">
      <div>
        <p className="text-sm font-semibold text-ink">
          {isRevise ? t('resolution.replan.title') : t('resolution.plan.title')}
        </p>
        {isRevise && <p className="mt-1 text-meta text-muted">{t('resolution.replan.hint')}</p>}
      </div>

      {isRevise && (
        <div className="flex flex-col gap-1">
          <label htmlFor="replan-reason" className="font-medium text-ink">
            {t('resolution.replan.reason')}{' '}
            <span className="font-normal text-brand-dark">({t('assignment.required')})</span>
          </label>
          <textarea
            id="replan-reason"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={2}
            className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
          />
          {errors.reason && <p className="text-sm text-reopened-fg">{errors.reason}</p>}
        </div>
      )}

      {topSuggestion && (
        <div className="rounded-card bg-brand-tint p-3">
          <span className="inline-flex rounded-full bg-surface px-2 py-0.5 text-[11px] font-bold text-brand-dark">
            {t('suggestion.top')}
          </span>
          <p className="mt-1 text-sm text-ink">{topSuggestion.text}</p>
        </div>
      )}

      <div className="flex flex-col gap-1">
        <label htmlFor="suggestionResponse" className="font-medium text-ink">
          {t('resolution.plan.suggestionResponse')}{' '}
          <span className="font-normal text-brand-dark">({t('assignment.required')})</span>
        </label>
        <textarea
          id="suggestionResponse"
          value={suggestionResponse}
          onChange={(e) => setSuggestionResponse(e.target.value)}
          rows={3}
          className={`w-full rounded-control border bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand ${
            topSuggestion ? 'border-brand' : 'border-hairline'
          }`}
        />
        {errors.suggestionResponse && <p className="text-sm text-reopened-fg">{errors.suggestionResponse}</p>}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="strategy" className="font-medium text-ink">
          {t('resolution.plan.strategy')}
        </label>
        <textarea
          id="strategy"
          value={strategy}
          onChange={(e) => setStrategy(e.target.value)}
          rows={3}
          className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
        />
        {errors.strategy && <p className="text-sm text-reopened-fg">{errors.strategy}</p>}
      </div>

      <div className="flex flex-col gap-2">
        <span className="font-medium text-ink">{t('resolution.plan.weeklyTasks')}</span>
        {tasks.map((task, i) => (
          <div key={i} className="flex items-center gap-2">
            <label htmlFor={`task-${i}`} className="w-20 shrink-0 text-meta text-muted">
              {t('resolution.plan.week', { count: i + 1 })}
            </label>
            <input
              id={`task-${i}`}
              type="text"
              value={task}
              onChange={(e) => setTaskAt(i, e.target.value)}
              placeholder={t('resolution.plan.taskPlaceholder')}
              className="min-h-11 w-full rounded-control border border-hairline bg-surface px-3 text-ink focus-visible:outline-2 focus-visible:outline-brand"
            />
            {tasks.length > 1 && (
              <button
                type="button"
                onClick={() => removeWeek(i)}
                aria-label={t('resolution.plan.removeWeek', { count: i + 1 })}
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-control border border-hairline text-muted hover:text-ink"
              >
                ✕
              </button>
            )}
          </div>
        ))}
        {errors.tasks && <p className="text-sm text-reopened-fg">{errors.tasks}</p>}
        <Button type="button" variant="secondary" onClick={addWeek} className="self-start">
          {t('resolution.plan.addWeek')}
        </Button>
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="obstacles" className="font-medium text-ink">
          {t('resolution.plan.obstacles')}
        </label>
        <textarea
          id="obstacles"
          value={obstacles}
          onChange={(e) => setObstacles(e.target.value)}
          rows={2}
          className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
        />
      </div>

      <Button type="submit" disabled={isSubmitting} className="self-start">
        {isRevise ? t('resolution.replan.submit') : t('resolution.plan.submit')}
      </Button>
    </form>
  )
}
