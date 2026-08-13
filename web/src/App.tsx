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

function Nav() {
  const session = loadSession()
  const staff = session?.roles.some((r) => r !== 'customer')

  return (
    <header className="bg-nourish-deep text-beige">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-4 px-4 py-3">
        <Link to="/" className="font-display text-xl">evermore</Link>
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
    <div className="min-h-screen flex flex-col">
      <Nav />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-8">
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
      </main>
      <footer className="border-t border-nourish-deep/20 py-6 text-center text-sm text-ink-muted">
        Evermore · Jakarta
      </footer>
    </div>
  )
}
