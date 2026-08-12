import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/useAuth.js'
import { useAreas } from '../../hooks/useAreas.js'
import { PHONE_PATTERN, normalizePhone } from '../../lib/phone.js'
import Input from '../../components/ui/Input.jsx'
import Select from '../../components/ui/Select.jsx'
import Button from '../../components/ui/Button.jsx'
import Card from '../../components/ui/Card.jsx'

export default function Register() {
  const { t } = useTranslation()
  const { register } = useAuth()
  const navigate = useNavigate()
  const { data: unions, isLoading: unionsLoading, isError: unionsError, refetch: refetchUnions } = useAreas({
    level: 'union',
  })

  const [form, setForm] = useState({
    name: '',
    phone: '',
    nid: '',
    unionId: '',
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
    // A name with no letter in it is a phone number in the wrong box — which is
    // exactly what happened, leaving an account greeted as "স্বাগতম, 01540787241"
    // and its owner's number sitting in a field meant for a name. \p{L} accepts
    // Bengali and Latin alike; only an all-digits/punctuation string is refused.
    else if (!/\p{L}/u.test(form.name)) next.name = t('auth.errors.nameInvalid')
    // Normalize first: the backend accepts any spelling of the number, so this
    // check must not be stricter than the server it is previewing.
    if (!PHONE_PATTERN.test(normalizePhone(form.phone))) next.phone = t('auth.errors.phoneInvalid')
    if (!form.nid.trim()) next.nid = t('auth.errors.required')
    // Still required: a select can be left unchosen just as a text field can be left blank.
    if (!form.unionId) next.unionId = t('auth.errors.required')
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
      await register({
        name: form.name.trim(),
        phone: canonicalPhone,
        nid: form.nid.trim(),
        unionId: form.unionId,
        password: form.password,
      })
      navigate('/login', { state: { registeredPhone: canonicalPhone } })
    } catch (err) {
      setSubmitError(err?.status === 409 ? t('auth.errors.phoneTaken') : t('auth.errors.registerFailed'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="mx-auto max-w-120">
      <h1 className="text-h1 font-semibold text-ink">{t('auth.register.title')}</h1>
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
        <Select
          label={t('auth.register.unionId')}
          value={form.unionId}
          onChange={updateField('unionId')}
          error={errors.unionId}
          disabled={unionsLoading || unionsError}
        >
          <option value="">
            {unionsLoading ? t('auth.register.unionsLoading') : t('auth.register.unionIdChoose')}
          </option>
          {(unions ?? []).map((union) => (
            <option key={union.id} value={union.id}>
              {union.name}
            </option>
          ))}
        </Select>
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

        {/* Without the geography there is no union to submit, and the backend would
            answer the empty one with a 400 invalid_union the registrant cannot act
            on. Say what actually failed and offer the retry instead. */}
        {unionsError && (
          <div className="flex flex-col items-start gap-2">
            <p className="text-sm text-reopened-fg">{t('auth.register.unionsError')}</p>
            <Button type="button" variant="secondary" onClick={() => refetchUnions()}>
              {t('common.retry')}
            </Button>
          </div>
        )}

        <Button type="submit" disabled={isSubmitting || unionsError} className="w-full">
          {t('auth.register.submit')}
        </Button>
      </form>
      <p className="mt-4 text-sm text-muted">
        {t('auth.register.haveAccount')}{' '}
        <Link to="/login" className="font-medium text-brand">
          {t('auth.register.loginLink')}
        </Link>
      </p>
      <p className="mt-1 text-sm text-muted">
        {t('auth.register.officialPrompt')}{' '}
        <Link to="/register/official" className="font-medium text-brand">
          {t('auth.register.officialLink')}
        </Link>
      </p>
    </Card>
  )
}
