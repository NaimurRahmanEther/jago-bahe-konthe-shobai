import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { registerOfficial } from '../../lib/api/auth.js'
import { PHONE_PATTERN, normalizePhone } from '../../lib/phone.js'
import Input from '../../components/ui/Input.jsx'
import Button from '../../components/ui/Button.jsx'
import Card from '../../components/ui/Card.jsx'
import OfficialPicker from '../../components/problem/OfficialPicker.jsx'

export default function RegisterOfficial() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [form, setForm] = useState({
    name: '',
    phone: '',
    nid: '',
    officialId: '',
    password: '',
    confirmPassword: '',
  })
  const [errors, setErrors] = useState({})
  const [submitError, setSubmitError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  function updateField(field) {
    return (e) => setForm((f) => ({ ...f, [field]: e.target.value }))
  }

  function validate() {
    const next = {}
    if (!form.name.trim()) next.name = t('auth.errors.required')
    else if (!/\p{L}/u.test(form.name)) next.name = t('auth.errors.nameInvalid')
    if (!PHONE_PATTERN.test(normalizePhone(form.phone))) next.phone = t('auth.errors.phoneInvalid')
    if (!form.nid.trim()) next.nid = t('auth.errors.required')
    if (!form.officialId) next.officialId = t('auth.registerOfficial.errors.officeRequired')
    if (form.password.length < 6) next.password = t('auth.errors.passwordLength')
    if (form.confirmPassword !== form.password) next.confirmPassword = t('auth.errors.passwordMismatch')
    setErrors(next)
    return Object.keys(next).length === 0
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setSubmitError('')
    if (!validate()) return

    setIsSubmitting(true)
    const canonicalPhone = normalizePhone(form.phone)
    try {
      // Deliberately NOT auto-logged-in, despite the token the endpoint returns: a
      // fresh official account acts as nothing until a reviewer approves the claim,
      // and dropping them onto /official would show an empty case list that reads as
      // "you have no work" rather than "your claim is pending" (CLAUDE.md F13).
      await registerOfficial({
        name: form.name.trim(),
        phone: canonicalPhone,
        nid: form.nid.trim(),
        officialId: form.officialId,
        password: form.password,
      })
      navigate('/login', { state: { registeredOfficial: true, registeredPhone: canonicalPhone } })
    } catch (err) {
      setSubmitError(err?.status === 409 ? t('auth.errors.phoneTaken') : t('auth.errors.registerFailed'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="mx-auto max-w-120">
      <h1 className="text-h1 font-semibold text-ink">{t('auth.registerOfficial.title')}</h1>
      <p className="mt-2 text-sm text-muted">{t('auth.registerOfficial.subtitle')}</p>
      <form className="mt-4 flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
        <Input
          label={t('auth.register.name')}
          value={form.name}
          onChange={updateField('name')}
          error={errors.name}
          autoComplete="name"
        />
        <Input
          label={t('auth.register.phone')}
          value={form.phone}
          onChange={updateField('phone')}
          error={errors.phone}
          inputMode="numeric"
          autoComplete="tel"
          placeholder="01XXXXXXXXX"
        />
        <Input
          label={t('auth.register.nid')}
          value={form.nid}
          onChange={updateField('nid')}
          error={errors.nid}
          inputMode="numeric"
        />

        {/* The office is chosen from the real directory — never typed. A registrant
            who could name their own tier could declare themselves MP; the office's
            own record supplies tier and area. */}
        <OfficialPicker value={form.officialId} onChange={(id) => setForm((f) => ({ ...f, officialId: id }))} />
        {errors.officialId && <p className="-mt-2 text-sm text-reopened-fg">{errors.officialId}</p>}

        <Input
          label={t('auth.register.password')}
          type="password"
          value={form.password}
          onChange={updateField('password')}
          error={errors.password}
          autoComplete="new-password"
        />
        <Input
          label={t('auth.register.confirmPassword')}
          type="password"
          value={form.confirmPassword}
          onChange={updateField('confirmPassword')}
          error={errors.confirmPassword}
          autoComplete="new-password"
        />

        {submitError && <p className="text-sm text-reopened-fg">{submitError}</p>}

        <Button type="submit" disabled={isSubmitting} className="w-full">
          {t('auth.registerOfficial.submit')}
        </Button>
      </form>
      <p className="mt-4 text-sm text-muted">
        {t('auth.register.haveAccount')}{' '}
        <Link to="/login" className="font-medium text-brand">
          {t('auth.register.loginLink')}
        </Link>
      </p>
    </Card>
  )
}
