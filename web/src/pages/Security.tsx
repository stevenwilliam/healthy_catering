import { FormEvent, useEffect, useState } from 'react'
import {
  ApiFailure,
  MfaStatus,
  mfaConfirm,
  mfaDisable,
  mfaStart,
  mfaStatus,
} from '../lib/api'
import { CopyButton, FieldError, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'

// Security is where a signed-in user manages their second factor.
export default function Security() {
  const t = useT()
  const [status, setStatus] = useState<MfaStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  // Enrolment in progress.
  const [secret, setSecret] = useState<string | null>(null)
  const [otpauth, setOtpauth] = useState('')
  const [code, setCode] = useState('')
  const [recovery, setRecovery] = useState<string[] | null>(null)

  const [password, setPassword] = useState('')

  async function refresh() {
    try {
      setStatus(await mfaStatus())
    } catch (err) {
      setError(err instanceof ApiFailure ? err.message : t('security.status_failed'))
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  async function start() {
    setPending(true)
    setError(null)
    try {
      const out = await mfaStart()
      setSecret(out.secret)
      setOtpauth(out.otpauth_url)
      setCode('')
    } catch (err) {
      setError(err instanceof ApiFailure ? err.message : t('security.start_failed'))
    } finally {
      setPending(false)
    }
  }

  async function confirm(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      const out = await mfaConfirm(code)
      setRecovery(out.recovery_codes)
      setSecret(null)
      await refresh()
    } catch (err) {
      setError(err instanceof ApiFailure ? err.message : t('mfa.failed'))
      setCode('')
    } finally {
      setPending(false)
    }
  }

  async function disable(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      await mfaDisable(password)
      setPassword('')
      setRecovery(null)
      await refresh()
    } catch (err) {
      setError(err instanceof ApiFailure ? err.message : t('security.disable_failed'))
    } finally {
      setPending(false)
    }
  }

  if (!status) return <p>{t('ui.loading')}</p>

  if (!status.available) {
    return (
      <div className="mx-auto max-w-xl">
        <h1 className="text-3xl mb-4">{t('security.title')}</h1>
        <p className="text-sm text-beige-deep">
          {t('security.unavailable')}
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-xl">
      <h1 className="text-3xl mb-2">{t('security.title')}</h1>
      <p className="text-sm text-beige-deep mb-6">
        {t('security.intro')}
        {status.required && ` ${t('security.required')}`}
      </p>

      <FieldError message={error ?? undefined} />

      {/* The recovery codes are shown once and only once. */}
      {recovery && (
        <div className="mb-8 rounded border border-edge p-4">
          <h2 className="text-xl mb-2">{t('security.save_recovery')}</h2>
          <p className="text-sm text-beige-deep mb-4">
            {t('security.recovery_note')}
          </p>
          <ul className="grid grid-cols-2 gap-2 font-mono text-sm">
            {recovery.map((c) => (
              <li key={c} className="rounded bg-beige/5 border border-edge px-3 py-2">{c}</li>
            ))}
          </ul>
          <div className="mt-4">
            <CopyButton value={recovery.join('\n')} label={t('security.copy_all')} />
          </div>
        </div>
      )}

      {status.enabled ? (
        <div>
          <p className="mb-4">
            <span className="font-medium">{t('security.on')}</span>{' '}
            {typeof status.recovery_codes_left === 'number' &&
              `${status.recovery_codes_left} ${t('security.codes_left')}`}
          </p>
          {status.required ? (
            <p className="text-sm text-beige-deep">
              {t('security.locked_on')}
            </p>
          ) : (
            <form onSubmit={disable} className="mt-6">
              <label className="label" htmlFor="pw">
                {t('security.password_to_disable')}
              </label>
              <input
                id="pw"
                className="field"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
              <SubmitButton pending={pending}>{t('security.disable')}</SubmitButton>
            </form>
          )}
        </div>
      ) : secret ? (
        <div>
          <h2 className="text-xl mb-2">{t('security.step1')}</h2>
          <p className="text-sm text-beige-deep mb-3">
            {t('security.step1_hint')}
          </p>
          <p className="font-mono text-sm break-all rounded bg-beige/5 border border-edge px-3 py-2 mb-2">
            {secret}
          </p>
          <p className="text-xs text-beige-deep mb-6 break-all">{otpauth}</p>

          <h2 className="text-xl mb-2">{t('security.step2')}</h2>
          <form onSubmit={confirm} noValidate>
            <label className="label" htmlFor="code">{t('security.six_digit')}</label>
            <input
              id="code"
              className="field tracking-[0.4em] text-center text-xl"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={8}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
            />
            <p className="mt-2 mb-4 text-xs text-beige-deep">
              {t('security.nothing_changes')}
            </p>
            <SubmitButton pending={pending}>{t('security.enable')}</SubmitButton>
          </form>
        </div>
      ) : (
        <button type="button" className="btn-primary" disabled={pending} onClick={() => void start()}>
          {t('security.turn_on')}
        </button>
      )}
    </div>
  )
}
