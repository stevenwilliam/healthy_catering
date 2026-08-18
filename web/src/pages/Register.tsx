import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiFailure, registerCustomer } from '../lib/api'
import { FieldError, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'

export default function Register() {
  const t = useT()
  const [form, setForm] = useState({ email: '', password: '', full_name: '', phone: '' })
  const [pending, setPending] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [fields, setFields] = useState<Record<string, string>>({})

  function set(k: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [k]: e.target.value })
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setPending(true); setError(null); setFields({})

    // Client-side validation is for FEEDBACK only. The server re-checks all of
    // it, because this form can be bypassed with curl (CLAUDE.md §4).
    if (form.password.length < 12) {
      setFields({ password: t('register.password_short') })
      setPending(false)
      return
    }

    try {
      await registerCustomer(form)
      setDone(true)
    } catch (err) {
      if (err instanceof ApiFailure) {
        setError(err.message)
        setFields(err.details)
      } else {
        setError(t('register.failed'))
      }
    } finally {
      setPending(false)
    }
  }

  if (done) {
    return (
      <div className="mx-auto max-w-md">
        <h1 className="text-3xl mb-4">{t('register.done_title')}</h1>
        <p>{t('register.done_body')}</p>
        <p className="mt-6">
          <Link className="underline" to="/login">{t('register.back_to_login')}</Link>
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-md">
      <h1 className="text-3xl mb-6">{t('register.title')}</h1>
      <form onSubmit={submit} noValidate>
        <div className="mb-4">
          <label className="label" htmlFor="name">{t('register.name')}</label>
          <input id="name" className="field" value={form.full_name} onChange={set('full_name')} required />
          <FieldError message={fields.full_name} />
        </div>
        <div className="mb-4">
          <label className="label" htmlFor="email">{t('field.email')}</label>
          <input id="email" className="field" type="email" autoComplete="email"
                 value={form.email} onChange={set('email')} required />
          <FieldError message={fields.email} />
        </div>
        <div className="mb-4">
          <label className="label" htmlFor="phone">{t('register.phone')}</label>
          <input id="phone" className="field" inputMode="tel" placeholder="0812 3456 7890"
                 value={form.phone} onChange={set('phone')} />
          <FieldError message={fields.phone} />
        </div>
        <div className="mb-4">
          <label className="label" htmlFor="password">{t('field.password')}</label>
          <input id="password" className="field" type="password" autoComplete="new-password"
                 value={form.password} onChange={set('password')} required />
          <p className="text-xs text-ink-muted mt-1">{t('register.password_hint')}</p>
          <FieldError message={fields.password} />
        </div>
        <FieldError message={error ?? undefined} />
        <SubmitButton pending={pending}>{t('register.submit')}</SubmitButton>
      </form>
    </div>
  )
}
