import { useState } from 'react'
import { useT } from '../lib/i18n'
import { loadSession } from '../lib/api'

/** Export to CSV, on every data grid (99 §8).
 *
 * A fetch-and-download rather than a plain <a href>: these endpoints need the
 * Authorization header, and a link cannot carry one — it would land on the
 * login page instead of a file, which looks like a broken button.
 *
 * `params` is the grid's CURRENT search and filters, so what downloads is what
 * is on screen. An export that quietly returns everything, or page one, is
 * worse than no export because the difference is invisible until someone acts
 * on the numbers.
 */
export default function ExportCsv({
  path,
  params,
  filename,
}: {
  path: string
  params?: Record<string, string | number | undefined>
  filename: string
}) {
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState(false)

  async function run() {
    setBusy(true)
    setFailed(false)
    try {
      const q = new URLSearchParams({ format: 'csv' })
      for (const [k, v] of Object.entries(params ?? {})) {
        if (v !== undefined && v !== '') q.set(k, String(v))
      }
      const session = loadSession()
      const res = await fetch(`/api/v1${path}?${q}`, {
        headers: session ? { Authorization: `Bearer ${session.access_token}` } : {},
      })
      if (!res.ok) throw new Error(String(res.status))

      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${filename}.csv`
      document.body.appendChild(a)
      a.click()
      a.remove()
      // Revoked on the next tick: revoking synchronously can cancel the
      // download in some browsers before it has started reading the blob.
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch {
      setFailed(true)
    } finally {
      setBusy(false)
    }
  }

  return (
    <span className="inline-flex items-center gap-2">
      <button type="button" className="btn-ghost" disabled={busy} onClick={() => void run()}>
        {busy ? t('ui.processing') : t('csv.export')}
      </button>
      {/* Announced, not colour-only (99 §8). */}
      <span role="status" aria-live="polite" className="text-xs text-ink-muted">
        {failed && t('csv.failed')}
      </span>
    </span>
  )
}
