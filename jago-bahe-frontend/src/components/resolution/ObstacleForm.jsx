import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import Button from '../ui/Button.jsx'

/** The obstacle categories, in display order (Concept §8). */
const CATEGORIES = ['budget', 'legal_authority', 'higher_tier', 'land_dispute', 'inter_department', 'technical']

/**
 * @param {{
 *   onSubmit: (payload: {category: string, whatBlocks: string, whoUnblocks: string, proofTried: string}) => Promise<unknown>,
 *   isSubmitting: boolean,
 * }} props
 */
export default function ObstacleForm({ onSubmit, isSubmitting }) {
  const { t } = useTranslation()
  const [category, setCategory] = useState('')
  const [whatBlocks, setWhatBlocks] = useState('')
  const [whoUnblocks, setWhoUnblocks] = useState('')
  const [proofTried, setProofTried] = useState('')
  const [errors, setErrors] = useState({})

  function validate() {
    const next = {}
    if (!category) next.category = t('auth.errors.required')
    if (!whatBlocks.trim()) next.whatBlocks = t('auth.errors.required')
    if (!whoUnblocks.trim()) next.whoUnblocks = t('auth.errors.required')
    if (!proofTried.trim()) next.proofTried = t('auth.errors.required')
    setErrors(next)
    return Object.keys(next).length === 0
  }

  function handleSubmit(e) {
    e.preventDefault()
    if (!validate()) return
    onSubmit({ category, whatBlocks: whatBlocks.trim(), whoUnblocks: whoUnblocks.trim(), proofTried: proofTried.trim() })
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 rounded-card border border-hairline bg-surface p-4">
      <div className="flex flex-col gap-1">
        <label htmlFor="blockCategory" className="font-medium text-ink">
          {t('obstacle.categoryLabel')}
        </label>
        <select
          id="blockCategory"
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          className="min-h-11 w-full rounded-control border border-hairline bg-surface px-3 text-ink focus-visible:outline-2 focus-visible:outline-brand"
        >
          <option value="" disabled>
            {t('obstacle.categoryPlaceholder')}
          </option>
          {CATEGORIES.map((c) => (
            <option key={c} value={c}>
              {t(`obstacle.categories.${c}`)}
            </option>
          ))}
        </select>
        {errors.category && <p className="text-sm text-reopened-fg">{errors.category}</p>}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="whatBlocks" className="font-medium text-ink">
          {t('resolution.obstacle.whatBlocks')}
        </label>
        <textarea
          id="whatBlocks"
          value={whatBlocks}
          onChange={(e) => setWhatBlocks(e.target.value)}
          rows={2}
          className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
        />
        {errors.whatBlocks && <p className="text-sm text-reopened-fg">{errors.whatBlocks}</p>}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="whoUnblocks" className="font-medium text-ink">
          {t('resolution.obstacle.whoUnblocks')}
        </label>
        <input
          id="whoUnblocks"
          type="text"
          value={whoUnblocks}
          onChange={(e) => setWhoUnblocks(e.target.value)}
          className="min-h-11 w-full rounded-control border border-hairline bg-surface px-3 text-ink"
        />
        {errors.whoUnblocks && <p className="text-sm text-reopened-fg">{errors.whoUnblocks}</p>}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="proofTried" className="font-medium text-ink">
          {t('resolution.obstacle.proofTried')}
        </label>
        <textarea
          id="proofTried"
          value={proofTried}
          onChange={(e) => setProofTried(e.target.value)}
          rows={2}
          className="w-full rounded-control border border-hairline bg-surface px-3 py-2 text-ink focus-visible:outline-2 focus-visible:outline-brand"
        />
        {errors.proofTried && <p className="text-sm text-reopened-fg">{errors.proofTried}</p>}
      </div>

      <Button type="submit" disabled={isSubmitting} className="self-start">
        {t('resolution.obstacle.submit')}
      </Button>
    </form>
  )
}
