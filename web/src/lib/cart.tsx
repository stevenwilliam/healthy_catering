import {
  createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode,
} from 'react'

/** The cart, as artboards 01–04 need it.
 *
 * It lives in the client and in localStorage, not on the server, and that is a
 * decision rather than a shortcut: a cart line is not a reservation. Capacity
 * is taken at checkout inside one transaction with `SELECT … FOR UPDATE`
 * (CLAUDE.md §4), so holding a server-side cart would either oversell or
 * reserve stock for someone who has closed the tab.
 *
 * What it stores is a snapshot of the meal — name, slot, date, diet — because
 * the cart has to render before the menu request comes back on a cold load,
 * and because a meal that gets unpublished must still be nameable when we tell
 * the customer it is gone. It is NOT the price: prices come from /quote on the
 * whole basket every time the quantity changes, because the tier depends on
 * the total (docs/02 B-11) and a remembered price is a wrong price.
 */

export type CartLine = {
  mealID: string
  qty: number
  /** Snapshot, for rendering only. Never trusted for money. */
  name: string
  slot: string
  serviceDate: string
  dietType: string
  dietTypeID: string
  photoKey?: string
}

type Ctx = {
  lines: CartLine[]
  totalMeals: number
  /** Distinct service date + slot pairs — what "2 pengantaran" counts. */
  dropCount: number
  qtyOf: (mealID: string) => number
  setQty: (meal: Omit<CartLine, 'qty'>, qty: number) => void
  add: (meal: Omit<CartLine, 'qty'>, by?: number) => void
  remove: (mealID: string) => void
  clear: () => void
}

const CartContext = createContext<Ctx | null>(null)
const KEY = 'evermore.cart'

/** Reading is defensive on purpose: this value survives deploys, so a shape
 *  written by an older build must not crash the app it loads into. */
function load(): CartLine[] {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((l): l is CartLine =>
      !!l && typeof l === 'object' &&
      typeof (l as CartLine).mealID === 'string' &&
      typeof (l as CartLine).qty === 'number' &&
      (l as CartLine).qty > 0)
  } catch {
    return []
  }
}

export function CartProvider({ children }: { children: ReactNode }) {
  const [lines, setLines] = useState<CartLine[]>(load)

  useEffect(() => {
    try {
      localStorage.setItem(KEY, JSON.stringify(lines))
    } catch {
      // A private window that refuses storage still gets a working cart for
      // the life of the tab; losing it on refresh beats failing to add.
    }
  }, [lines])

  const setQty = useCallback((meal: Omit<CartLine, 'qty'>, qty: number) => {
    // Clamped, not trusted: the stepper is bounded but a restored localStorage
    // value or a fast double-tap is not.
    const n = Math.max(0, Math.min(999, Math.floor(qty) || 0))
    setLines((prev) => {
      const without = prev.filter((l) => l.mealID !== meal.mealID)
      return n === 0 ? without : [...without, { ...meal, qty: n }]
    })
  }, [])

  const value = useMemo<Ctx>(() => {
    const qtyOf = (id: string) => lines.find((l) => l.mealID === id)?.qty ?? 0
    return {
      lines,
      totalMeals: lines.reduce((n, l) => n + l.qty, 0),
      // The canvas's "Ongkos kirim · 2 pengantaran": one drop per date+slot,
      // however many meals ride in it.
      dropCount: new Set(lines.map((l) => `${l.serviceDate}|${l.slot}`)).size,
      qtyOf,
      setQty,
      add: (meal, by = 1) => setQty(meal, qtyOf(meal.mealID) + by),
      remove: (id) => setLines((prev) => prev.filter((l) => l.mealID !== id)),
      clear: () => setLines([]),
    }
  }, [lines, setQty])

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>
}

export function useCart(): Ctx {
  const ctx = useContext(CartContext)
  if (!ctx) throw new Error('useCart outside CartProvider')
  return ctx
}
