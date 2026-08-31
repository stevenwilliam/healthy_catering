import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiFailure, Page, request } from '../lib/api'
import { State } from '../components/ui'
import { AppBar, BottomBar, Phone, dayLong } from '../components/mobile'
import { useI18n, useT } from '../lib/i18n'
import { localeTag, serviceDateWIB } from './Menu'

/** Artboard 06 — the credit balance, and where it went.
 *
 * The ledger is append-only in the database, and this screen shows it as one:
 * every movement, with its reason, in the order it happened. A refund appears
 * as "+1 · Dapur tidak bisa memenuhi", not as a silently larger balance —
 * which is the difference between a customer trusting the number and having to
 * ask about it.
 *
 * The expiry is stated twice on purpose: as a date with days remaining on the
 * card, and as the rule at the foot. Credit that quietly evaporates is the
 * complaint this screen exists to prevent.
 */

type Mine = {
  id: string
  package_name: string
  status: string
  purchased_credits: number
  remaining_credits: number
  expires_at?: string
  price_paid: string
}

type Entry = {
  entry_type: string
  qty: number
  running_balance: number
  note: string
  occurred_at: string
}

export default function Credits() {
  const t = useT()
  const { locale } = useI18n()
  const [mine, setMine] = useState<Mine[]>([])
  const [ledger, setLedger] = useState<Entry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    request<Page<Mine>>('/my/packages')
      .then(async (p) => {
        // A Page<T> whose `items` is missing is not a page. Treating it as an
        // empty list gives the customer the "no active package" state instead
        // of a blank screen.
        const items = Array.isArray(p?.items) ? p.items : []
        setMine(items)
        const active = items.find((x) => x.remaining_credits > 0) ?? items[0]
        if (active) {
          try {
            setLedger(await request<Entry[]>(`/my/packages/${active.id}/ledger`))
          } catch {
            // The balance is still worth showing without its history.
          }
        }
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('packages.load_failed')))
      .finally(() => setLoading(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const active = useMemo(
    () => mine.find((x) => x.remaining_credits > 0) ?? mine[0],
    [mine],
  )

  const daysLeft = useMemo(() => {
    if (!active?.expires_at) return undefined
    // Whole days between two business dates, both read in Asia/Jakarta.
    const end = Date.parse(active.expires_at.slice(0, 10))
    const today = Date.parse(serviceDateWIB())
    if (Number.isNaN(end) || Number.isNaN(today)) return undefined
    return Math.max(0, Math.round((end - today) / 86_400_000))
  }, [active])

  const pct = active && active.purchased_credits > 0
    ? Math.round((active.remaining_credits / active.purchased_credits) * 100)
    : 0

  return (
    <Phone>
      <AppBar title={t('c06.title')} />

      <State loading={loading} error={error}>
        <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
          {!active ? (
            <div className="flex flex-1 flex-col items-center justify-center gap-4 py-16 text-center">
              <p className="text-beige-deep">{t('c06.none')}</p>
              <Link to="/packages" className="btn-primary">{t('c06.buy')}</Link>
            </div>
          ) : (
            <>
              <section className="rounded border border-beige p-4">
                <div className="kicker">{active.package_name}</div>
                <div className="font-display text-4xl font-bold leading-none">
                  {active.remaining_credits}{' '}
                  <span className="text-xl font-semibold text-beige-deep">
                    {t('c06.of_total', active.purchased_credits)}
                  </span>
                </div>
                {/* A bar, and the same fact in words above it — the bar alone
                    would be the only signal, which colour never is. */}
                <div
                  className="mt-3 h-2.5 overflow-hidden bg-bar"
                  role="progressbar"
                  aria-valuenow={active.remaining_credits}
                  aria-valuemin={0}
                  aria-valuemax={active.purchased_credits}
                >
                  <div className="h-full bg-beige" style={{ width: `${pct}%` }} />
                </div>
                {active.expires_at && daysLeft !== undefined && (
                  <p className="mt-3 text-sm text-beige-deep">
                    {t('c06.valid_until',
                       dayLong(active.expires_at.slice(0, 10), localeTag(locale)), daysLeft)}
                  </p>
                )}
              </section>

              <Link to="/book" className="btn-primary btn-block">{t('c06.use_credit')}</Link>

              <section>
                <h2 className="kicker mb-2">{t('c06.history')}</h2>
                {ledger.length === 0 ? (
                  <p className="text-sm text-beige-deep">{t('ui.empty')}</p>
                ) : (
                  <ul className="m-0 flex list-none flex-col gap-3 p-0">
                    {ledger.map((e, i) => (
                      <li key={`${e.occurred_at}-${i}`}
                          className="flex items-baseline justify-between gap-3 border-b border-rule pb-2 last:border-b-0">
                        <div className="min-w-0">
                          <div className="text-sm font-semibold">
                            {entryLabel(t, e.entry_type)} · {e.occurred_at.slice(0, 10)}
                          </div>
                          {e.note && (
                            <div className="text-sm text-beige-deep">{e.note}</div>
                          )}
                        </div>
                        {/* The sign is a character, not a colour: a credit
                            reads "+1" whether or not the blue renders. */}
                        <span className={`whitespace-nowrap text-base font-bold ${
                          e.qty > 0 ? 'text-ocean-light' : ''}`}>
                          {e.qty > 0 ? '+' : '−'}{Math.abs(e.qty)}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </section>

              <div className="note-emph">{t('c06.expiry_note')}</div>
            </>
          )}
        </div>
      </State>

      <BottomBar kicker={t('c06.need_more')} total={<span className="text-lg">{' '}</span>}>
        <Link to="/packages" className="btn-ghost">{t('c06.buy')}</Link>
      </BottomBar>
    </Phone>
  )
}

/** The ledger's entry types, worded for a customer.
 *
 * An unknown type falls back to the raw value rather than to a blank: an
 * unlabelled movement is still a movement the customer needs to see. */
function entryLabel(t: ReturnType<typeof useT>, kind: string): string {
  switch (kind) {
    case 'PURCHASE': return t('credit.purchase')
    case 'CONSUME': return t('credit.used')
    case 'REFUND': return t('credit.refunded')
    case 'EXPIRE': return t('credit.expired')
    default: return kind
  }
}
