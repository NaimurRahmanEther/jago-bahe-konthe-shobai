import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/useAuth.js'

const HOW_STEPS = ['report', 'validate', 'assign', 'act', 'confirm', 'record']

// The elected ladder as it actually stands in Dhamoirhat: no minister holds this
// seat, and no pourashava councillors are in the directory, so neither is shown.
// Rungs read top-down, the way a problem climbs.
const LADDER = [
  ['mp'],
  ['upazila_chairman', 'upazila_vice_chairman'],
  ['union_chairman', 'pourashava_mayor'],
  ['women_member', 'ward_member'],
]

function HeroCtas({ t, canReport }) {
  return (
    <div className="flex flex-wrap gap-3">
      <Link
        to={canReport ? '/report' : '/problems'}
        className="inline-flex min-h-11 items-center rounded-control bg-brand px-6 font-medium text-white shadow-sm transition duration-150 ease-standard hover:bg-brand-dark active:scale-[0.98] motion-reduce:active:scale-100"
      >
        {canReport ? t('landing.hero.ctaReport') : t('landing.hero.ctaBrowse')}
      </Link>
      {canReport && (
        <Link
          to="/problems"
          className="inline-flex min-h-11 items-center rounded-control border border-hairline bg-surface px-6 font-medium text-ink transition duration-150 ease-standard hover:bg-canvas active:scale-[0.98] motion-reduce:active:scale-100"
        >
          {t('landing.hero.ctaBrowse')}
        </Link>
      )}
    </div>
  )
}

export default function Landing() {
  const { t } = useTranslation()
  const { role } = useAuth()
  const canReport = !role || role === 'resident'

  return (
    <div className="flex w-full flex-col gap-10">
      <section className="rounded-card border border-hairline bg-gradient-to-b from-brand-tint to-canvas p-6 shadow-sm animate-rise-in sm:p-10">
        <h1 className="text-[28px] font-semibold leading-snug text-brand-dark">{t('landing.hero.tagline')}</h1>
        <div className="mt-6">
          <HeroCtas t={t} canReport={canReport} />
        </div>
      </section>

      <section aria-labelledby="landing-how">
        <h2 id="landing-how" className="text-h1 font-semibold text-ink">
          {t('landing.how.title')}
        </h2>
        <ol className="mt-4 grid gap-3 sm:grid-cols-2">
          {HOW_STEPS.map((key, i) => (
            <li
              key={key}
              style={{ animationDelay: `${Math.min(i, 5) * 80}ms` }}
              className="rounded-card border border-hairline bg-surface p-4 transition duration-150 ease-standard hover:border-brand/40 hover:shadow-md animate-rise-in"
            >
              <span className="inline-flex h-7 w-7 items-center justify-center rounded-full bg-brand text-sm font-semibold text-white">
                {i + 1}
              </span>
              <p className="mt-2 font-semibold text-ink">{t(`landing.how.steps.${key}.title`)}</p>
              <p className="mt-1 text-sm text-muted">{t(`landing.how.steps.${key}.body`)}</p>
            </li>
          ))}
        </ol>
      </section>

      <section aria-labelledby="landing-ladder" className="animate-fade-in">
        <h2 id="landing-ladder" className="text-h1 font-semibold text-ink">
          {t('landing.ladder.title')}
        </h2>
        <p className="mt-2 text-sm text-muted">{t('landing.ladder.intro')}</p>
        <ol className="mt-4 flex flex-col gap-2">
          {LADDER.map((rung) => (
            <li
              key={rung.join('-')}
              className="rounded-card border border-hairline bg-surface p-3 text-sm text-ink transition duration-150 ease-standard hover:border-brand/40 hover:shadow-sm"
            >
              {rung.map((tierKey) => t(`problem.tier.${tierKey}`)).join(' · ')}
            </li>
          ))}
        </ol>
      </section>

      <section aria-labelledby="landing-accountability" className="animate-fade-in">
        <h2 id="landing-accountability" className="text-h1 font-semibold text-ink">
          {t('landing.accountability.title')}
        </h2>
        <p className="mt-2 text-ink">{t('landing.accountability.body')}</p>

        <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div className="rounded-card border border-hairline bg-surface p-4 text-center">
            <p className="font-semibold text-resolved-fg">{t('scorecard.resolved')}</p>
          </div>
          <div className="rounded-card border border-hairline bg-surface p-4 text-center">
            <p className="font-semibold text-ink">{t('scorecard.pending')}</p>
          </div>
          <div className="rounded-card bg-blocked-bg p-4 text-center">
            <p className="font-semibold text-blocked-fg">{t('scorecard.blocked')}</p>
          </div>
          <div className="rounded-card border border-hairline bg-surface p-4 text-center">
            <p className="font-semibold text-ink">{t('scorecard.avgResponse')}</p>
          </div>
        </div>
        <p className="mt-3 text-sm text-muted">{t('scorecard.fairnessNote')}</p>

        <Link
          to="/officials"
          className="mt-4 inline-flex min-h-11 items-center rounded-control border border-hairline bg-surface px-4 text-sm font-medium text-brand hover:bg-canvas"
        >
          {t('landing.accountability.cta')}
        </Link>
      </section>

      <section className="rounded-card border border-hairline bg-gradient-to-b from-brand-tint to-canvas p-6 text-center shadow-sm animate-fade-in sm:p-8">
        <p className="text-ink">{t('landing.closing.body')}</p>
        <div className="mt-4 flex flex-wrap justify-center gap-3">
          <HeroCtas t={t} canReport={canReport} />
        </div>
      </section>
    </div>
  )
}
