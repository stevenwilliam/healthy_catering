import { useEffect, useState } from 'react'
import { Link, Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { loadSession, logout } from './lib/api'
import { I18nProvider, useT } from './lib/i18n'
import LanguageSelector from './components/LanguageSelector'
import Login from './pages/Login'
import Register from './pages/Register'
import Menu from './pages/Menu'
import Addresses from './pages/Addresses'
import Orders from './pages/Orders'
import OrderDetail from './pages/OrderDetail'
import Packages from './pages/Packages'
import AdminPayments from './pages/AdminPayments'
import AdminDeliveries from './pages/AdminDeliveries'
import AdminSettings from './pages/AdminSettings'
import AdminContent from './pages/AdminContent'
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

function Nav() {
  const t = useT()
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
        <Link
          to="/"
          className="col-start-1 row-start-1 flex items-center"
          aria-label={t('nav.home_aria')}
        >
          {/* The supplied wordmark, reversed out for the dark fill. Served
              from /images by the Go server, not bundled by Vite. */}
          <img
            src="/images/evermore-wordmark-light.png"
            width={560}
            height={60}
            alt="Evermore"
            className="block h-6 w-auto sm:h-7"
          />
        </Link>
        {/* text-bar, not text-sm: beige on the mid-green bar is 3.93, which is
            AA for LARGE text only (docs/10 §2.7). */}
        {session && (
          <nav
            className="col-span-full row-start-2 flex flex-wrap gap-4 text-bar font-bold
                       xl:col-span-1 xl:col-start-2 xl:row-start-1"
            aria-label={t('nav.aria')}
          >
            <Link to="/menu">{t('nav.menu')}</Link>
            <Link to="/orders">{t('nav.orders')}</Link>
            <Link to="/packages">{t('nav.packages')}</Link>
            <Link to="/addresses">{t('nav.addresses')}</Link>
            <Link to="/keamanan">{t('nav.security')}</Link>
            {staff && <Link to="/admin/payments">{t('nav.payments')}</Link>}
            {staff && <Link to="/admin/deliveries">{t('nav.deliveries')}</Link>}
            {staff && <Link to="/admin/settings">{t('nav.settings')}</Link>}
            {staff && <Link to="/admin/content">{t('nav.content')}</Link>}
          </nav>
        )}
        <div
          className="col-start-2 row-start-1 flex items-center justify-end gap-3
                     text-bar font-bold xl:col-start-3"
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

function NotFound() {
  const t = useT()
  return <p>{t('route.notfound')}</p>
}

export default function App() {
  return (
    <I18nProvider>
      <Shell />
    </I18nProvider>
  )
}

function Shell() {
  return (
    <div className="min-h-screen flex flex-col pb-[var(--footer-h)]">
      <Nav />
      {/* The page ground is deep #1C3D34 since the colour swap. The app is
          forms and tables, and its controls were all designed against a light
          surface, so the content keeps its one beige sheet — deep ink on it is
          11.32:1 — and the ground frames it. */}
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
        <div className="rounded-2xl bg-sheet px-4 py-6 text-ink shadow-lift sm:px-8 sm:py-8">
        <Routes>
          <Route path="/" element={<Navigate to="/menu" replace />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/menu" element={<RequireAuth><Menu /></RequireAuth>} />
          <Route path="/addresses" element={<RequireAuth><Addresses /></RequireAuth>} />
          <Route path="/orders" element={<RequireAuth><Orders /></RequireAuth>} />
          <Route path="/orders/:id" element={<RequireAuth><OrderDetail /></RequireAuth>} />
          <Route path="/packages" element={<RequireAuth><Packages /></RequireAuth>} />
          <Route path="/keamanan" element={<RequireAuth><Security /></RequireAuth>} />
          <Route path="/admin/payments" element={<RequireAuth><AdminPayments /></RequireAuth>} />
          <Route path="/admin/deliveries" element={<RequireAuth><AdminDeliveries /></RequireAuth>} />
          <Route path="/admin/settings" element={<RequireAuth><AdminSettings /></RequireAuth>} />
          <Route path="/admin/content" element={<RequireAuth><AdminContent /></RequireAuth>} />
          <Route path="*" element={<NotFound />} />
        </Routes>
        </div>
      </main>
      <WhatsAppFloat />
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
