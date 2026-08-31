import { useEffect, useMemo, useState } from 'react'
import { ApiFailure, request } from '../lib/api'
import { SearchBox, State } from '../components/ui'
import ExportCsv from '../components/ExportCsv'
import { useT } from '../lib/i18n'
import { serviceDateWIB } from './AdminDashboard'

/** S5 — kitchens and their service areas.
 *
 * The plot is a SCHEMATIC drawn from the real coordinates, not a Google Maps
 * tile layer. That is deliberate and it is not a placeholder pretending to be
 * a map: the kitchen positions and the service radii are the live values,
 * projected and drawn to scale, so the relative geography is truthful. What it
 * does not show is streets. Wiring the tile layer needs the browser Maps key
 * plumbed to the SPA — the public pages already receive one (PageData.MapsKey)
 * and the SPA does not — and that is recorded in RUN-WHEN-BACK.md rather than
 * faked here with a picture of a map.
 *
 * Coordinates are PII under UU PDP (CLAUDE.md §10), so nothing on this screen
 * is logged or sent anywhere; it renders and stops.
 */

type SlotLoad = {
  slot_id: string
  alias: string
  slot_time: string
  quota: number
  used: number
  available: boolean
}

type Kitchen = {
  id: string
  code: string
  name: string
  district: string
  city: string
  latitude: number
  longitude: number
  service_radius_km: number
  has_polygon: boolean
  polygon_points: number
  priority: number
  is_active: boolean
  slots: SlotLoad[]
}

type CoverageRow = {
  district: string
  city: string
  attempts: number
  notify_requests: number
  avg_distance_to_nearest_km: number
  nearest_kitchen: string
}

/** The pin colours, in kitchen priority order. Every one of these carries deep
 *  ink on it and measures ≥7:1 (docs/10 §4.8); they are also distinguishable
 *  without colour because each pin is labelled with its kitchen code. */
const PIN = ['#FFFAE0', '#B6DAFA', '#FFBC8F']

export default function AdminCoverage() {
  const t = useT()
  const [kitchens, setKitchens] = useState<Kitchen[]>([])
  const [gaps, setGaps] = useState<CoverageRow[]>([])
  const [selected, setSelected] = useState<string>('')
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const date = serviceDateWIB()

  useEffect(() => {
    Promise.all([
      request<Kitchen[]>(`/admin/kitchens?date=${date}`),
      request<CoverageRow[]>('/admin/reports/coverage'),
    ])
      .then(([k, c]) => {
        setKitchens(k)
        setGaps(c)
        if (k.length && !selected) setSelected(k[0]!.id)
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('cov.load_failed')))
      .finally(() => setLoading(false))
  }, [date]) // eslint-disable-line react-hooks/exhaustive-deps

  const active = kitchens.find((k) => k.id === selected) ?? kitchens[0]
  const plot = useMemo(() => project(kitchens), [kitchens])

  const filteredGaps = useMemo(() => {
    const needle = q.trim().toLowerCase()
    if (!needle) return gaps
    return gaps.filter((g) =>
      `${g.district} ${g.city} ${g.nearest_kitchen}`.toLowerCase().includes(needle))
  }, [gaps, q])

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-end justify-between gap-4">
        <h1>{t('cov.title')}</h1>
        <p className="max-w-xl text-sm text-beige-deep">{t('cov.rule')}</p>
      </div>

      <State loading={loading} error={error}>
        <div className="grid gap-6 xl:grid-cols-[1fr_430px] xl:items-start">
          {/* ── The schematic ──────────────────────────────────────────────── */}
          <section className="panel overflow-hidden">
            <div className="panel-head">
              <h2 className="text-xl">{t('cov.title')}</h2>
              <span className="text-xs text-beige-deep">{t('cov.schematic')}</span>
            </div>
            {/* SQUARE, deliberately. project() maps both axes with one
                kilometre scale, so a 4:3 box stretched every distance
                horizontally and turned the radius circles into ellipses —
                which made the picture lie about the geography it exists to
                show. */}
            <div className="relative aspect-square w-full bg-bar">
              {/* The graticule. Decorative, so it is aria-hidden and drawn in
                  the deep ink at low alpha rather than as content. */}
              <div
                aria-hidden="true"
                className="absolute inset-0"
                style={{
                  background:
                    'repeating-linear-gradient(90deg, rgba(28,61,52,0.16) 0 1px, transparent 1px 64px),' +
                    'repeating-linear-gradient(0deg, rgba(28,61,52,0.16) 0 1px, transparent 1px 64px)',
                }}
              />
              {plot.map((p, i) => {
                const colour = PIN[i % PIN.length]!
                return (
                  <div key={p.k.id}>
                    {/* Service radius, drawn to the same scale as the pins. */}
                    <div
                      aria-hidden="true"
                      className="absolute -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-dashed"
                      style={{
                        left: `${p.x}%`, top: `${p.y}%`,
                        // Width only, plus aspect-ratio: a percentage HEIGHT
                        // resolves against the container's height and a
                        // percentage width against its width, so setting both
                        // gives a circle only when the box happens to be
                        // square. This stays round whatever the box does.
                        width: `${p.rPct * 2}%`, aspectRatio: '1',
                        borderColor: colour,
                        background: `${colour}1F`,
                      }}
                    />
                    <button
                      type="button"
                      onClick={() => setSelected(p.k.id)}
                      aria-pressed={p.k.id === selected}
                      className="absolute flex -translate-x-1/2 -translate-y-1/2 items-center gap-2"
                      style={{ left: `${p.x}%`, top: `${p.y}%` }}
                    >
                      <span
                        className="h-4 w-4 rounded-full border-4 border-canvas"
                        style={{ background: colour }}
                      />
                      <span
                        className="whitespace-nowrap rounded-full px-3 py-1 text-xs font-bold text-nourish-deep"
                        style={{ background: colour }}
                      >
                        {p.k.code} {p.k.name}
                      </span>
                    </button>
                  </div>
                )
              })}
              {plot.length === 0 && (
                <p className="absolute inset-0 flex items-center justify-center text-sm text-beige">
                  {t('ui.empty')}
                </p>
              )}
            </div>
          </section>

          {/* ── The selected kitchen ───────────────────────────────────────── */}
          {active && (
            <section className="panel">
              <div className="border-b border-edge px-5 py-4">
                <h2 className="mb-1 text-2xl">{active.name}</h2>
                <p className="text-sm text-beige-deep">
                  {active.code} · {t('cov.priority')} {active.priority} ·{' '}
                  {active.is_active ? t('cov.active') : t('cov.inactive')}
                </p>
              </div>
              <div className="flex flex-col gap-4 p-5">
                <div>
                  <div className="kicker mb-2">{t('cov.radius')}</div>
                  <div className="flex min-h-touch items-center rounded-full border border-edge px-5 text-sm font-semibold">
                    {active.service_radius_km} km
                  </div>
                </div>
                <div>
                  <div className="kicker mb-2">{t('cov.polygon')}</div>
                  <div className="flex min-h-touch items-center justify-between rounded-full border border-edge px-5 text-sm font-semibold">
                    {active.has_polygon ? t('cov.polygon') : t('ui.none')}
                    {active.has_polygon && (
                      <span className="text-beige-deep">
                        {active.polygon_points} {t('cov.points')}
                      </span>
                    )}
                  </div>
                </div>

                <div className="gtable">
                  <div className="border-b border-edge px-4 py-3 font-display text-lg font-semibold">
                    {t('cov.slots_today')}
                  </div>
                  <div className="grid" style={{ gridTemplateColumns: '1fr 1fr 1fr' }}>
                    <div className="gtable-head">{t('cov.slot')}</div>
                    <div className="gtable-head">{t('cov.quota')}</div>
                    <div className="gtable-head">{t('cov.used')}</div>
                    {active.slots.map((s, i) => {
                      const last = i === active.slots.length - 1
                      const cell = `gtable-cell ${last ? 'is-last' : ''}`
                      const full = s.available && s.used >= s.quota
                      return (
                        <div key={s.slot_id} className="contents">
                          <div className={cell}>{s.slot_time}</div>
                          <div className={cell}>{s.available ? s.quota : t('dash.closed')}</div>
                          <div className={cell}>
                            {!s.available
                              ? '—'
                              : full
                                // docs/10 §4.1 #1 — berry deep, ringed.
                                ? <span className="pill-full">{s.used}</span>
                                : s.used}
                          </div>
                        </div>
                      )
                    })}
                    {active.slots.length === 0 && (
                      <p className="col-span-3 px-4 py-6 text-sm text-beige-deep">
                        {t('ui.empty')}
                      </p>
                    )}
                  </div>
                </div>

                <div className="note-emph">
                  <p className="m-0">{t('cov.manual_note')}</p>
                </div>
              </div>
            </section>
          )}
        </div>

        {/* ── Where demand exists that we cannot serve ────────────────────── */}
        <section className="mt-8">
          <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
            <h2>{t('dash.out_of_range')}</h2>
            <ExportCsv path="/admin/reports/coverage" filename="coverage" />
          </div>
          <SearchBox value={q} onChange={setQ} resultCount={filteredGaps.length} />
          <div className="overflow-x-auto">
            <div className="gtable min-w-[42rem]">
              <div className="grid" style={{ gridTemplateColumns: '1.4fr 1fr 0.7fr 0.9fr 1.1fr' }}>
                <div className="gtable-head">{t('cov.district')}</div>
                <div className="gtable-head">{t('cov.city')}</div>
                <div className="gtable-head">{t('cov.attempts')}</div>
                <div className="gtable-head">{t('cov.notify')}</div>
                <div className="gtable-head">{t('cov.nearest')}</div>
                {filteredGaps.map((g, i) => {
                  const cell = `gtable-cell ${i === filteredGaps.length - 1 ? 'is-last' : ''}`
                  return (
                    <div key={`${g.district}-${g.city}`} className="contents">
                      <div className={`${cell} font-semibold`}>{g.district}</div>
                      <div className={cell}>{g.city}</div>
                      <div className={cell}>{g.attempts}</div>
                      <div className={cell}>{g.notify_requests}</div>
                      <div className={cell}>
                        {g.nearest_kitchen}
                        <span className="ml-2 text-beige-deep">
                          {g.avg_distance_to_nearest_km.toFixed(1)} km
                        </span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
          {filteredGaps.length === 0 && (
            <p className="py-6 text-sm text-beige-deep">{t('ui.empty')}</p>
          )}
        </section>
      </State>
    </div>
  )
}

/** project places the kitchens in the plot box as percentages.
 *
 * An equirectangular projection with a cos(lat) correction on longitude, which
 * over one city is indistinguishable from anything fancier and keeps the
 * radius circles round. The extent is padded by the largest service radius so
 * no circle is clipped at the edge, and a single kitchen — where the extent
 * would be zero and every division a NaN — lands in the middle.
 */
export function project(kitchens: Kitchen[]): {
  k: Kitchen; x: number; y: number; rPct: number
}[] {
  if (kitchens.length === 0) return []

  const KM_PER_DEG_LAT = 110.574
  const midLat = kitchens.reduce((s, k) => s + k.latitude, 0) / kitchens.length
  const kmPerDegLng = 111.320 * Math.cos((midLat * Math.PI) / 180)

  // Work in kilometres from the centroid, so x and y share one scale.
  const pts = kitchens.map((k) => ({
    k,
    kx: (k.longitude - kitchens[0]!.longitude) * kmPerDegLng,
    ky: -(k.latitude - kitchens[0]!.latitude) * KM_PER_DEG_LAT,
  }))

  const maxR = Math.max(...kitchens.map((k) => k.service_radius_km), 1)
  const pad = maxR * 1.15
  const minX = Math.min(...pts.map((p) => p.kx)) - pad
  const maxX = Math.max(...pts.map((p) => p.kx)) + pad
  const minY = Math.min(...pts.map((p) => p.ky)) - pad
  const maxY = Math.max(...pts.map((p) => p.ky)) + pad

  // One scale for both axes — the wider of the two — or the circles turn into
  // ellipses and the picture stops being true.
  const span = Math.max(maxX - minX, maxY - minY) || 1
  const cx = (minX + maxX) / 2
  const cy = (minY + maxY) / 2

  return pts.map((p) => ({
    k: p.k,
    x: 50 + ((p.kx - cx) / span) * 100,
    y: 50 + ((p.ky - cy) / span) * 100,
    rPct: (p.k.service_radius_km / span) * 100,
  }))
}
