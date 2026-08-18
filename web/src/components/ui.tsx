import { ReactNode, useEffect, useState } from 'react'
import { useT } from '../lib/i18n'

/** Shared UI pieces. Small on purpose: each one exists because the house rules
 *  demand a behaviour the raw element does not give. */

/** SearchBox — every list screen has one, no exceptions (CLAUDE.md §7).
 *
 * Debounced so typing does not fire a request per keystroke, and it announces
 * its result count to a screen reader rather than only showing it.
 */
export function SearchBox({
  value, onChange, placeholder, resultCount,
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  resultCount?: number
}) {
  const t = useT()
  const [local, setLocal] = useState(value)

  useEffect(() => {
    const t = setTimeout(() => {
      if (local !== value) onChange(local)
    }, 250)
    return () => clearTimeout(t)
  }, [local]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="mb-4">
      <label className="label" htmlFor="search">{t('ui.search')}</label>
      <input
        id="search"
        type="search"
        className="field"
        value={local}
        placeholder={placeholder ?? t('ui.search_placeholder')}
        onChange={(e) => setLocal(e.target.value)}
      />
      <p className="sr-only" role="status" aria-live="polite">
        {resultCount === undefined ? '' : `${resultCount} ${t('ui.results')}`}
      </p>
    </div>
  )
}

/** Button that disables itself for the life of the request (docs/10 §4).
 *
 * A double-tapped checkout is the classic way to create two orders; the
 * Idempotency-Key on the API is the real defence and this is the first one.
 */
export function SubmitButton({
  children, pending, className, ...rest
}: {
  children: ReactNode
  pending: boolean
  className?: string
} & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const t = useT()
  return (
    <button
      {...rest}
      type={rest.type ?? 'submit'}
      disabled={pending || rest.disabled}
      aria-busy={pending}
      className={className ?? 'btn-primary'}
    >
      {pending ? t('ui.processing') : children}
    </button>
  )
}

/** A disabled control explains itself rather than being a grey box (99 §8). */
export function Disabled({ reason, children }: { reason: string; children: ReactNode }) {
  return (
    <span className="inline-flex flex-col gap-1">
      {children}
      <span className="text-xs text-ink-muted">{reason}</span>
    </span>
  )
}

/** Empty, loading and error states, so no screen ships without all three. */
export function State({
  loading, error, empty, emptyText, children,
}: {
  loading?: boolean
  error?: string | null
  empty?: boolean
  emptyText?: string
  children: ReactNode
}) {
  const t = useT()
  if (loading) return <p className="text-ink-muted py-8">{t('ui.loading')}</p>
  if (error) {
    return (
      <p role="alert" className="error py-8">
        {error}
      </p>
    )
  }
  if (empty) return <p className="text-ink-muted py-8">{emptyText ?? t('ui.empty')}</p>
  return <>{children}</>
}

/** Field-level error, tied to its input for assistive technology. */
export function FieldError({ message }: { message?: string }) {
  if (!message) return null
  return (
    <p className="error" role="alert">
      {message}
    </p>
  )
}

/** Money is rendered from the server's preformatted string when there is one:
 *  the server already knows the locale rules and the integer arithmetic. */
export function Money({ formatted, amount }: { formatted?: string; amount?: number }) {
  if (formatted) return <span className="tabular-nums">{formatted}</span>
  return <span className="tabular-nums">Rp {(amount ?? 0).toLocaleString('id-ID')}</span>
}

/** Confirm before anything irreversible (docs/10 §4). */
export function ConfirmButton({
  label, question, onConfirm, className,
}: {
  label: string
  question: string
  onConfirm: () => void
  className?: string
}) {
  const t = useT()
  const [asking, setAsking] = useState(false)
  if (!asking) {
    return (
      <button type="button" className={className ?? 'btn-ghost'} onClick={() => setAsking(true)}>
        {label}
      </button>
    )
  }
  return (
    <span className="inline-flex items-center gap-2">
      <span className="text-sm">{question}</span>
      <button type="button" className="btn-danger" onClick={onConfirm}>{t('ui.yes')}</button>
      <button type="button" className="btn-ghost" onClick={() => setAsking(false)}>
        {t('ui.cancel')}
      </button>
    </span>
  )
}

/** copyText copies a string, working outside a secure context too.
 *
 * navigator.clipboard is undefined on plain HTTP — the same trap that made
 * crypto.randomUUID() break checkout. The dev host is HTTP today, so without
 * the fallback the copy buttons would silently do nothing, which is worse than
 * not offering them: the customer believes they hold the account number.
 */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (globalThis.isSecureContext && navigator.clipboard) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // Fall through — a permissions refusal is not a reason to give up.
  }
  try {
    const el = document.createElement('textarea')
    el.value = text
    // Off-screen rather than hidden: display:none cannot be selected.
    el.style.position = 'fixed'
    el.style.opacity = '0'
    document.body.appendChild(el)
    el.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(el)
    return ok
  } catch {
    return false
  }
}

/** CopyButton copies a value and confirms it in words, not just colour. */
export function CopyButton({
  value,
  label,
  className = '',
}: {
  value: string
  /** Defaults to the catalogue's "Copy" — a default parameter cannot call a
   *  hook, so the fallback is applied in the body. */
  label?: string
  className?: string
}) {
  const t = useT()
  const [state, setState] = useState<'idle' | 'ok' | 'fail'>('idle')

  async function run() {
    const ok = await copyText(value)
    setState(ok ? 'ok' : 'fail')
    window.setTimeout(() => setState('idle'), 2000)
  }

  return (
    <span className="inline-flex items-center gap-2">
      <button type="button" className={`btn-ghost ${className}`} onClick={() => void run()}>
        {label ?? t('ui.copy')}
      </button>
      {/* Announced, so the confirmation is not colour-only (99 §8). */}
      <span role="status" aria-live="polite" className="text-xs text-ink-muted">
        {state === 'ok' && t('ui.copied')}
        {state === 'fail' && t('ui.copy_failed')}
      </span>
    </span>
  )
}
