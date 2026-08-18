import { useEffect, useState } from 'react'
import { Link, Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { loadSession, logout } from './lib/api'
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
      aria-label="Hubungi kami di WhatsApp"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path fill="currentColor" d="M12.04 2C6.58 2 2.13 6.45 2.13 11.91c0 1.75.46 3.45 1.32 4.95L2 22l5.25-1.38a9.9 9.9 0 0 0 4.79 1.22h.01c5.46 0 9.91-4.45 9.91-9.91C21.96 6.45 17.5 2 12.04 2zm0 18.15h-.01a8.2 8.2 0 0 1-4.19-1.15l-.3-.18-3.12.82.83-3.04-.2-.31a8.19 8.19 0 0 1-1.26-4.38c0-4.54 3.7-8.23 8.25-8.23 2.2 0 4.27.86 5.83 2.42a8.18 8.18 0 0 1 2.41 5.82c0 4.54-3.7 8.23-8.24 8.23zm4.52-6.16c-.25-.12-1.47-.72-1.69-.81-.23-.08-.39-.12-.56.13-.16.24-.64.8-.78.97-.14.16-.29.18-.54.06-.25-.13-1.05-.39-1.99-1.23-.74-.66-1.24-1.47-1.38-1.72-.15-.25-.02-.38.11-.5.11-.11.25-.29.37-.43.13-.15.17-.25.25-.41.08-.17.04-.31-.02-.43-.06-.12-.56-1.35-.77-1.84-.2-.49-.4-.42-.56-.43h-.47c-.17 0-.43.06-.66.31-.23.25-.86.85-.86 2.07 0 1.22.89 2.4 1.01 2.56.12.17 1.75 2.67 4.23 3.74.59.26 1.05.41 1.41.52.59.19 1.13.16 1.56.1.48-.07 1.47-.6 1.68-1.18.21-.58.21-1.07.14-1.18-.06-.11-.22-.17-.47-.29z"/>
      </svg>
    </a>
  )
}

function Nav() {
  const session = loadSession()
  const staff = session?.roles.some((r) => r !== 'customer')

  return (
    <header className="bg-nourish-deep text-beige">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-4 px-4 py-3">
        <Link to="/" className="flex items-center" aria-label="Evermore — beranda">
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
        {session && (
          <nav className="flex flex-wrap gap-4 text-sm" aria-label="Menu">
            <Link to="/menu">Menu</Link>
            <Link to="/orders">Pesanan</Link>
            <Link to="/packages">Paket</Link>
            <Link to="/addresses">Alamat</Link>
            <Link to="/keamanan">Keamanan</Link>
            {staff && <Link to="/admin/payments">Pembayaran</Link>}
            {staff && <Link to="/admin/deliveries">Pengiriman</Link>}
            {staff && <Link to="/admin/settings">Pengaturan</Link>}
          </nav>
        )}
        <div className="ml-auto flex items-center gap-3 text-sm">
          {session ? (
            <>
              <span className="hidden sm:inline">{session.full_name}</span>
              <button
                className="min-h-touch underline"
                onClick={async () => {
                  await logout()
                  window.location.href = '/app/login'
                }}
              >
                Keluar
              </button>
            </>
          ) : (
            <Link to="/login" className="underline">Masuk</Link>
          )}
        </div>
      </div>
    </header>
  )
}

export default function App() {
  return (
    <div className="min-h-screen flex flex-col pb-[var(--footer-h)]">
      <Nav />
      {/* The ground is Nourish Green #468973, on which nothing reaches AA at
          reading size. The app is forms and tables, so rather than audit every
          screen the content sits on one beige sheet — deep ink on it is
          11.32:1, and the green stays visible as the frame around it. */}
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
          <Route path="*" element={<p>Halaman tidak ditemukan.</p>} />
        </Routes>
        </div>
      </main>
      <WhatsAppFloat />
      {/* Fixed to the bottom, in the masthead's own fill, thin (Steven,
          2026-08-18). Its height is the --footer-h token, which is also what
          the wrapper above reserves as padding. */}
      <footer
        className="fixed inset-x-0 bottom-0 z-40 flex h-[var(--footer-h)] items-center
                   justify-center bg-nourish-deep text-xs text-beige sm:text-sm"
      >
        Evermore · Jakarta
      </footer>
    </div>
  )
}
