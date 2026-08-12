import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import FilterChip from './FilterChip.jsx'
import Input from './Input.jsx'
import { presetsFor, presetRange, todayValue } from '../../lib/date.js'

/**
 * Narrow a list to a span of dates: four Bangla preset chips, plus two date
 * fields for an exact range.
 *
 * The presets carry the weight, and that is deliberate. A native
 * `<input type="date">` draws its own picker in the BROWSER's locale — Latin
 * digits, often English month names — and none of that is stylable or
 * translatable. The alternative is a hand-built calendar popover, which the
 * design rules rule out ("no component kit", A.6) and which would be a large
 * accessible-widget build for a filter. So the common cases are one Bangla tap,
 * and the two fields are the escape hatch for "which reports came in on the
 * 3rd" — the one question presets cannot answer. Everything the app itself
 * draws around them is Bangla.
 *
 * There is one source of truth: a preset fills both fields, and typing in
 * either drops the preset to `custom`, so the chips can never claim a range the
 * fields contradict.
 *
 * `direction` decides which way the presets point and whether tomorrow is
 * selectable. A list of things that HAPPENED is capped at today — nothing was
 * reported tomorrow, and offering the date only invites an empty result. A list
 * of DEADLINES is not: most of them are ahead of you, and an official asking
 * what falls due next week is the main thing that filter is for.
 *
 * @param {{
 *   value: {preset: string, from: string, to: string},
 *   onChange: (next: {preset: string, from: string, to: string}) => void,
 *   label: string,
 *   direction?: import('../../lib/date.js').DateDirection,
 * }} props
 */
export default function DateRangeFilter({ value, onChange, label, direction = 'past' }) {
  const { t } = useTranslation()
  const groupId = useId()
  const presets = presetsFor(direction)
  const latest = direction === 'future' ? undefined : todayValue()

  function choosePreset(preset) {
    onChange({ preset, ...presetRange(preset) })
  }

  function setBound(key) {
    return (e) => onChange({ ...value, preset: 'custom', [key]: e.target.value })
  }

  return (
    <section className="flex flex-col gap-2" aria-labelledby={groupId}>
      <h2 id={groupId} className="text-micro font-semibold uppercase tracking-wider text-muted">
        {label}
      </h2>

      <div className="flex flex-wrap gap-2" role="group" aria-labelledby={groupId}>
        {presets.map((p) => (
          <FilterChip key={p} active={value.preset === p} onClick={() => choosePreset(p)}>
            {t(`filter.date.${p}`)}
          </FilterChip>
        ))}
      </div>

      {/* The exact range is always visible rather than hidden behind a toggle:
          a disclosure would cost the one interaction this control exists for an
          extra tap, and two fields are not clutter worth hiding. */}
      <div className="flex flex-wrap items-end gap-3">
        <div className="min-w-[9.5rem] flex-1">
          <Input
            label={t('filter.date.from')}
            type="date"
            value={value.from}
            max={value.to || latest}
            onChange={setBound('from')}
          />
        </div>
        <div className="min-w-[9.5rem] flex-1">
          <Input
            label={t('filter.date.to')}
            type="date"
            value={value.to}
            min={value.from || undefined}
            max={latest}
            onChange={setBound('to')}
          />
        </div>
      </div>
    </section>
  )
}
