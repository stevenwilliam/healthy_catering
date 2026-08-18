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

/** Every locale is REQUIRED, so a half-translated key is a compile error
 *  rather than a string that quietly renders in the wrong language. This is
 *  the type-level equivalent of TestPublicMessagesComplete on the server. */
type Entry = Record<Locale, string>

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

  // ── Shared field labels ──────────────────────────────────────────────────
  'field.email': { id: 'Email', en: 'Email', zh: '邮箱' },
  'field.password': { id: 'Kata sandi', en: 'Password', zh: '密码' },

  // ── Shared UI ────────────────────────────────────────────────────────────
  'ui.search': { id: 'Cari', en: 'Search', zh: '搜索' },
  'ui.search_placeholder': {
    id: 'Ketik untuk menyaring…', en: 'Type to filter…', zh: '输入以筛选…',
  },
  // Rendered as "12 results" — a suffix rather than a formatted sentence,
  // which keeps it correct in all three without a plural rule engine.
  'ui.results': { id: 'hasil', en: 'results', zh: '条结果' },
  'ui.processing': { id: 'Memproses…', en: 'Processing…', zh: '处理中…' },
  'ui.loading': { id: 'Memuat…', en: 'Loading…', zh: '加载中…' },
  'ui.empty': { id: 'Belum ada data.', en: 'Nothing here yet.', zh: '暂无数据。' },
  'ui.yes': { id: 'Ya', en: 'Yes', zh: '是' },
  'ui.cancel': { id: 'Batal', en: 'Cancel', zh: '取消' },
  'ui.copy': { id: 'Salin', en: 'Copy', zh: '复制' },
  'ui.copied': { id: 'Disalin', en: 'Copied', zh: '已复制' },
  'ui.copy_failed': {
    id: 'Tidak dapat menyalin — salin manual',
    en: 'Could not copy — please copy manually',
    zh: '无法复制——请手动复制',
  },

  // ── Sign in ──────────────────────────────────────────────────────────────
  'login.title': { id: 'Masuk', en: 'Sign in', zh: '登录' },
  'login.submit': { id: 'Masuk', en: 'Sign in', zh: '登录' },
  'login.failed': { id: 'Tidak dapat masuk.', en: 'Could not sign in.', zh: '无法登录。' },
  'login.no_account': {
    id: 'Belum punya akun?', en: "Don't have an account?", zh: '还没有账号？',
  },
  'login.register_link': { id: 'Daftar', en: 'Sign up', zh: '注册' },

  'mfa.title': { id: 'Verifikasi dua langkah', en: 'Two-step verification', zh: '两步验证' },
  'mfa.intro': {
    id: 'Masukkan kode enam digit dari aplikasi autentikator Anda.',
    en: 'Enter the six-digit code from your authenticator app.',
    zh: '请输入身份验证器应用中的六位验证码。',
  },
  'mfa.code': { id: 'Kode', en: 'Code', zh: '验证码' },
  'mfa.recovery_hint': {
    id: 'Kehilangan ponsel? Masukkan salah satu kode pemulihan Anda di sini.',
    en: 'Lost your phone? Enter one of your recovery codes here.',
    zh: '手机丢失？请在此输入您的恢复码之一。',
  },
  'mfa.verify': { id: 'Verifikasi', en: 'Verify', zh: '验证' },
  'mfa.back': { id: 'Kembali', en: 'Back', zh: '返回' },
  'mfa.failed': {
    id: 'Kode tidak dapat diverifikasi.',
    en: 'The code could not be verified.',
    zh: '无法验证该验证码。',
  },

  // ── Sign up ──────────────────────────────────────────────────────────────
  'register.title': { id: 'Daftar', en: 'Sign up', zh: '注册' },
  'register.name': { id: 'Nama lengkap', en: 'Full name', zh: '姓名' },
  'register.phone': { id: 'Nomor HP', en: 'Mobile number', zh: '手机号码' },
  'register.password_hint': {
    id: 'Minimal 12 karakter.', en: 'At least 12 characters.', zh: '至少 12 个字符。',
  },
  'register.password_short': {
    id: 'Gunakan minimal 12 karakter.', en: 'Use at least 12 characters.', zh: '请使用至少 12 个字符。',
  },
  'register.submit': { id: 'Daftar', en: 'Sign up', zh: '注册' },
  'register.failed': { id: 'Pendaftaran gagal.', en: 'Registration failed.', zh: '注册失败。' },
  'register.done_title': { id: 'Cek email Anda', en: 'Check your email', zh: '请查收邮件' },
  'register.done_body': {
    id: 'Kami mengirim tautan konfirmasi. Konfirmasi email sebelum pesanan pertama Anda.',
    en: "We've sent a confirmation link. Confirm your email before your first order.",
    zh: '我们已发送确认链接。请在首次下单前确认您的邮箱。',
  },
  'register.back_to_login': {
    id: 'Kembali ke halaman masuk', en: 'Back to sign in', zh: '返回登录页面',
  },

  // ── Menu ─────────────────────────────────────────────────────────────────
  'menu.title': { id: 'Menu minggu ini', en: "This week's menu", zh: '本周菜单' },
  'menu.verify_email': {
    id: 'Konfirmasi email Anda sebelum pesanan pertama. Kami sudah mengirim tautannya ke',
    en: "Confirm your email before your first order. We've sent the link to",
    zh: '请在首次下单前确认您的邮箱。我们已将链接发送至',
  },
  'menu.search_placeholder': {
    id: 'Cari lauk, mis. ayam', en: 'Search dishes, e.g. chicken', zh: '搜索菜品，例如鸡肉',
  },
  'menu.empty': {
    id: 'Menu belum dipublikasikan untuk minggu ini.',
    en: "This week's menu hasn't been published yet.",
    zh: '本周菜单尚未发布。',
  },
  'menu.kcal': { id: 'kkal', en: 'kcal', zh: '千卡' },
  'menu.protein': { id: 'g protein', en: 'g protein', zh: '克蛋白质' },
  'menu.estimated': { id: 'perkiraan', en: 'estimated', zh: '估算值' },
  'menu.sold_out': { id: 'Habis untuk tanggal ini.', en: 'Sold out for this date.', zh: '该日期已售罄。' },
  'menu.qty': { id: 'Jumlah', en: 'Quantity', zh: '数量' },
  'menu.summary_aria': { id: 'Ringkasan pesanan', en: 'Order summary', zh: '订单摘要' },
  'menu.portions': { id: 'porsi', en: 'portions', zh: '份' },
  'menu.per_portion': { id: 'per porsi', en: 'per portion', zh: '每份' },
  'menu.tier': { id: 'tarif', en: 'tier', zh: '价格档' },
  'menu.savings': { id: 'hemat', en: 'save', zh: '节省' },
  'menu.deliver_to': { id: 'Antar ke', en: 'Deliver to', zh: '配送至' },
  'menu.need_address': {
    id: 'Tambahkan alamat pengiriman dulu — kami perlu titik peta Anda.',
    en: 'Add a delivery address first — we need your map pin.',
    zh: '请先添加配送地址——我们需要您的地图坐标。',
  },
  'menu.order_now': { id: 'Pesan sekarang', en: 'Order now', zh: '立即下单' },
  'menu.load_failed': { id: 'Gagal memuat menu.', en: 'Could not load the menu.', zh: '无法加载菜单。' },
  'menu.order_failed': {
    id: 'Tidak dapat membuat pesanan.', en: 'Could not place the order.', zh: '无法创建订单。',
  },

  // ── Orders ───────────────────────────────────────────────────────────────
  'orders.title': { id: 'Pesanan saya', en: 'My orders', zh: '我的订单' },
  'orders.search_placeholder': {
    id: 'Cari kode pesanan atau status', en: 'Search order code or status', zh: '搜索订单编号或状态',
  },
  'orders.empty': { id: 'Belum ada pesanan.', en: 'No orders yet.', zh: '暂无订单。' },
  'orders.load_failed': { id: 'Gagal memuat pesanan.', en: 'Could not load orders.', zh: '无法加载订单。' },
  'orders.package': { id: 'Paket', en: 'Package', zh: '套餐' },
  'orders.deliveries': { id: 'pengiriman', en: 'deliveries', zh: '次配送' },

  'status.awaiting_payment': { id: 'Menunggu pembayaran', en: 'Awaiting payment', zh: '待付款' },
  'status.payment_submitted': { id: 'Bukti terkirim', en: 'Proof submitted', zh: '凭证已提交' },
  'status.paid': { id: 'Lunas', en: 'Paid', zh: '已付款' },
  'status.completed': { id: 'Selesai', en: 'Completed', zh: '已完成' },
  'status.expired': { id: 'Kedaluwarsa', en: 'Expired', zh: '已过期' },
  'status.cancelled': { id: 'Dibatalkan', en: 'Cancelled', zh: '已取消' },
  'status.refunded': { id: 'Dikembalikan', en: 'Refunded', zh: '已退款' },

  // ── One order ────────────────────────────────────────────────────────────
  'order.load_failed': { id: 'Gagal memuat pesanan.', en: 'Could not load the order.', zh: '无法加载订单。' },
  'order.upload_failed': { id: 'Unggahan gagal.', en: 'Upload failed.', zh: '上传失败。' },
  'order.how_to_pay': { id: 'Cara membayar', en: 'How to pay', zh: '付款方式' },
  'order.transfer_exactly': {
    id: 'Transfer tepat sebesar:', en: 'Transfer exactly:', zh: '请转账准确金额：',
  },
  'order.unique_code_note': {
    id: 'Tiga digit terakhir adalah kode unik Anda. Mohon jangan dibulatkan — angka itulah yang kami pakai untuk mencocokkan pembayaran Anda.',
    en: 'The last three digits are your unique code. Please do not round it — that is how we match your payment.',
    zh: '末尾三位是您的专属识别码。请勿四舍五入——我们依靠它核对您的付款。',
  },
  'order.bank': { id: 'Bank', en: 'Bank', zh: '银行' },
  'order.copy_account': { id: 'Salin nomor rekening', en: 'Copy account number', zh: '复制账号' },
  'order.account_holder': { id: 'a.n.', en: 'Account name:', zh: '户名：' },
  'order.upload_proof': { id: 'Unggah bukti transfer', en: 'Upload transfer proof', zh: '上传转账凭证' },
  'order.proof_formats': {
    id: 'JPEG, PNG, WebP atau PDF, maksimal 5 MB.',
    en: 'JPEG, PNG, WebP or PDF, up to 5 MB.',
    zh: 'JPEG、PNG、WebP 或 PDF，最大 5 MB。',
  },
  'order.send_proof': { id: 'Kirim bukti', en: 'Send proof', zh: '提交凭证' },
  'order.lines': { id: 'Rincian', en: 'Items', zh: '订单明细' },
  'order.total': { id: 'Total', en: 'Total', zh: '合计' },
  'order.deliveries': { id: 'Pengiriman', en: 'Deliveries', zh: '配送' },

  // ── Packages ─────────────────────────────────────────────────────────────
  'packages.title': { id: 'Paket kredit', en: 'Meal-credit packages', zh: '餐额套餐' },
  'packages.mine': { id: 'Paket saya', en: 'My packages', zh: '我的套餐' },
  // Sold as days, not credits (Steven, 2026-08-18). The label only — the
  // balance underneath is still meal credits, which is what a booking spends.
  'packages.credits': { id: 'hari', en: 'days', zh: '天' },
  'packages.valid_until': { id: 'berlaku sampai', en: 'valid until', zh: '有效期至' },
  // "Usage" rather than "credit history": the card above it now counts days,
  // and two different words for the same number on one screen is worse than
  // either word alone.
  'packages.ledger': { id: 'Riwayat pemakaian', en: 'Usage history', zh: '使用记录' },
  'packages.col_time': { id: 'Waktu', en: 'Time', zh: '时间' },
  'packages.col_type': { id: 'Jenis', en: 'Type', zh: '类型' },
  'packages.col_change': { id: 'Perubahan', en: 'Change', zh: '变动' },
  'packages.col_balance': { id: 'Saldo', en: 'Balance', zh: '余额' },
  'packages.col_note': { id: 'Catatan', en: 'Note', zh: '备注' },
  'packages.buy': { id: 'Beli paket', en: 'Buy a package', zh: '购买套餐' },
  'packages.buy_button': { id: 'Beli', en: 'Buy', zh: '购买' },
  'packages.search_placeholder': { id: 'Cari paket', en: 'Search packages', zh: '搜索套餐' },
  'packages.load_failed': { id: 'Gagal memuat paket.', en: 'Could not load packages.', zh: '无法加载套餐。' },
  'packages.buy_failed': { id: 'Pembelian gagal.', en: 'Purchase failed.', zh: '购买失败。' },

  // ── Addresses ────────────────────────────────────────────────────────────
  'addresses.title': { id: 'Alamat pengiriman', en: 'Delivery addresses', zh: '配送地址' },
  'addresses.empty': {
    id: 'Belum ada alamat tersimpan.', en: 'No addresses saved yet.', zh: '尚未保存地址。',
  },
  'addresses.add': { id: 'Tambah alamat', en: 'Add an address', zh: '添加地址' },
  'addresses.label': { id: 'Label', en: 'Label', zh: '标签' },
  'addresses.label_placeholder': { id: 'Rumah', en: 'Home', zh: '家' },
  'addresses.recipient': { id: 'Nama penerima', en: 'Recipient name', zh: '收件人姓名' },
  'addresses.recipient_phone': { id: 'Nomor HP penerima', en: 'Recipient mobile', zh: '收件人手机号' },
  'addresses.district': { id: 'Kecamatan', en: 'District', zh: '区' },
  'addresses.line': { id: 'Alamat lengkap', en: 'Full address', zh: '详细地址' },
  'addresses.note': { id: 'Catatan untuk kurir', en: 'Note for the courier', zh: '给配送员的备注' },
  'addresses.note_placeholder': {
    id: 'pagar abu-abu, bel di kanan',
    en: 'grey gate, bell on the right',
    zh: '灰色大门，门铃在右侧',
  },
  'addresses.save': { id: 'Simpan alamat', en: 'Save address', zh: '保存地址' },
  'addresses.load_failed': { id: 'Gagal memuat alamat.', en: 'Could not load addresses.', zh: '无法加载地址。' },
  'addresses.save_failed': {
    id: 'Tidak dapat menyimpan alamat.', en: 'Could not save the address.', zh: '无法保存地址。',
  },

  // ── Security ─────────────────────────────────────────────────────────────
  'security.title': { id: 'Keamanan', en: 'Security', zh: '账户安全' },
  'security.unavailable': {
    id: 'Verifikasi dua langkah belum dikonfigurasi di server ini. Hubungi administrator sistem.',
    en: 'Two-step verification is not configured on this server. Contact your system administrator.',
    zh: '此服务器尚未配置两步验证。请联系系统管理员。',
  },
  'security.intro': {
    id: 'Verifikasi dua langkah menambahkan kode dari ponsel Anda saat masuk.',
    en: 'Two-step verification adds a code from your phone when you sign in.',
    zh: '两步验证会在登录时额外要求手机上的验证码。',
  },
  'security.required': {
    id: 'Untuk peran Anda, ini wajib.',
    en: 'It is mandatory for your role.',
    zh: '对于您的角色，这是强制要求。',
  },
  'security.save_recovery': {
    id: 'Simpan kode pemulihan Anda', en: 'Save your recovery codes', zh: '请保存您的恢复码',
  },
  'security.recovery_note': {
    id: 'Setiap kode hanya dapat dipakai satu kali. Ini satu-satunya kali kode ditampilkan — tanpa kode ini, ponsel yang hilang berarti akun yang hilang.',
    en: 'Each code works once. This is the only time they are shown — without them, a lost phone means a lost account.',
    zh: '每个恢复码仅可使用一次。这是唯一一次显示——若没有它们，手机丢失即意味着账号丢失。',
  },
  'security.copy_all': { id: 'Salin semua', en: 'Copy all', zh: '全部复制' },
  'security.on': { id: 'Aktif.', en: 'On.', zh: '已启用。' },
  'security.codes_left': {
    id: 'kode pemulihan tersisa.', en: 'recovery codes left.', zh: '个恢复码剩余。',
  },
  'security.locked_on': {
    id: 'Verifikasi dua langkah wajib untuk peran Anda dan tidak dapat dimatikan.',
    en: 'Two-step verification is mandatory for your role and cannot be turned off.',
    zh: '两步验证对您的角色为强制启用，无法关闭。',
  },
  'security.password_to_disable': {
    id: 'Masukkan kata sandi untuk menonaktifkan',
    en: 'Enter your password to turn it off',
    zh: '请输入密码以关闭',
  },
  'security.disable': { id: 'Nonaktifkan', en: 'Turn off', zh: '关闭' },
  'security.step1': { id: 'Langkah 1 — pindai atau ketik', en: 'Step 1 — scan or type', zh: '第 1 步 — 扫描或输入' },
  'security.step1_hint': {
    id: 'Tambahkan ini ke Google Authenticator, Authy atau sejenisnya.',
    en: 'Add this to Google Authenticator, Authy or similar.',
    zh: '请将其添加到 Google Authenticator、Authy 或类似应用。',
  },
  'security.step2': { id: 'Langkah 2 — buktikan', en: 'Step 2 — prove it', zh: '第 2 步 — 验证' },
  'security.six_digit': { id: 'Kode enam digit', en: 'Six-digit code', zh: '六位验证码' },
  'security.nothing_changes': {
    id: 'Tidak ada yang berubah sampai kode ini benar.',
    en: 'Nothing changes until this code is correct.',
    zh: '在此验证码正确之前，不会有任何更改。',
  },
  'security.enable': { id: 'Aktifkan', en: 'Turn on', zh: '启用' },
  'security.turn_on': {
    id: 'Aktifkan verifikasi dua langkah', en: 'Turn on two-step verification', zh: '启用两步验证',
  },
  'security.status_failed': {
    id: 'Tidak dapat memuat status.', en: 'Could not load the status.', zh: '无法加载状态。',
  },
  'security.start_failed': {
    id: 'Tidak dapat memulai pendaftaran.', en: 'Could not start enrolment.', zh: '无法开始绑定。',
  },
  'security.disable_failed': {
    id: 'Tidak dapat menonaktifkan.', en: 'Could not turn it off.', zh: '无法关闭。',
  },

  // ── Admin: payments ──────────────────────────────────────────────────────
  'pay.title': { id: 'Antrean pembayaran', en: 'Payment queue', zh: '付款审核队列' },
  'pay.oldest_first': {
    id: 'Diurutkan dari yang paling lama menunggu.',
    en: 'Sorted by who has waited longest.',
    zh: '按等待时间最长优先排序。',
  },
  'pay.search_placeholder': {
    id: 'Cari kode pesanan, nama, jumlah',
    en: 'Search order code, name, amount',
    zh: '搜索订单编号、姓名、金额',
  },
  'pay.empty': {
    id: 'Tidak ada pembayaran menunggu verifikasi.',
    en: 'No payments awaiting verification.',
    zh: '没有待审核的付款。',
  },
  'pay.waiting': { id: 'menunggu', en: 'waiting', zh: '已等待' },
  'pay.minutes': { id: 'menit', en: 'minutes', zh: '分钟' },
  'pay.proofs': { id: 'bukti', en: 'proofs', zh: '份凭证' },
  'pay.unique_code': { id: 'kode unik', en: 'unique code', zh: '识别码' },
  'pay.view_proof': { id: 'Lihat bukti', en: 'View proof', zh: '查看凭证' },
  'pay.verify': { id: 'Verifikasi', en: 'Verify', zh: '通过' },
  'pay.reject': { id: 'Tolak', en: 'Reject', zh: '驳回' },
  'pay.reject_reason': { id: 'Alasan penolakan', en: 'Reason for rejection', zh: '驳回原因' },
  'pay.load_failed': { id: 'Gagal memuat antrean.', en: 'Could not load the queue.', zh: '无法加载队列。' },
  'pay.verify_failed': { id: 'Verifikasi gagal.', en: 'Verification failed.', zh: '审核失败。' },
  'pay.reject_failed': { id: 'Penolakan gagal.', en: 'Rejection failed.', zh: '驳回失败。' },

  // ── Admin: deliveries ────────────────────────────────────────────────────
  'deliv.title': { id: 'Pengiriman', en: 'Deliveries', zh: '配送' },
  'deliv.date': { id: 'Tanggal', en: 'Date', zh: '日期' },
  'deliv.search_placeholder': {
    id: 'Cari kode, nama, alamat', en: 'Search code, name, address', zh: '搜索编号、姓名、地址',
  },
  'deliv.empty': {
    id: 'Tidak ada pengiriman untuk tanggal ini.',
    en: 'No deliveries for this date.',
    zh: '该日期没有配送。',
  },
  'deliv.manual': { id: 'dipindah manual', en: 'moved manually', zh: '人工调整' },
  'deliv.start_cooking': { id: 'Mulai masak', en: 'Start cooking', zh: '开始备餐' },
  'deliv.depart': { id: 'Berangkat', en: 'Out for delivery', zh: '已出发' },
  'deliv.delivered': { id: 'Terkirim', en: 'Delivered', zh: '已送达' },
  'deliv.failed': { id: 'Gagal', en: 'Failed', zh: '配送失败' },
  'deliv.no_next': {
    id: 'Tidak ada tindakan berikutnya.', en: 'No further action.', zh: '无后续操作。',
  },
  'deliv.load_failed': {
    id: 'Gagal memuat pengiriman.', en: 'Could not load deliveries.', zh: '无法加载配送。',
  },
  'deliv.status_failed': {
    id: 'Tidak dapat mengubah status.', en: 'Could not change the status.', zh: '无法更改状态。',
  },

  // ── Admin: settings ──────────────────────────────────────────────────────
  'set.title': { id: 'Pengaturan', en: 'Settings', zh: '设置' },
  'set.audit_note': {
    id: 'Setiap perubahan tercatat di audit log: siapa, kapan, dari apa, menjadi apa, dan alasannya.',
    en: 'Every change is recorded in the audit log: who, when, from what, to what, and why.',
    zh: '每次更改都会记入审计日志：操作人、时间、原值、新值及原因。',
  },
  'set.search_placeholder': {
    id: 'Cari kunci, label atau grup', en: 'Search key, label or group', zh: '搜索键名、标签或分组',
  },
  'set.secret': { id: 'rahasia', en: 'secret', zh: '机密' },
  'set.new_value': { id: 'Nilai baru untuk', en: 'New value for', zh: '新值：' },
  'set.reason': {
    id: 'Alasan perubahan (opsional)', en: 'Reason for the change (optional)', zh: '更改原因（可选）',
  },
  'set.save': { id: 'Simpan', en: 'Save', zh: '保存' },
  'set.load_failed': {
    id: 'Gagal memuat pengaturan.', en: 'Could not load settings.', zh: '无法加载设置。',
  },
  'set.save_failed': { id: 'Gagal menyimpan.', en: 'Could not save.', zh: '保存失败。' },

  // ── Admin: public content ────────────────────────────────────────────────
  'nav.content': { id: 'Konten', en: 'Content', zh: '内容' },
  'content.title': { id: 'Konten halaman depan', en: 'Home page content', zh: '首页内容' },
  'content.intro': {
    id: 'Tulis dalam Bahasa Indonesia. Bahasa Inggris dan Mandarin dibuat otomatis dari teks itu, dan bisa Anda timpa sendiri bila hasilnya kurang tepat.',
    en: 'Write in Indonesian. English and Chinese are produced from it automatically, and you can override either when the machine gets it wrong.',
    zh: '请用印尼语撰写。英文和中文将据此自动生成，如机器翻译不准确，您可以自行覆盖。',
  },
  'content.auto_on': { id: 'Terjemahan otomatis aktif', en: 'Auto-translation on', zh: '自动翻译已开启' },
  'content.auto_off': { id: 'Terjemahan otomatis nonaktif', en: 'Auto-translation off', zh: '自动翻译已关闭' },
  'content.auto_on_hint': {
    id: 'Menyimpan teks Indonesia akan memperbarui bahasa lain, kecuali yang Anda timpa.',
    en: 'Saving the Indonesian refreshes the other languages, except the ones you have overridden.',
    zh: '保存印尼语文本会刷新其他语言，但不会覆盖您手动修改的内容。',
  },
  'content.auto_off_hint': {
    id: 'Belum ada penyedia terjemahan yang dikonfigurasi, jadi Inggris dan Mandarin harus diisi manual.',
    en: 'No translation provider is configured, so English and Chinese must be written by hand.',
    zh: '尚未配置翻译服务，因此英文和中文需要手动填写。',
  },
  'content.retranslate_all': {
    id: 'Terjemahkan ulang semua', en: 'Re-translate everything', zh: '重新翻译全部',
  },
  'content.search_placeholder': {
    id: 'Cari kunci atau teks', en: 'Search key or text', zh: '搜索键名或文本',
  },
  'content.source_label': {
    id: 'Bahasa Indonesia (sumber)', en: 'Indonesian (source)', zh: '印尼语（源语言）',
  },
  'content.save_source': { id: 'Simpan sumber', en: 'Save source', zh: '保存源文本' },
  'content.save_override': { id: 'Simpan sebagai timpaan', en: 'Save as override', zh: '保存为覆盖' },
  'content.release': {
    id: 'Lepas ke terjemahan otomatis', en: 'Release to auto-translation', zh: '恢复自动翻译',
  },
  'content.override': { id: 'ditimpa manual', en: 'overridden', zh: '手动覆盖' },
  'content.stale': {
    id: 'teks Indonesia sudah berubah', en: 'Indonesian has changed since', zh: '印尼语原文已更改',
  },
  'content.empty': { id: 'kosong', en: 'empty', zh: '空' },
  'content.empty_hint': {
    id: 'Pengunjung akan melihat teks Bahasa Indonesia.',
    en: 'Visitors see the Indonesian text instead.',
    zh: '访客将看到印尼语文本。',
  },
  'content.out_translated': { id: 'diterjemahkan', en: 'translated', zh: '已翻译' },
  'content.out_kept': {
    id: 'timpaan Anda dipertahankan', en: 'your override was kept', zh: '保留了您的覆盖内容',
  },
  'content.out_no_translator': {
    id: 'tidak ada penerjemah', en: 'no translator configured', zh: '未配置翻译服务',
  },
  'content.out_failed': { id: 'terjemahan gagal', en: 'translation failed', zh: '翻译失败' },
  'content.out_error': { id: 'gagal', en: 'error', zh: '出错' },
  'content.load_failed': {
    id: 'Gagal memuat konten.', en: 'Could not load the content.', zh: '无法加载内容。',
  },
  'content.save_failed': { id: 'Gagal menyimpan.', en: 'Could not save.', zh: '保存失败。' },
  // ── Rich-text editor ─────────────────────────────────────────────────────
  'rich.toolbar': { id: 'Format teks', en: 'Text formatting', zh: '文本格式' },
  'rich.bold': { id: 'Tebal', en: 'Bold', zh: '加粗' },
  'rich.italic': { id: 'Miring', en: 'Italic', zh: '斜体' },
  'rich.bullets': { id: 'Daftar poin', en: 'Bulleted list', zh: '项目符号列表' },
  'rich.numbers': { id: 'Daftar bernomor', en: 'Numbered list', zh: '编号列表' },
  'rich.link': { id: 'Tautan', en: 'Link', zh: '链接' },
  'rich.clear': { id: 'Hapus format', en: 'Clear formatting', zh: '清除格式' },
  'rich.link_prompt': {
    id: 'Alamat tautan (http, https, atau mailto):',
    en: 'Link address (http, https or mailto):',
    zh: '链接地址（http、https 或 mailto）：',
  },
  'rich.link_invalid': {
    id: 'Hanya tautan http, https, atau mailto yang diperbolehkan.',
    en: 'Only http, https or mailto links are allowed.',
    zh: '仅允许 http、https 或 mailto 链接。',
  },
} satisfies Record<string, Entry>

export type MessageKey = keyof typeof messages
