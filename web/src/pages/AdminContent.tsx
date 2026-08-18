import { useEffect, useState } from 'react'
import { ApiFailure, request } from '../lib/api'
import { SearchBox, State, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'
import RichText from '../components/RichText'

/** The home hero copy, in three languages.
 *
 * The model, which the screen has to make obvious or editors will fight it:
 * Indonesian is the SOURCE and the only language anyone must write. English and
 * Chinese are produced from it. Either can be overridden by hand, and an
 * override is then permanent — the translator never touches it again — which is
 * why "release" exists as an explicit action rather than an empty field.
 */

type Translation = {
  value: string
  is_html: boolean
  is_override: boolean
  stale: boolean
  empty: boolean
}

type Entry = {
  key: string
  source: string
  is_html: boolean
  values: Record<string, Translation>
}

type Payload = {
  entries: Entry[]
  translator: { available: boolean; provider: string }
  locales: { Locale: string; Tag: string; Endonym: string; English: string }[]
}

const DERIVED = ['en', 'zh'] as const

export default function AdminContent() {
  const t = useT()
  const [data, setData] = useState<Payload | null>(null)
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [note, setNote] = useState<string | null>(null)

  // Drafts are held per field so a half-typed edit is never sent, and so the
  // list can refresh under an open editor without stealing what was typed.
  const [draft, setDraft] = useState<Record<string, string>>({})

  function load() {
    setLoading(true)
    request<Payload>('/admin/content')
      .then((d) => {
        setData(d)
        setError(null)
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('content.load_failed')))
      .finally(() => setLoading(false))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const fieldKey = (key: string, locale: string) => `${key}:${locale}`

  async function saveSource(key: string) {
    setBusy(fieldKey(key, 'id'))
    setNote(null)
    try {
      const out = await request<{ translations: Record<string, string> }>(
        `/admin/content/${encodeURIComponent(key)}`,
        { method: 'PUT', body: { value: draft[fieldKey(key, 'id')] ?? '' } },
      )
      // Say what happened to each language rather than a bare "saved" — the
      // editor needs to know whether the machine actually ran.
      setNote(
        DERIVED.map((l) => `${l.toUpperCase()}: ${t(outcomeKey(out.translations[l]))}`).join(' · '),
      )
      setDraft((d) => {
        const next = { ...d }
        delete next[fieldKey(key, 'id')]
        DERIVED.forEach((l) => delete next[fieldKey(key, l)])
        return next
      })
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('content.save_failed'))
    } finally {
      setBusy(null)
    }
  }

  async function saveOverride(key: string, locale: string) {
    setBusy(fieldKey(key, locale))
    setNote(null)
    try {
      await request(`/admin/content/${encodeURIComponent(key)}/${locale}`, {
        method: 'PUT',
        body: { value: draft[fieldKey(key, locale)] ?? '' },
      })
      setDraft((d) => {
        const next = { ...d }
        delete next[fieldKey(key, locale)]
        return next
      })
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('content.save_failed'))
    } finally {
      setBusy(null)
    }
  }

  async function release(key: string, locale: string) {
    setBusy(fieldKey(key, locale))
    setNote(null)
    try {
      const out = await request<{ outcome: string }>(
        `/admin/content/${encodeURIComponent(key)}/${locale}`,
        { method: 'DELETE' },
      )
      setNote(`${locale.toUpperCase()}: ${t(outcomeKey(out.outcome))}`)
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('content.save_failed'))
    } finally {
      setBusy(null)
    }
  }

  async function retranslateAll() {
    setBusy('all')
    setNote(null)
    try {
      await request('/admin/content/retranslate', { method: 'POST', body: {} })
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('content.save_failed'))
    } finally {
      setBusy(null)
    }
  }

  const entries = (data?.entries ?? []).filter(
    (e) =>
      !q ||
      e.key.toLowerCase().includes(q.toLowerCase()) ||
      e.source.toLowerCase().includes(q.toLowerCase()),
  )

  return (
    <div>
      <h1 className="text-3xl mb-2">{t('content.title')}</h1>
      <p className="text-sm text-ink-muted mb-4 max-w-prose">{t('content.intro')}</p>

      {/* Whether the machine is on is the first thing an editor needs to know:
          it decides whether they must type the English themselves. */}
      {data && (
        <div className="card mb-6 flex flex-wrap items-center gap-3">
          <span className="badge">
            {data.translator.available
              ? `${t('content.auto_on')} · ${data.translator.provider}`
              : t('content.auto_off')}
          </span>
          <span className="text-sm text-ink-muted max-w-prose">
            {data.translator.available ? t('content.auto_on_hint') : t('content.auto_off_hint')}
          </span>
          {data.translator.available && (
            <button
              type="button"
              className="btn-ghost ml-auto"
              disabled={busy === 'all'}
              onClick={() => void retranslateAll()}
            >
              {t('content.retranslate_all')}
            </button>
          )}
        </div>
      )}

      {note && (
        <p className="card mb-4 text-sm" role="status">
          {note}
        </p>
      )}

      <SearchBox value={q} onChange={setQ} placeholder={t('content.search_placeholder')}
                 resultCount={entries.length} />

      <State loading={loading} error={error} empty={entries.length === 0}>
        <ul className="grid gap-4">
          {entries.map((e) => (
            <li key={e.key} className="card">
              <code className="text-xs text-ink-muted">{e.key}</code>

              {/* Source */}
              <div className="mt-2">
                <label className="label" htmlFor={`src-${e.key}`}>
                  {t('content.source_label')}
                </label>
                {e.is_html ? (
                  <RichText
                    id={`src-${e.key}`}
                    value={draft[fieldKey(e.key, 'id')] ?? e.source}
                    onChange={(html) =>
                      setDraft({ ...draft, [fieldKey(e.key, 'id')]: html })
                    }
                  />
                ) : (
                  <textarea
                    id={`src-${e.key}`}
                    className="field min-h-[4.5rem]"
                    value={draft[fieldKey(e.key, 'id')] ?? e.source}
                    onChange={(ev) =>
                      setDraft({ ...draft, [fieldKey(e.key, 'id')]: ev.target.value })
                    }
                  />
                )}
                <div className="mt-2">
                  <SubmitButton
                    pending={busy === fieldKey(e.key, 'id')}
                    type="button"
                    onClick={() => void saveSource(e.key)}
                    disabled={draft[fieldKey(e.key, 'id')] === undefined}
                  >
                    {t('content.save_source')}
                  </SubmitButton>
                </div>
              </div>

              {/* Derived languages */}
              <div className="mt-4 grid gap-4 sm:grid-cols-2">
                {DERIVED.map((l) => {
                  const tr =
                    e.values[l] ??
                    { value: '', is_html: e.is_html, is_override: false, stale: false, empty: true }
                  return (
                    <div key={l}>
                      <div className="flex flex-wrap items-center gap-2">
                        <label className="label mb-0" htmlFor={`${l}-${e.key}`}>
                          {l.toUpperCase()}
                        </label>
                        {tr.is_override && <span className="badge">{t('content.override')}</span>}
                        {/* Stale is the one that costs money if missed: the
                            Indonesian moved and this hand-written text did
                            not. */}
                        {tr.stale && (
                          <span className="badge bg-ember-light">{t('content.stale')}</span>
                        )}
                        {tr.empty && <span className="badge">{t('content.empty')}</span>}
                      </div>
                      {e.is_html ? (
                        <RichText
                          id={`${l}-${e.key}`}
                          value={draft[fieldKey(e.key, l)] ?? tr.value}
                          onChange={(html) =>
                            setDraft({ ...draft, [fieldKey(e.key, l)]: html })
                          }
                        />
                      ) : (
                        <textarea
                          id={`${l}-${e.key}`}
                          className="field mt-1 min-h-[4.5rem]"
                          lang={l}
                          value={draft[fieldKey(e.key, l)] ?? tr.value}
                          onChange={(ev) =>
                            setDraft({ ...draft, [fieldKey(e.key, l)]: ev.target.value })
                          }
                        />
                      )}
                      <div className="mt-2 flex flex-wrap gap-2">
                        <SubmitButton
                          pending={busy === fieldKey(e.key, l)}
                          type="button"
                          className="btn-ghost"
                          onClick={() => void saveOverride(e.key, l)}
                          disabled={draft[fieldKey(e.key, l)] === undefined}
                        >
                          {t('content.save_override')}
                        </SubmitButton>
                        {tr.is_override && (
                          <button
                            type="button"
                            className="btn-ghost"
                            disabled={busy === fieldKey(e.key, l)}
                            onClick={() => void release(e.key, l)}
                          >
                            {t('content.release')}
                          </button>
                        )}
                      </div>
                      {tr.empty && (
                        <p className="mt-1 text-xs text-ink-muted">{t('content.empty_hint')}</p>
                      )}
                    </div>
                  )
                })}
              </div>
            </li>
          ))}
        </ul>
      </State>
    </div>
  )
}

/** Maps a server outcome to a catalogue key, so the four states are worded in
 *  the editor's own language rather than echoed as API tokens. */
function outcomeKey(outcome: string | undefined) {
  switch (outcome) {
    case 'translated':
      return 'content.out_translated' as const
    case 'kept-override':
      return 'content.out_kept' as const
    case 'no-translator':
      return 'content.out_no_translator' as const
    case 'failed':
      return 'content.out_failed' as const
    default:
      return 'content.out_error' as const
  }
}
