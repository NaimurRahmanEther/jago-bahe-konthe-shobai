import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/useAuth.js'
import { PHONE_PATTERN, normalizePhone } from '../../lib/phone.js'
import Input from '../../components/ui/Input.jsx'
import Button from '../../components/ui/Button.jsx'
import Card from '../../components/ui/Card.jsx'

export default function Login() {
  const { t } = useTranslation()
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const [phone, setPhone] = useState(location.state?.registeredPhone ?? '')
  const [password, setPassword] = useState('')
  const [errors, setErrors] = useState({})
  const [submitError, setSubmitError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setSubmitError('')

    // Normalize before validating, so a number pasted by autofill as
    // "+880 1712-345678" is the same number the account was registered with.
    const canonicalPhone = normalizePhone(phone)

    const next = {}
    if (!canonicalPhone) next.phone = t('auth.errors.required')
    else if (!PHONE_PATTERN.test(canonicalPhone)) next.phone = t('auth.errors.phoneInvalid')
    if (!password) next.password = t('auth.errors.required')
    setErrors(next)
    if (Object.keys(next).length > 0) return

    setIsSubmitting(true)
    try {
      const session = await login(canonicalPhone, password)
      const from = location.state?.from?.pathname
      const roleHome =
        session.role === 'admin'
          ? '/admin'
          : session.role === 'super_admin'
            ? '/super'
            : session.role === 'official'
              ? '/official'
              : '/problems'
      navigate(from ?? roleHome, { replace: true })
    } catch (err) {
      // Only a 401 means the credentials were wrong. This used to be a bare
      // catch showing "phone or password is not valid" for a malformed phone, a
      // rate limit and a dead server alike — telling the user to doubt the one
      // thing that was correct. client.js normalizes every failure to {status}.
      if (err?.status === 400) setErrors({ phone: t('auth.errors.phoneInvalid') })
      else if (err?.status === 401) setSubmitError(t('auth.errors.loginFailed'))
      else if (err?.status === 429) setSubmitError(t('auth.errors.tooManyAttempts'))
      else setSubmitError(t('auth.errors.serverUnavailable'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="mx-auto max-w-105">
      <h1 className="text-h1 font-semibold text-ink">{t('auth.login.title')}</h1>
      {location.state?.registeredOfficial ? (
        <p className="mt-2 text-sm text-validated-fg">{t('auth.registerOfficial.success')}</p>
      ) : location.state?.registeredPhone ? (
        <p className="mt-2 text-sm text-validated-fg">{t('auth.register.success')}</p>
      ) : null}
      <form className="mt-4 flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
        <Input
          label={t('auth.login.phone')}
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          error={errors.phone}
          inputMode="numeric"
          autoComplete="tel"
        />
        <Input
          label={t('auth.login.password')}
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          error={errors.password}
          autoComplete="current-password"
        />

        {submitError && <p className="text-sm text-reopened-fg">{submitError}</p>}

        <Button type="submit" disabled={isSubmitting} className="w-full">
          {t('auth.login.submit')}
        </Button>
      </form>
      <p className="mt-4 text-sm text-muted">
        {t('auth.login.noAccount')}{' '}
        <Link to="/register" className="font-medium text-brand">
          {t('auth.login.registerLink')}
        </Link>
      </p>
    </Card>
  )
}
