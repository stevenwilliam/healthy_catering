// Screenshot the app's screens, including the ones behind a login.
//
// docs/10 §4 was read off a design canvas, and CLAUDE.md §6 says visual work is
// verified by LOOKING at it, not by reading the CSS. The public pages can be
// shot straight from the server; the signed-in screens cannot, because
// RequireAuth bounces a session-less browser to /login and `api create-staff`
// needs a database write this environment does not grant.
//
// So this puts a session in localStorage and serves the API from FIXTURES.
// What that does and does not prove, stated plainly:
//
//   PROVES  — the real React components, the real compiled stylesheet, real
//             response SHAPES. Layout, spacing, colour, the component layer,
//             overflow, and every empty/loading/error branch that the data
//             drives.
//   DOES NOT PROVE — that the server returns these shapes. That is what the Go
//             tests are for, and the two can drift. A screen that looks right
//             here can still 500 against the real API.
//
// Run:
//   export NODE_PATH=/home/dev/.npm/_npx/e41f203b7505f1fb/node_modules
//   node scripts/shoot-screens.js [baseURL] [outDir]

const { chromium } = require('playwright')
const path = require('path')
const fs = require('fs')

const BASE = process.argv[2] || 'http://127.0.0.1:8081'
const OUT = process.argv[3] || path.join(__dirname, '..', 'docs', 'screenshots')

// A session with every permission. Presentation only — the real API re-checks
// each request, so this grants nothing; it just stops the router redirecting.
const SESSION = {
  access_token: 'fixture', refresh_token: 'fixture',
  user_id: '00000000-0000-0000-0000-000000000001',
  full_name: 'Ratna Wijaya', email: 'ratna@example.co.id',
  roles: ['admin'], permissions: ['*'], email_verified: true,
}

const u = (n) => `0192${n.toString().padStart(4, '0')}-0000-7000-8000-000000000000`

const SLOTS = [
  { slot_id: u(1), alias: 'Sarapan', slot_time: '07:00' },
  { slot_id: u(2), alias: 'Makan siang', slot_time: '11:30' },
  { slot_id: u(3), alias: 'Makan siang', slot_time: '12:00' },
  { slot_id: u(4), alias: 'Makan malam', slot_time: '18:00' },
]

const KITCHENS = [
  {
    id: u(10), code: 'KTC-01', name: 'Tebet', district: 'Tebet', city: 'Jakarta Selatan',
    latitude: -6.2297, longitude: 106.8556, service_radius_km: 6.5,
    has_polygon: true, polygon_points: 18, priority: 1, is_active: true,
    slots: [
      { ...SLOTS[0], quota: 40, used: 18, available: true },
      { ...SLOTS[1], quota: 40, used: 40, available: true },
      { ...SLOTS[2], quota: 40, used: 31, available: true },
      { ...SLOTS[3], quota: 40, used: 22, available: true },
    ],
  },
  {
    id: u(11), code: 'KBY-02', name: 'Kebayoran', district: 'Kebayoran Baru', city: 'Jakarta Selatan',
    latitude: -6.2444, longitude: 106.7997, service_radius_km: 5.5,
    has_polygon: false, polygon_points: 0, priority: 2, is_active: true,
    slots: [
      { ...SLOTS[0], quota: 35, used: 12, available: true },
      { ...SLOTS[1], quota: 35, used: 33, available: true },
      { ...SLOTS[2], quota: 35, used: 28, available: true },
      { ...SLOTS[3], quota: 35, used: 19, available: true },
    ],
  },
  {
    id: u(12), code: 'KLG-03', name: 'Kelapa Gading', district: 'Kelapa Gading', city: 'Jakarta Utara',
    latitude: -6.1580, longitude: 106.9060, service_radius_km: 5.0,
    has_polygon: true, polygon_points: 24, priority: 3, is_active: true,
    slots: [
      { ...SLOTS[0], quota: 30, used: 6, available: true },
      { ...SLOTS[1], quota: 30, used: 24, available: true },
      { ...SLOTS[2], quota: 30, used: 17, available: true },
      { ...SLOTS[3], quota: 0, used: 0, available: false },
    ],
  },
]

const DIETS = {
  items: [
    { ID: u(20), Name: 'Balanced' },
    { ID: u(21), Name: 'Weight Loss' },
    { ID: u(22), Name: 'Muscle Gain' },
    { ID: u(23), Name: 'Special Diet' },
  ],
}

const PAYMENTS = {
  items: [
    { payment_id: u(30), order_code: 'EVM-2609-0131', customer_name: 'Sinta Prameswari', customer_email: 'sinta@example.co.id', expected_amount: 'Rp 480.148', unique_code: 148, status: 'SUBMITTED', waiting_minutes: 22, proof_count: 1, bank_name: 'BCA' },
    { payment_id: u(31), order_code: 'EVM-2609-0134', customer_name: 'PT Sinar Mas · Dewi', customer_email: 'dewi@sinarmas.co.id', expected_amount: 'Rp 3.408.221', unique_code: 221, status: 'SUBMITTED', waiting_minutes: 19, proof_count: 1, bank_name: 'BCA' },
    { payment_id: u(32), order_code: 'EVM-2609-0136', customer_name: 'Bagas Nugroho', customer_email: 'bagas@example.co.id', expected_amount: 'Rp 156.093', unique_code: 93, status: 'SUBMITTED', waiting_minutes: 14, proof_count: 2, bank_name: 'Mandiri' },
  ],
  total: 3, page: 1, page_size: 25,
}

/** The calendar, anchored on the Monday of the CURRENT week so the screen
 *  under test is showing the week it would really open on. */
function calendar() {
  const now = new Date()
  const day = (now.getUTCDay() + 6) % 7
  const monday = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - day, 12))
  const iso = (n) => new Date(monday.getTime() + n * 86400000).toISOString().slice(0, 10)
  const dish = (i, name, slot, status, cap, res, kcal, comps) => ({
    id: u(100 + i), service_date: iso(Math.floor(i / 2)),
    diet_type_id: DIETS.items[0].ID, diet_type: 'Balanced',
    slot_id: slot.slot_id, slot: slot.slot_time, name, status,
    qty_capacity: cap, qty_reserved: res,
    items: comps.map((f) => ({ food_name: f })),
    nutrition: { calories_kcal: kcal, protein_mg: 38200, complete: status === 'PUBLISHED' },
  })
  return [
    dish(0, 'Ayam panggang lemon & quinoa', SLOTS[1], 'PUBLISHED', 40, 40, 520, ['Ayam panggang lemon', 'Quinoa herba', 'Brokoli kukus', 'Infused water timun']),
    dish(1, 'Salmon teriyaki', SLOTS[3], 'PUBLISHED', 40, 22, 610, ['Salmon fillet', 'Nasi merah', 'Buncis', 'Infused water']),
    dish(2, 'Nasi merah rendang jamur', SLOTS[1], 'PUBLISHED', 40, 27, 540, ['Rendang jamur', 'Nasi merah', 'Urap', 'Air kelapa']),
    dish(3, 'Tumis ayam paprika', SLOTS[3], 'PUBLISHED', 40, 15, 580, ['Dada ayam paprika', 'Kentang herba', 'Salad', 'Infused water']),
    dish(4, 'Dori panggang & kentang herba', SLOTS[1], 'PUBLISHED', 40, 12, 495, ['Dori panggang', 'Kentang herba', 'Brokoli', 'Air lemon']),
    dish(5, 'Sop iga bening & ubi', SLOTS[3], 'DRAFT', 40, 0, 610, ['Iga sapi', 'Ubi kukus', 'Sayur bening', 'Air kelapa']),
    dish(6, 'Ayam kari hijau & brown rice', SLOTS[1], 'DRAFT', 40, 0, 560, ['Ayam kari hijau', 'Brown rice', 'Tumis buncis']),
  ]
}

const PRICES = {
  items: [
    { id: u(40), scope_key: 'CT:sinar-mas', scope: 'Corporate · PT Sinar Mas', tier_id: u(50), tier: 'Tier 1', price_idr: 78000, price: 'Rp 78.000', valid_from: '2026-09-01', is_active: true },
    { id: u(41), scope_key: 'CT:sinar-mas', scope: 'Corporate · PT Sinar Mas', tier_id: u(51), tier: 'Tier 2', price_idr: 75000, price: 'Rp 75.000', valid_from: '2026-09-01', is_active: true },
    { id: u(42), scope_key: 'CT:sinar-mas', scope: 'Corporate · PT Sinar Mas', tier_id: u(52), tier: 'Tier 3', price_idr: 71000, price: 'Rp 71.000', valid_from: '2026-09-01', is_active: true },
    { id: u(43), scope_key: 'CT:sinar-mas', scope: 'Corporate · PT Sinar Mas', tier_id: u(52), tier: 'Tier 3', price_idr: 69000, price: 'Rp 69.000', valid_from: '2026-07-01', valid_to: '2026-08-31', promo_label: 'Promo Agustus', is_active: false },
  ],
  total: 4, page: 1, page_size: 25,
}

const TIERS = [
  { ID: u(50), Label: 'Tier 1', MinQty: 1, MaxQty: 3, Active: true },
  { ID: u(51), Label: 'Tier 2', MinQty: 4, MaxQty: 9, Active: true },
  { ID: u(52), Label: 'Tier 3', MinQty: 10, MaxQty: null, Active: true },
]

const COVERAGE = [
  { district: 'Bintaro', city: 'Tangerang Selatan', attempts: 31, notify_requests: 12, avg_distance_to_nearest_km: 8.4, nearest_kitchen: 'KBY-02' },
  { district: 'Cibubur', city: 'Jakarta Timur', attempts: 18, notify_requests: 5, avg_distance_to_nearest_km: 14.2, nearest_kitchen: 'KTC-01' },
  { district: 'Serpong', city: 'Tangerang Selatan', attempts: 11, notify_requests: 7, avg_distance_to_nearest_km: 16.9, nearest_kitchen: 'KBY-02' },
]

const PRODUCTION = [
  ['Ayam panggang lemon', 'MAIN', '11:30', 40], ['Ayam panggang lemon', 'MAIN', '12:00', 18],
  ['Quinoa herba', 'SIDE', '11:30', 40], ['Quinoa herba', 'SIDE', '12:00', 18],
  ['Brokoli kukus', 'SIDE', '11:30', 40], ['Brokoli kukus', 'SIDE', '12:00', 39],
  ['Dada ayam paprika', 'MAIN', '11:30', 12], ['Dada ayam paprika', 'MAIN', '12:00', 9],
  ['Salmon fillet', 'MAIN', '18:00', 14], ['Nasi merah', 'SIDE', '18:00', 14],
  ['Infused water timun', 'DRINK', '11:30', 52], ['Infused water timun', 'DRINK', '18:00', 59],
].map(([food_name, item_role, slot, portions]) => ({
  service_date: '2026-09-01', slot, kitchen: 'Dapur Tebet KTC-01',
  diet_type: item_role === 'MAIN' && food_name.includes('paprika') ? 'Weight Loss' : 'Balanced',
  food_name, item_role, portions,
}))

const LABELS = [
  { delivery_id: u(60), delivery_code: 'EVM-2609-0131', service_date: '2026-09-01', slot: '11:30', kitchen_code: 'KTC-01', customer_name: 'Sinta Prameswari', phone: '0812 8899 4410', address_line: 'Jl. Wijaya IX No. 12, Petogogan', district: 'Kebayoran Baru, Jaksel 12170', diet_type: 'Balanced', qty: 1, foods: 'Ayam panggang lemon & quinoa', allergens: 'kedelai, wijen', driver_note: 'Titip resepsionis' },
  { delivery_id: u(61), delivery_code: 'EVM-2609-0136', service_date: '2026-09-01', slot: '12:00', kitchen_code: 'KTC-01', customer_name: 'Bagas Nugroho', phone: '0813 1122 8890', address_line: 'Jl. Casablanca Raya No. 88', district: 'Tebet, Jaksel 12870', diet_type: 'Weight Loss', qty: 2, foods: 'Tumis ayam paprika', allergens: '', driver_note: '' },
]

const MENU = calendar().filter((m) => m.status === 'PUBLISHED').map((m) => ({
  ...m, diet_type_id: DIETS.items[0].ID,
}))

const ADDRESSES = [
  { ID: u(70), Label: 'Rumah', AddressLine: 'Jl. Wijaya IX No. 12, Petogogan' },
]

const QUOTE = {
  unit_price: 'Rp 75.000', line_total: 'Rp 450.000', normal_price: 'Rp 78.000',
  is_promo: false, tier: 'Tier 2 · 4–9', savings: 'Rp 18.000',
}

/** Route the SPA's API calls to the fixtures above. Anything not matched
 *  returns an empty array rather than hanging, so a screen that asks for
 *  something unlisted renders its empty state instead of its spinner. */
async function stubAPI(page) {
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url())
    const p = url.pathname.replace(/^\/api\/v1/, '')
    const send = (data) => route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify(p === '/public/company' ? data : { data }),
    })

    if (p === '/public/company') return send({ name: 'Evermore', whatsapp: '628118889000' })
    if (p === '/admin/kitchens') return send(KITCHENS)
    if (p === '/admin/payments') return send(PAYMENTS)
    if (p === '/admin/diet-types') return send(DIETS)
    if (p === '/admin/calendar') return send(calendar())
    if (p.startsWith('/admin/prices/')) return send(PRICES)
    if (p === '/admin/price-tiers') return send(TIERS)
    if (p === '/admin/reports/coverage') return send(COVERAGE)
    if (p === '/admin/reports/production') return send(PRODUCTION)
    if (p === '/admin/reports/packing-labels') return send(LABELS)
    if (p === '/quote') return send(QUOTE)
    if (p === '/menu') return send(MENU)
    if (p === '/addresses') return send(ADDRESSES)
    return send([])
  })
}

const SHOTS = [
  // Public — real server, real data, no stubbing.
  { name: 'public-home-desktop', url: '/', w: 1440, h: 1400, live: true },
  { name: 'public-home-mobile', url: '/', w: 390, h: 1600, live: true },
  { name: 'spa-login', url: '/app/login', w: 1280, h: 900, live: true },
  // Signed-in — real components, fixture data.
  { name: 'spa-dashboard', url: '/app/admin', w: 1440, h: 1000 },
  { name: 'spa-calendar', url: '/app/admin/calendar', w: 1440, h: 1000 },
  { name: 'spa-pricing', url: '/app/admin/pricing', w: 1440, h: 1100 },
  { name: 'spa-coverage', url: '/app/admin/coverage', w: 1440, h: 1400 },
  { name: 'spa-production', url: '/app/admin/production', w: 1000, h: 1500 },
  { name: 'spa-labels', url: '/app/admin/labels', w: 1000, h: 900 },
  { name: 'spa-menu', url: '/app/menu', w: 1280, h: 1200 },
  { name: 'spa-menu-mobile', url: '/app/menu', w: 390, h: 1200 },
]

;(async () => {
  fs.mkdirSync(OUT, { recursive: true })
  const browser = await chromium.launch()
  let failures = 0

  for (const s of SHOTS) {
    const ctx = await browser.newContext({
      viewport: { width: s.w, height: s.h },
      deviceScaleFactor: 2,
    })
    const page = await ctx.newPage()

    // Console and page errors are the point of running a browser at all — a
    // screen that renders but throws is not working.
    const errors = []
    page.on('pageerror', (e) => errors.push(String(e)))
    page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })

    if (!s.live) {
      await stubAPI(page)
      await ctx.addInitScript((sess) => {
        localStorage.setItem('evermore.session', sess)
      }, JSON.stringify(SESSION))
    }

    await page.goto(BASE + s.url, { waitUntil: 'networkidle' })
    // Fonts must be in before the shot or every heading is measured in the
    // fallback serif and the layout in the picture is not the shipped one.
    await page.evaluate(() => document.fonts.ready)
    await page.waitForTimeout(400)

    const file = path.join(OUT, `${s.name}.png`)
    await page.screenshot({ path: file, fullPage: true })

    const noisy = errors.filter((e) => !/favicon|Failed to load resource/i.test(e))
    if (noisy.length) {
      failures++
      console.log(`✗ ${s.name}`)
      for (const e of noisy.slice(0, 4)) console.log(`    ${e}`)
    } else {
      console.log(`✓ ${s.name}  ${path.relative(process.cwd(), file)}`)
    }
    await ctx.close()
  }

  await browser.close()
  if (failures) {
    console.log(`\n${failures} screen(s) logged errors.`)
    process.exit(1)
  }
})()
