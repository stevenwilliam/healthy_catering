/** The one place the SPA talks to the API.
 *
 * Types are hand-written here for now and generated from OpenAPI later
 * (docs/02 D-3) — the important part is that there is exactly ONE module that
 * knows the wire format, so a contract change breaks in one file.
 */

export type ApiError = {
  code: string
  message: string
  details?: Record<string, string>
}

export class ApiFailure extends Error {
  code: string
  details: Record<string, string>
  status: number
  constructor(status: number, e: ApiError) {
    super(e.message)
    this.status = status
    this.code = e.code
    this.details = e.details ?? {}
  }
}

const TOKEN_KEY = 'evermore.session'

type Session = {
  access_token: string
  refresh_token: string
  user_id: string
  customer_id?: string
  full_name: string
  email: string
  roles: string[]
  permissions: string[]
  email_verified: boolean

  // Set when the password was right but a second factor is still owed. The
  // access_token is EMPTY in that case, so this is never a usable session.
  mfa_required?: boolean
  mfa_token?: string
  mfa_hint?: string
}

export function loadSession(): Session | null {
  const raw = localStorage.getItem(TOKEN_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as Session
  } catch {
    // A corrupt entry is cleared rather than crashing every page that reads it.
    localStorage.removeItem(TOKEN_KEY)
    return null
  }
}

export function saveSession(s: Session) {
  localStorage.setItem(TOKEN_KEY, JSON.stringify(s))
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
}

export function can(permission: string): boolean {
  // The UI hides what you cannot do — but the SERVER is what enforces it. This
  // is presentation only; every one of these is re-checked per request.
  return loadSession()?.permissions.includes(permission) ?? false
}

type RequestOptions = {
  method?: string
  body?: unknown
  form?: FormData
  idempotencyKey?: string
}

let refreshing: Promise<boolean> | null = null

/** request is the single fetch wrapper: auth, refresh, and one error shape. */
export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const doFetch = async (): Promise<Response> => {
    const session = loadSession()
    const headers: Record<string, string> = {}
    if (session) headers['Authorization'] = `Bearer ${session.access_token}`
    if (opts.idempotencyKey) headers['Idempotency-Key'] = opts.idempotencyKey

    let body: BodyInit | undefined
    if (opts.form) {
      body = opts.form
    } else if (opts.body !== undefined) {
      headers['Content-Type'] = 'application/json'
      body = JSON.stringify(opts.body)
    }

    return fetch(`/api/v1${path}`, { method: opts.method ?? 'GET', headers, body })
  }

  let res = await doFetch()

  // One refresh attempt, shared across concurrent 401s so a page with four
  // panels does not rotate the refresh token four times and invalidate itself.
  if (res.status === 401 && loadSession()?.refresh_token) {
    refreshing ??= refreshSession().finally(() => {
      refreshing = null
    })
    if (await refreshing) res = await doFetch()
  }

  const text = await res.text()
  const payload = text ? JSON.parse(text) : {}

  if (!res.ok) {
    throw new ApiFailure(res.status, payload.error ?? { code: 'UNKNOWN', message: 'Something went wrong.' })
  }
  return payload.data as T
}

async function refreshSession(): Promise<boolean> {
  const session = loadSession()
  if (!session) return false
  const res = await fetch('/api/v1/auth/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: session.refresh_token }),
  })
  if (!res.ok) {
    clearSession()
    return false
  }
  const { data } = await res.json()
  saveSession(data)
  return true
}

export async function login(email: string, password: string): Promise<Session> {
  const s = await request<Session>('/auth/login', { method: 'POST', body: { email, password } })
  // A challenge is NOT stored: saving it would put a token in localStorage that
  // looks like a session to every other page but opens nothing.
  if (!s.mfa_required) saveSession(s)
  return s
}

// completeMfa exchanges a challenge plus a code for the real session.
export async function completeMfa(mfaToken: string, code: string): Promise<Session> {
  const s = await request<Session>('/auth/mfa', {
    method: 'POST',
    body: { mfa_token: mfaToken, code },
  })
  saveSession(s)
  return s
}

export type MfaStatus = {
  available: boolean
  required: boolean
  enabled: boolean
  pending?: boolean
  recovery_codes_left?: number
}

export const mfaStatus = () => request<MfaStatus>('/me/mfa')

export const mfaStart = () =>
  request<{ secret: string; otpauth_url: string; message: string }>('/me/mfa/start', {
    method: 'POST',
  })

export const mfaConfirm = (code: string) =>
  request<{ recovery_codes: string[]; message: string }>('/me/mfa/confirm', {
    method: 'POST',
    body: { code },
  })

export const mfaDisable = (password: string) =>
  request<{ enabled: boolean; message: string }>('/me/mfa/disable', {
    method: 'POST',
    body: { password },
  })

export async function registerCustomer(input: {
  email: string
  password: string
  full_name: string
  phone?: string
}): Promise<{ message: string }> {
  return request('/auth/register', { method: 'POST', body: input })
}

export async function logout() {
  try {
    await request('/auth/logout', { method: 'POST' })
  } finally {
    clearSession()
  }
}

// ── Money ───────────────────────────────────────────────────────────────────

/** formatIDR renders whole rupiah the Indonesian way: Rp 500.000.
 *
 * Amounts arrive as integers and stay integers; there is no float arithmetic
 * on this side either (CLAUDE.md §4). The API also sends a preformatted
 * string, which is preferred when present.
 */
export function formatIDR(amount: number): string {
  return 'Rp ' + Math.round(amount).toLocaleString('id-ID')
}

export type Page<T> = { items: T[]; total: number; page: number; page_size: number }

/** newIdempotencyKey returns a unique key for a mutating request.
 *
 * NOT crypto.randomUUID(): that is only defined in a SECURE CONTEXT, so over
 * plain HTTP on a LAN address it is undefined and throws — which silently
 * broke checkout with a generic "could not create order" and no clue why.
 * The site will be HTTPS in production, but a key generator that works only
 * after the certificate arrives is a trap for every future developer testing
 * by IP.
 */
export function newIdempotencyKey(): string {
  const bytes = new Uint8Array(16)
  if (globalThis.crypto?.getRandomValues) {
    globalThis.crypto.getRandomValues(bytes)
  } else {
    for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256)
  }
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}
