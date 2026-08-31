import { useEffect, useState } from 'react'
import { ApiFailure, Page, request } from '../lib/api'
import { SearchBox, State, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'

type Setting = {
  key: string; value: string; value_type: string; group: string
  label: string; description: string; is_secret: boolean; is_system: boolean
  updated_at: string; updated_by?: string
}

export default function AdminSettings() {
  const t = useT()
  const [page, setPage] = useState<Page<Setting> | null>(null)
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState<string | null>(null)
  const [draft, setDraft] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [fieldError, setFieldError] = useState<string | null>(null)

  function load() {
    setLoading(true)
    request<Page<Setting>>(`/admin/settings?q=${encodeURIComponent(q)}&page_size=200`)
      .then(setPage)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('set.load_failed')))
      .finally(() => setLoading(false))
  }
  useEffect(load, [q]) // eslint-disable-line react-hooks/exhaustive-deps

  async function save(key: string) {
    setSaving(true); setFieldError(null)
    try {
      await request(`/admin/settings/${encodeURIComponent(key)}`, {
        method: 'PUT',
        body: { value: draft, reason },
      })
      setEditing(null); setReason(''); load()
    } catch (e) {
      // The server type-checks the value, so "6pm" for a time comes back with
      // a message rather than being silently ignored.
      setFieldError(e instanceof ApiFailure ? (e.details.value ?? e.message) : t('set.save_failed'))
    } finally { setSaving(false) }
  }

  const groups = [...new Set(page?.items.map((s) => s.group) ?? [])]

  return (
    <div>
      <h1 className="text-3xl mb-2">{t('set.title')}</h1>
      <p className="text-sm text-beige-deep mb-6 max-w-prose">
        {t('set.audit_note')}
      </p>

      <SearchBox value={q} onChange={setQ} placeholder={t('set.search_placeholder')}
                 resultCount={page?.total} />

      <State loading={loading} error={error} empty={(page?.items.length ?? 0) === 0}>
        {groups.map((g) => (
          <section key={g} className="mb-8">
            <h2 className="text-xl mb-3 capitalize">{g}</h2>
            <ul className="grid gap-2">
              {page?.items.filter((s) => s.group === g).map((s) => (
                <li key={s.key} className="card">
                  <div className="flex flex-wrap items-baseline gap-2">
                    <span className="font-medium">{s.label}</span>
                    <code className="text-xs text-beige-deep">{s.key}</code>
                    {s.is_secret && <span className="badge">{t('set.secret')}</span>}
                  </div>
                  {s.description && (
                    <p className="text-sm text-beige-deep mt-1 max-w-prose">{s.description}</p>
                  )}

                  {editing === s.key ? (
                    <div className="mt-3 grid gap-2 max-w-xl">
                      <input className="field" value={draft} onChange={(e) => setDraft(e.target.value)}
                             aria-label={`${t('set.new_value')} ${s.label}`} />
                      <input className="field" placeholder={t('set.reason')}
                             value={reason} onChange={(e) => setReason(e.target.value)} />
                      {fieldError && <p className="error" role="alert">{fieldError}</p>}
                      <div className="flex gap-2">
                        <SubmitButton pending={saving} type="button" onClick={() => save(s.key)}>
                          {t('set.save')}
                        </SubmitButton>
                        <button className="btn-ghost" onClick={() => setEditing(null)}>{t('ui.cancel')}</button>
                      </div>
                    </div>
                  ) : (
                    <div className="mt-2 flex flex-wrap items-center gap-3">
                      <code className="text-sm break-all">{s.value || '—'}</code>
                      <button className="btn-ghost"
                              onClick={() => { setEditing(s.key); setDraft(s.is_secret ? '' : s.value); setFieldError(null) }}>
                        Ubah
                      </button>
                      {s.updated_by && (
                        <span className="text-xs text-beige-deep">
                          terakhir diubah oleh {s.updated_by}
                        </span>
                      )}
                    </div>
                  )}
                </li>
              ))}
            </ul>
          </section>
        ))}
      </State>
    </div>
  )
}
