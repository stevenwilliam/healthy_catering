import { useEffect, useState } from 'react'
import { Link, NavLink, Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom'
import { loadSession, logout, request } from './lib/api'
import { I18nProvider, LOCALE_INFO, useI18n, useT } from './lib/i18n'
import { CartProvider } from './lib/cart'
import LanguageSelector from './components/LanguageSelector'
import Login from './pages/Login'
import Register from './pages/Register'
import Menu from './pages/Menu'
import MealDetail from './pages/MealDetail'
import Cart from './pages/Cart'
import Checkout from './pages/Checkout'
import Credits from './pages/Credits'
import BookSlot from './pages/BookSlot'
import Addresses from './pages/Addresses'
import Orders from './pages/Orders'
import OrderDetail from './pages/OrderDetail'
import Packages from './pages/Packages'
import AdminPayments from './pages/AdminPayments'
import AdminDeliveries from './pages/AdminDeliveries'
import AdminSettings from './pages/AdminSettings'
import AdminContent from './pages/AdminContent'
import AdminDashboard from './pages/AdminDashboard'
import AdminCalendar from './pages/AdminCalendar'
import AdminPricing from './pages/AdminPricing'
import AdminCoverage from './pages/AdminCoverage'
import Production from './pages/Production'
import PackingLabels from './pages/PackingLabels'
import Security from './pages/Security'

/** RequireAuth gates a route in the UI.
 *
 * Presentation only — the API re-checks every request. Hiding a screen a user
 * cannot use is a courtesy, not a control (99 §7).
 */
function RequireAuth({ children }: { children: JSX.Element }) {
  const location = useLocation()
  if (!loadSession()) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />
  }
  return children
}

/** WhatsAppFloat is the floating contact button, on every screen.
 *
 * The number is a sys_parameters row, not a build-time constant (CLAUDE.md
 * §7), so it comes from the public company endpoint. Plain fetch rather than
 * the request() helper: this route is unauthenticated and must also work on
 * the login screen, before there is a session to attach or refresh.
 *
 * Renders nothing until the number arrives, and nothing at all if none is
 * configured — a wa.me link with an empty number is a dead link, which is
 * worse than no button.
 */
function WhatsAppFloat() {
  const t = useT()
  const [number, setNumber] = useState('')

  useEffect(() => {
    let live = true
    fetch('/api/v1/public/company')
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => { if (live && d?.whatsapp) setNumber(d.whatsapp) })
      .catch(() => { /* the button is a convenience; never break the app for it */ })
    return () => { live = false }
  }, [])

  if (!number) return null

  return (
    <a
      className="wa-float"
      href={`https://wa.me/${number}`}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={t('wa.aria')}
    >
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path fill="currentColor" d="M12.04 2C6.58 2 2.13 6.45 2.13 11.91c0 1.75.46 3.45 1.32 4.95L2 22l5.25-1.38a9.9 9.9 0 0 0 4.79 1.22h.01c5.46 0 9.91-4.45 9.91-9.91C21.96 6.45 17.5 2 12.04 2zm0 18.15h-.01a8.2 8.2 0 0 1-4.19-1.15l-.3-.18-3.12.82.83-3.04-.2-.31a8.19 8.19 0 0 1-1.26-4.38c0-4.54 3.7-8.23 8.25-8.23 2.2 0 4.27.86 5.83 2.42a8.18 8.18 0 0 1 2.41 5.82c0 4.54-3.7 8.23-8.24 8.23zm4.52-6.16c-.25-.12-1.47-.72-1.69-.81-.23-.08-.39-.12-.56.13-.16.24-.64.8-.78.97-.14.16-.29.18-.54.06-.25-.13-1.05-.39-1.99-1.23-.74-.66-1.24-1.47-1.38-1.72-.15-.25-.02-.38.11-.5.11-.11.25-.29.37-.43.13-.15.17-.25.25-.41.08-.17.04-.31-.02-.43-.06-.12-.56-1.35-.77-1.84-.2-.49-.4-.42-.56-.43h-.47c-.17 0-.43.06-.66.31-.23.25-.86.85-.86 2.07 0 1.22.89 2.4 1.01 2.56.12.17 1.75 2.67 4.23 3.74.59.26 1.05.41 1.41.52.59.19 1.13.16 1.56.1.48-.07 1.47-.6 1.68-1.18.21-.58.21-1.07.14-1.18-.06-.11-.22-.17-.47-.29z"/>
      </svg>
    </a>
  )
}

/** The wordmark image. Served from /images by the Go server, not bundled by
 *  Vite, so it is the same file the public pages use. */
function Wordmark() {
  return (
    <img
      src="/images/evermore-wordmark-light.png"
      width={560}
      height={60}
      alt="Evermore"
      className="block h-6 w-auto sm:h-7"
    />
  )
}

function Nav() {
  const t = useT()
  const { locale } = useI18n()
  const session = loadSession()
  const staff = session?.roles.some((r) => r !== 'customer')

  return (
    <header className="bg-bar text-beige">
      {/* GRID, not a wrapping flexbox — the same bug the public masthead had:
          as a flex row the right-hand group wrapped onto a line of its own, so
          on a phone the header was a logo, then an orphaned language pill,
          before any content. Explicit placement keeps the picker beside the
          wordmark and lets only the nav take a second row. xl (1280px) is the
          same threshold the public masthead uses. */}
      <div
        className="mx-auto grid max-w-6xl grid-cols-[1fr_auto] items-center gap-x-4
                   gap-y-3 px-4 py-3 xl:grid-cols-[auto_1fr_auto]"
      >
        {/* Where the wordmark leads depends on whether there is a session.
            Signed in it is the app's own home. Signed out — the login screen
            above all — it leaves for the public site, because `to="/"` inside
            the app redirects to /menu, which requires auth and bounces
            straight back to login: clicking it there did nothing at all.

            The signed-out case is a plain <a>, not a <Link>. The public site
            is served by Go, not by the router, so a client-side navigation
            would rewrite the URL and render nothing. It also carries the
            language across — the app STORES its locale while the public site
            PREFIXES it, so without this a Chinese reader lands on the
            Indonesian home page. */}
        {session ? (
          <Link
            to="/menu"
            className="col-start-1 row-start-1 flex items-center"
            aria-label={t('nav.home_aria')}
          >
            <Wordmark />
          </Link>
        ) : (
          <a
            href={LOCALE_INFO[locale].publicPrefix}
            className="col-start-1 row-start-1 flex items-center"
            aria-label={t('nav.home_aria')}
          >
            <Wordmark />
          </a>
        )}
        {/* text-onbar, not text-sm: beige on the mid-green bar is 3.93, which is
            AA for LARGE text only (docs/10 §2.7). */}
        {session && (
          <nav
            className="col-span-full row-start-2 flex flex-wrap gap-4 text-onbar font-bold
                       xl:col-span-1 xl:col-start-2 xl:row-start-1"
            aria-label={t('nav.aria')}
          >
            <Link to="/menu">{t('nav.menu')}</Link>
            <Link to="/orders">{t('nav.orders')}</Link>
            <Link to="/packages">{t('nav.packages')}</Link>
            <Link to="/credits">{t('c06.title')}</Link>
            <Link to="/addresses">{t('nav.addresses')}</Link>
            <Link to="/keamanan">{t('nav.security')}</Link>
            {staff && <Link to="/admin">{t('bo.title')}</Link>}
            {staff && <Link to="/admin/payments">{t('nav.payments')}</Link>}
            {staff && <Link to="/admin/deliveries">{t('nav.deliveries')}</Link>}
            {staff && <Link to="/admin/settings">{t('nav.settings')}</Link>}
            {staff && <Link to="/admin/content">{t('nav.content')}</Link>}
          </nav>
        )}
        <div
          className="col-start-2 row-start-1 flex items-center justify-end gap-3
                     text-onbar font-bold xl:col-start-3"
        >
          {session ? (
            <>
              <span className="hidden md:inline">{session.full_name}</span>
              <button
                className="min-h-touch underline"
                onClick={async () => {
                  await logout()
                  window.location.href = '/app/login'
                }}
              >
                {t('auth.signout')}
              </button>
            </>
          ) : (
            <Link to="/login" className="underline">{t('auth.signin')}</Link>
          )}
          <LanguageSelector />
        </div>
      </div>
    </header>
  )
}

/** BackOffice is the staff shell — the canvas's S1 sidebar (docs/10 §4.10).
 *
 * One adaptation from the canvas, deliberate: the artboards repeat the
 * wordmark and the signed-in name inside the sidebar because each board is a
 * whole window. Here the app already has a masthead carrying the wordmark,
 * language and sign-out, so the sidebar carries navigation and the staff
 * identity only. Repeating the wordmark 22px under itself looks like a bug.
 *
 * Hidden below `lg` rather than reflowed: a 228px rail on a 390px phone leaves
 * 162px of content. Staff screens are desktop screens (the canvas draws them
 * at 1440×900) and the top bar still reaches every route on a phone.
 */
function BackOffice() {
  const t = useT()
  const session = loadSession()
  const [pending, setPending] = useState(0)
  const item = ({ isActive }: { isActive: boolean }) =>
    isActive ? 'sidenav-item-active' : 'sidenav-item'

  // S1 draws a count badge on "Pembayaran". It is live, because a badge
  // showing a stale number is worse than no badge — staff act on it.
  useEffect(() => {
    let live = true
    request<{ items: unknown[] }>('/admin/payments?status=SUBMITTED')
      .then((p) => { if (live) setPending(p.items.length) })
      .catch(() => { /* a badge must never break the shell */ })
    return () => { live = false }
  }, [])

  return (
    <div className="mx-auto flex w-full max-w-[1440px] flex-1 gap-0">
      <nav className="sidenav hidden lg:flex" aria-label={t('bo.title')}>
        {/* #CCBDAA on the rail is 2.25 — illegal at every size — so this is
            beige at 19px (docs/10 §4.1). */}
        <span className="sidenav-label mb-5">{t('bo.title')}</span>
        <div className="flex flex-col gap-0.5">
          <NavLink end to="/admin" className={item}>{t('bo.dashboard')}</NavLink>
          <NavLink to="/admin/calendar" className={item}>{t('bo.calendar')}</NavLink>
          <NavLink to="/admin/foods" className={item}>{t('bo.foods')}</NavLink>
          <NavLink to="/admin/pricing" className={item}>{t('bo.pricing')}</NavLink>
          <NavLink to="/admin/orders" className={item}>{t('bo.orders')}</NavLink>
          <NavLink to="/admin/payments" className={item}>
            {t('nav.payments')}
            {pending > 0 && <span className="pill-emph">{pending}</span>}
          </NavLink>
          <NavLink to="/admin/deliveries" className={item}>{t('nav.deliveries')}</NavLink>
          <NavLink to="/admin/coverage" className={item}>{t('bo.coverage')}</NavLink>
          <NavLink to="/admin/customers" className={item}>{t('bo.customers')}</NavLink>
          <NavLink to="/admin/packages" className={item}>{t('bo.packages')}</NavLink>
          <NavLink to="/admin/production" className={item}>{t('bo.production')}</NavLink>
          <NavLink to="/admin/labels" className={item}>{t('bo.labels')}</NavLink>
          <NavLink to="/admin/settings" className={item}>{t('nav.settings')}</NavLink>
          <NavLink to="/admin/content" className={item}>{t('nav.content')}</NavLink>
        </div>
        {session && (
          <div className="mt-auto flex flex-col gap-0.5 px-6 pt-6">
            <span className="sidenav-label mx-0">{t('bo.signed_in_as')}</span>
            <span className="on-fill text-beige">{session.full_name}</span>
          </div>
        )}
      </nav>
      <div className="min-w-0 flex-1 px-4 py-6 lg:px-8">
        <Outlet />
      </div>
    </div>
  )
}

/** The floating contact button, on CUSTOMER surfaces only.
 *
 * The canvas draws it on no back-office artboard, and on S1 it sat directly
 * over the "Dapur & wilayah" action in the needs-action list — a control
 * covered by a decoration is a defect regardless of what the design says.
 * Staff have a phone directory; a customer has this button.
 */
function CustomerOnlyFloat() {
  const { pathname } = useLocation()
  if (pathname.startsWith('/admin')) return null
  return <WhatsAppFloat />
}

function NotFound() {
  const t = useT()
  return <p>{t('route.notfound')}</p>
}

export default function App() {
  return (
    <I18nProvider>
      <CartProvider>
      <Routes>
        {/* The two print artifacts (docs/10 §4.11) render BARE — no masthead,
            no fixed footer, no floating WhatsApp button. They exist to come
            out of the kitchen printer at 05.10 every morning, and a fixed
            footer prints on top of the last table row on every page. */}
        <Route path="/admin/production"
               element={<RequireAuth><Production /></RequireAuth>} />
        <Route path="/admin/labels"
               element={<RequireAuth><PackingLabels /></RequireAuth>} />

        {/* The phone artboards (01-06, M2, M3). Mounted ABOVE Shell rather
            than inside it: while they were inside, Shell's masthead sat as a
            dead band over the design and its fixed footer printed across the
            sticky total bar. Each artboard is a whole screen. */}
        <Route element={<PhoneApp />}>
          <Route path="/menu" element={<RequireAuth><Menu /></RequireAuth>} />
          <Route path="/menu/:id" element={<RequireAuth><MealDetail /></RequireAuth>} />
          <Route path="/cart" element={<RequireAuth><Cart /></RequireAuth>} />
          <Route path="/checkout" element={<RequireAuth><Checkout /></RequireAuth>} />
          <Route path="/packages" element={<RequireAuth><Packages /></RequireAuth>} />
          <Route path="/credits" element={<RequireAuth><Credits /></RequireAuth>} />
          <Route path="/book" element={<RequireAuth><BookSlot /></RequireAuth>} />
          <Route path="/orders/:id" element={<RequireAuth><OrderDetail /></RequireAuth>} />
        </Route>

        <Route path="*" element={<Shell />} />
      </Routes>
      </CartProvider>
    </I18nProvider>
  )
}

/** Customer is the ordering shell: one readable column on the ground.
 *
 * Used by the pages that are NOT drawn as phone artboards — sign-in,
 * addresses, order history, account security. Those keep the masthead. */
function Customer() {
  return (
    <div className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
      <Outlet />
    </div>
  )
}

/** PhoneApp is the shell for artboards 01-06, M2 and M3.
 *
 * They render with NO masthead and NO fixed footer, because each artboard is a
 * whole screen: its own app bar at the top and its own action bar at the
 * bottom. Stacking the desktop masthead above one leaves a dead green band
 * over the design, and the fixed footer prints straight across the sticky
 * total — which is exactly what the first screenshots showed.
 *
 * The language selector moves into the app bar, so nothing is lost with the
 * masthead. The floating WhatsApp button is deliberately absent here: the
 * canvas does not draw it on any customer screen, and it covers the bottom
 * bar's action at 390px. It remains on every other surface.
 */
function PhoneApp() {
  return (
    <div className="flex w-full flex-1 flex-col px-3 py-4">
      <Outlet />
    </div>
  )
}

function Shell() {
  return (
    <div className="min-h-screen flex flex-col pb-[var(--footer-h)]">
      <Nav />
      {/* 2026-08-31: the app came OFF its single beige sheet. It used to render
          every screen inside one, because its controls were specced deep-on-
          light; the canvas (docs/10 §4) re-specs them for the ground, where
          beige is 11.32. Panels and framed tables carry the structure the
          sheet used to carry. */}
      <main className="flex w-full flex-1 flex-col">
        <Routes>
          <Route element={<Customer />}>
            <Route path="/" element={<Navigate to="/menu" replace />} />
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/addresses" element={<RequireAuth><Addresses /></RequireAuth>} />
            <Route path="/orders" element={<RequireAuth><Orders /></RequireAuth>} />
            <Route path="/keamanan" element={<RequireAuth><Security /></RequireAuth>} />
            <Route path="*" element={<NotFound />} />
          </Route>
          <Route element={<BackOffice />}>
            <Route path="/admin" element={<RequireAuth><AdminDashboard /></RequireAuth>} />
            <Route path="/admin/calendar" element={<RequireAuth><AdminCalendar /></RequireAuth>} />
            <Route path="/admin/pricing" element={<RequireAuth><AdminPricing /></RequireAuth>} />
            <Route path="/admin/coverage" element={<RequireAuth><AdminCoverage /></RequireAuth>} />
            <Route path="/admin/payments" element={<RequireAuth><AdminPayments /></RequireAuth>} />
            <Route path="/admin/deliveries" element={<RequireAuth><AdminDeliveries /></RequireAuth>} />
            <Route path="/admin/settings" element={<RequireAuth><AdminSettings /></RequireAuth>} />
            <Route path="/admin/content" element={<RequireAuth><AdminContent /></RequireAuth>} />
          </Route>
        </Routes>
      </main>
      <CustomerOnlyFloat />
      {/* Fixed to the bottom, in the masthead's own fill, thin (Steven,
          2026-08-18). Its height is the --footer-h token, which is also what
          the wrapper above reserves as padding. */}
      {/* Fixed, thin, and in the bar colour. Reduced to text-sm on request
          (Steven, 2026-08-18) — which puts it below AA, because beige on
          #468973 is 3.93 and that is large-text-only. Kept in step with the
          public footer; see the note in web/public/css/public.css. */}
      <footer
        className="fixed inset-x-0 bottom-0 z-40 flex h-[var(--footer-h)] items-center
                   justify-center bg-bar text-sm font-semibold text-beige"
      >
        Evermore · Jakarta
      </footer>
    </div>
  )
}
