/** The app's message catalogue.
 *
 * One place, three languages, keyed by a stable identifier — CLAUDE.md §10
 * ("message catalogues from the first string, never inline"). Mirrors
 * internal/adapter/http/messages.go, which does the same job for the
 * server-rendered public pages.
 *
 * Two things this does NOT translate, both for the same reason — they are
 * database rows in one language, not UI strings:
 *   - catalogue content: diet-type, meal and food names
 *   - server messages: API validation errors arrive already worded
 * Both are raised as docs/03 Q-24.
 */
import type { Locale } from './i18n'

type Entry = Partial<Record<Locale, string>> & { id: string }

export const messages = {
  // ── Shell ────────────────────────────────────────────────────────────────
  'nav.menu': { id: 'Menu', en: 'Menu', zh: '菜单' },
  'nav.orders': { id: 'Pesanan', en: 'Orders', zh: '订单' },
  'nav.packages': { id: 'Paket', en: 'Packages', zh: '套餐' },
  'nav.addresses': { id: 'Alamat', en: 'Addresses', zh: '地址' },
  'nav.security': { id: 'Keamanan', en: 'Security', zh: '账户安全' },
  'nav.payments': { id: 'Pembayaran', en: 'Payments', zh: '付款' },
  'nav.deliveries': { id: 'Pengiriman', en: 'Deliveries', zh: '配送' },
  'nav.settings': { id: 'Pengaturan', en: 'Settings', zh: '设置' },
  'nav.aria': { id: 'Menu', en: 'Menu', zh: '菜单' },
  'nav.home_aria': {
    id: 'Evermore — beranda',
    en: 'Evermore — home',
    zh: 'Evermore — 首页',
  },
  'auth.signout': { id: 'Keluar', en: 'Sign out', zh: '退出登录' },
  'auth.signin': { id: 'Masuk', en: 'Sign in', zh: '登录' },
  'route.notfound': {
    id: 'Halaman tidak ditemukan.',
    en: 'Page not found.',
    zh: '页面未找到。',
  },

  // ── Language selector ────────────────────────────────────────────────────
  'lang.label': { id: 'Bahasa', en: 'Language', zh: '语言' },
  'lang.choose': { id: 'Pilih bahasa', en: 'Choose a language', zh: '选择语言' },

  // ── Floating contact ─────────────────────────────────────────────────────
  'wa.aria': {
    id: 'Hubungi kami di WhatsApp',
    en: 'Contact us on WhatsApp',
    zh: '通过 WhatsApp 联系我们',
  },
} satisfies Record<string, Entry>

export type MessageKey = keyof typeof messages
