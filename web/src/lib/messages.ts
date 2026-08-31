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
  'ui.all': { id: 'Semua', en: 'All', zh: '全部' },
  'ui.decrease': { id: 'Kurangi', en: 'Decrease', zh: '减少' },
  // A row LABEL, so it is capitalised — `menu.savings` is the same word used
  // mid-sentence and stays lowercase.
  'c03.saving_label': { id: 'Hemat', en: 'Saving', zh: '优惠' },
  'ui.increase': { id: 'Tambah', en: 'Increase', zh: '增加' },
  'ui.error': { id: 'Ada yang salah', en: 'Something is wrong', zh: '存在问题' },
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
  'addresses.note_label': { id: 'Catatan', en: 'Note', zh: '备注' },
  'addresses.latitude': { id: 'Lintang', en: 'Latitude', zh: '纬度' },
  'addresses.longitude': { id: 'Bujur', en: 'Longitude', zh: '经度' },
  'addresses.search_placeholder': {
    id: 'Cari label, penerima, kecamatan…', en: 'Search label, recipient, district…',
    zh: '搜索标签、收件人、区…',
  },
  'addresses.no_matches': {
    id: 'Tidak ada alamat yang cocok dengan pencarian.',
    en: 'No address matches that search.', zh: '没有符合搜索条件的地址。',
  },
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
  // ── CSV export ───────────────────────────────────────────────────────────
  'csv.export': { id: 'Ekspor CSV', en: 'Export CSV', zh: '导出 CSV' },
  'csv.failed': {
    id: 'Ekspor gagal. Coba lagi.', en: 'Export failed. Try again.', zh: '导出失败，请重试。',
  },

  // ══ Back office — the canvas's S1–S5 and the two print artifacts ═════════
  // docs/10 §4.10 / §4.11. The sidebar, the five screens and the two sheets
  // that come off the kitchen printer every morning.
  'bo.title': { id: 'Back office', en: 'Back office', zh: '后台' },
  'bo.dashboard': { id: 'Dasbor', en: 'Dashboard', zh: '仪表板' },
  'bo.calendar': { id: 'Jadwal menu', en: 'Menu schedule', zh: '菜单排期' },
  'bo.pricing': { id: 'Harga', en: 'Pricing', zh: '价格' },
  'bo.coverage': { id: 'Dapur & wilayah', en: 'Kitchens & areas', zh: '厨房与配送区' },
  'bo.production': { id: 'Lembar produksi', en: 'Production sheet', zh: '生产单' },
  'bo.labels': { id: 'Label kemasan', en: 'Packing labels', zh: '包装标签' },
  'bo.signed_in_as': { id: 'Masuk sebagai', en: 'Signed in as', zh: '当前登录' },

  // ── S1 · daily dashboard ─────────────────────────────────────────────────
  'dash.title': { id: 'Dasbor harian', en: 'Daily dashboard', zh: '每日仪表板' },
  'dash.subtitle': {
    id: 'Angka untuk tanggal layanan hari ini, zona Asia/Jakarta',
    en: "Figures for today's service date, Asia/Jakarta",
    zh: '今日服务日期数据，亚洲/雅加达时区',
  },
  'dash.cutoff': { id: 'Cut-off besok', en: 'Tomorrow cut-off', zh: '明日截单' },
  'dash.meals_today': { id: 'Meal hari ini', en: 'Meals today', zh: '今日餐数' },
  'dash.deliveries': { id: 'Pengantaran', en: 'Deliveries', zh: '配送' },
  'dash.needs_verify': { id: 'Perlu verifikasi', en: 'Awaiting verification', zh: '待核验' },
  'dash.revenue': { id: 'Omzet terverifikasi', en: 'Verified revenue', zh: '已核验营收' },
  'dash.out_of_range': { id: 'Di luar jangkauan', en: 'Out of range', zh: '超出配送范围' },
  'dash.out_of_range_sub': { id: 'percobaan checkout', en: 'checkout attempts', zh: '次结账尝试' },
  'dash.capacity': { id: 'Kapasitas per dapur & slot', en: 'Capacity by kitchen & slot', zh: '各厨房与时段产能' },
  'dash.capacity_legend': { id: 'terpakai / kuota', en: 'used / quota', zh: '已用 / 配额' },
  'dash.kitchen': { id: 'Dapur', en: 'Kitchen', zh: '厨房' },
  'dash.closed': { id: 'tutup', en: 'closed', zh: '休息' },
  'dash.actions': { id: 'Perlu tindakan', en: 'Needs action', zh: '待处理' },
  'dash.action_proofs': {
    id: 'bukti transfer menunggu', en: 'transfer proofs waiting', zh: '笔转账凭证待核验',
  },
  'dash.action_oldest': { id: 'tertua', en: 'oldest', zh: '最久' },
  'dash.verify': { id: 'Verifikasi', en: 'Verify', zh: '核验' },
  'dash.load_failed': {
    id: 'Dasbor tidak bisa dimuat.', en: 'The dashboard could not load.', zh: '仪表板加载失败。',
  },

  // ── S2 · menu schedule calendar ──────────────────────────────────────────
  'cal.title': { id: 'Jadwal menu', en: 'Menu schedule', zh: '菜单排期' },
  'cal.prev_week': { id: '‹ Minggu lalu', en: '‹ Last week', zh: '‹ 上周' },
  'cal.next_week': { id: 'Minggu depan ›', en: 'Next week ›', zh: '下周 ›' },
  'cal.publish_week': { id: 'Terbitkan minggu ini', en: 'Publish this week', zh: '发布本周' },
  'cal.published': { id: 'Published', en: 'Published', zh: '已发布' },
  'cal.draft': { id: 'Draft', en: 'Draft', zh: '草稿' },
  'cal.at_capacity': { id: 'Kapasitas penuh', en: 'At capacity', zh: '产能已满' },
  'cal.global_note': {
    id: 'Kalender global lintas dapur', en: 'Global calendar across kitchens', zh: '跨厨房全局日历',
  },
  'cal.schedule': { id: '+ Jadwalkan', en: '+ Schedule', zh: '+ 排期' },
  'cal.publish': { id: 'Terbitkan', en: 'Publish', zh: '发布' },
  'cal.components': { id: 'komponen', en: 'components', zh: '个组成' },
  'cal.filled': { id: 'terisi', en: 'filled', zh: '已订' },
  'cal.full': { id: 'penuh', en: 'full', zh: '已满' },
  'cal.diet': { id: 'Diet', en: 'Diet', zh: '饮食类型' },
  'cal.load_failed': {
    id: 'Kalender tidak bisa dimuat.', en: 'The calendar could not load.', zh: '日历加载失败。',
  },
  'cal.publish_failed': {
    id: 'Penerbitan gagal.', en: 'Publishing failed.', zh: '发布失败。',
  },

  // ── S3 · the four price forms ────────────────────────────────────────────
  'price.title': { id: 'Harga', en: 'Pricing', zh: '价格' },
  'price.tax_note': {
    id: 'Semua harga sudah termasuk PPN. Pemisahan basis dan pajak dihitung saat pesanan, bukan di baris harga.',
    en: 'All prices include VAT. The base/tax split is computed on the order, not on the price row.',
    zh: '所有价格均含增值税。税基与税额在订单结算时拆分，而非记录在价格行上。',
  },
  'price.meal_normal': { id: 'Meal · normal', en: 'Meal · normal', zh: '单餐 · 标准' },
  'price.meal_promo': { id: 'Meal · promo', en: 'Meal · promo', zh: '单餐 · 促销' },
  'price.pkg_normal': { id: 'Paket · normal', en: 'Package · normal', zh: '套餐 · 标准' },
  'price.pkg_promo': { id: 'Paket · promo', en: 'Package · promo', zh: '套餐 · 促销' },
  'price.scope': { id: 'Scope', en: 'Scope', zh: '适用范围' },
  'price.valid': { id: 'Berlaku', en: 'Valid', zh: '生效' },
  'price.tier': { id: 'Tier', en: 'Tier', zh: '档位' },
  'price.range': { id: 'Rentang meal', en: 'Meal range', zh: '餐数区间' },
  'price.incl_tax': { id: 'Harga termasuk PPN', en: 'Price incl. VAT', zh: '含税价格' },
  'price.base_tax': { id: 'Basis + pajak', en: 'Base + tax', zh: '税基 + 税额' },
  'price.active': { id: 'Aktif', en: 'Active', zh: '启用' },
  'price.yes': { id: 'Ya', en: 'Yes', zh: '是' },
  'price.archived': { id: 'Arsip', en: 'Archived', zh: '已归档' },
  'price.promo': { id: 'Promo', en: 'Promo', zh: '促销' },
  'price.tier_ok': { id: 'Tangga tier utuh', en: 'Tier ladder is sound', zh: '档位阶梯完整' },
  'price.tier_gap': { id: 'Tangga tier bolong', en: 'Tier ladder has a hole', zh: '档位阶梯有缺口' },
  'price.resolver': { id: 'Uji resolusi harga', en: 'Price resolution test', zh: '价格解析测试' },
  'price.run_resolver': { id: 'Jalankan resolver', en: 'Run resolver', zh: '运行解析器' },
  'price.result': { id: 'Hasil', en: 'Result', zh: '结果' },
  'price.per_meal': { id: 'per meal', en: 'per meal', zh: '每餐' },
  'price.trace': { id: 'Jejak resolusi', en: 'Resolution trace', zh: '解析轨迹' },
  'price.trace_note': {
    id: 'Jejak ini ikut tersimpan di pesanan, jadi pertanyaan "kenapa pelanggan ini bayar segitu" terjawab dari catatan tanpa menjalankan ulang resolver.',
    en: 'The trace is stored on the order, so "why did this customer pay that" is answered from the record without re-running the resolver.',
    zh: '解析轨迹随订单一并保存，因此“这位客户为何付这个价”可直接查记录，无需重新运行解析器。',
  },
  'price.qty': { id: 'Jumlah meal', en: 'Meal count', zh: '餐数' },
  'price.order_date': { id: 'Tanggal pesan', en: 'Order date', zh: '下单日期' },
  'price.load_failed': {
    id: 'Daftar harga tidak bisa dimuat.', en: 'The price list could not load.', zh: '价格表加载失败。',
  },

  // ── S5 · kitchen coverage ────────────────────────────────────────────────
  'cov.title': { id: 'Dapur & wilayah layanan', en: 'Kitchens & service areas', zh: '厨房与服务区域' },
  'cov.rule': {
    id: 'Poligon menang atas radius. Titik di luar semuanya diblokir dan dicatat.',
    en: 'Polygon beats radius. A point outside every area is blocked and logged.',
    zh: '多边形优先于半径。所有区域之外的坐标一律拒绝并记录。',
  },
  'cov.radius': { id: 'Radius layanan', en: 'Service radius', zh: '服务半径' },
  'cov.polygon': { id: 'Poligon', en: 'Polygon', zh: '多边形' },
  'cov.points': { id: 'titik', en: 'points', zh: '个顶点' },
  'cov.slots_today': { id: 'Slot & kapasitas hari ini', en: "Today's slots & capacity", zh: '今日时段与产能' },
  'cov.slot': { id: 'Slot', en: 'Slot', zh: '时段' },
  'cov.quota': { id: 'Kuota', en: 'Quota', zh: '配额' },
  'cov.used': { id: 'Terpakai', en: 'Used', zh: '已用' },
  'cov.manual_note': {
    id: 'Penugasan manual tidak pernah ditimpa oleh re-route, dan re-route setelah cut-off ditolak.',
    en: 'A manual assignment is never overwritten by a re-route, and a re-route after cut-off is refused.',
    zh: '人工指派不会被自动改派覆盖，截单后的改派一律拒绝。',
  },
  'cov.priority': { id: 'prioritas', en: 'priority', zh: '优先级' },
  'cov.schematic': {
    id: 'skema dari koordinat asli — bukan peta jalan',
    en: 'schematic from the real coordinates — not a street map',
    zh: '基于真实坐标的示意图 — 非街道地图',
  },
  'cov.district': { id: 'Kecamatan', en: 'District', zh: '区' },
  'cov.city': { id: 'Kota', en: 'City', zh: '城市' },
  'cov.attempts': { id: 'Percobaan', en: 'Attempts', zh: '尝试次数' },
  'cov.notify': { id: 'Minta dikabari', en: 'Notify requests', zh: '开通提醒' },
  'cov.nearest': { id: 'Dapur terdekat', en: 'Nearest kitchen', zh: '最近厨房' },
  'cov.active': { id: 'aktif', en: 'active', zh: '启用' },
  'cov.inactive': { id: 'nonaktif', en: 'inactive', zh: '停用' },
  'cov.load_failed': {
    id: 'Wilayah tidak bisa dimuat.', en: 'Coverage could not load.', zh: '配送区加载失败。',
  },

  // ── P1 · kitchen production sheet ────────────────────────────────────────
  'prod.title': { id: 'Lembar produksi', en: 'Production sheet', zh: '生产单' },
  'prod.printed': { id: 'Dicetak', en: 'Printed', zh: '打印于' },
  'prod.snapshot': {
    id: 'Snapshot setelah cut-off', en: 'Snapshot taken after cut-off', zh: '截单后快照',
  },
  'prod.total_portions': { id: 'Total porsi', en: 'Total portions', zh: '总份数' },
  'prod.slots': { id: 'Slot', en: 'Slots', zh: '时段' },
  'prod.allergen_notes': { id: 'Catatan alergen', en: 'Allergen notes', zh: '过敏原备注' },
  'prod.per_meal_slot': { id: 'Porsi per meal & slot', en: 'Portions by meal & slot', zh: '各餐品与时段份数' },
  'prod.meal': { id: 'Meal', en: 'Meal', zh: '餐品' },
  'prod.components': { id: 'Kebutuhan komponen', en: 'Component requirements', zh: '组成用量' },
  'prod.components_short': { id: 'Komponen', en: 'Components', zh: '组成种类' },
  'prod.role': { id: 'Peran', en: 'Role', zh: '角色' },
  'prod.allergen_note': {
    id: 'Rincian alergen dan permintaan khusus ada per pengantaran, tercetak di label kemasan.',
    en: 'Allergen detail and special requests are per delivery and print on the packing labels.',
    zh: '过敏原与特殊要求按每次配送记录，并打印在包装标签上。',
  },
  'prod.component': { id: 'Komponen', en: 'Component', zh: '组成' },
  'prod.per_portion': { id: 'Per porsi', en: 'Per portion', zh: '每份' },
  'prod.portions': { id: 'Porsi', en: 'Portions', zh: '份数' },
  'prod.total': { id: 'Total', en: 'Total', zh: '合计' },
  'prod.special': {
    id: 'Catatan alergen & permintaan khusus',
    en: 'Allergen notes & special requests',
    zh: '过敏原与特殊要求',
  },
  'prod.checked': { id: 'Diperiksa', en: 'Checked by', zh: '检查人' },
  'prod.head_chef': { id: 'Kepala dapur', en: 'Head chef', zh: '厨师长' },
  'prod.print': { id: 'Cetak', en: 'Print', zh: '打印' },
  'prod.load_failed': {
    id: 'Lembar produksi tidak bisa dimuat.', en: 'The production sheet could not load.', zh: '生产单加载失败。',
  },

  // ── P2 · packing label ───────────────────────────────────────────────────
  'label.title': { id: 'Label kemasan', en: 'Packing labels', zh: '包装标签' },
  'label.for': { id: 'Untuk', en: 'For', zh: '收件人' },
  'label.deliver': { id: 'Antar', en: 'Deliver', zh: '配送' },
  'label.kitchen': { id: 'Dapur', en: 'Kitchen', zh: '厨房' },
  'label.contents': { id: 'Isi', en: 'Contents', zh: '内容' },
  'label.allergens': { id: 'Alergen', en: 'Allergens', zh: '过敏原' },
  'label.order': { id: 'Pesanan', en: 'Order', zh: '订单' },
  'label.keep_cold': {
    id: 'Simpan dingin · habiskan dalam 24 jam',
    en: 'Keep refrigerated · consume within 24 hours',
    zh: '冷藏保存 · 24 小时内食用',
  },
  'label.track': { id: 'QR LACAK', en: 'QR TRACK', zh: 'QR 追踪' },
  'label.compact': { id: 'Label ringkas', en: 'Compact label', zh: '简版标签' },
  'label.load_failed': {
    id: 'Label tidak bisa dimuat.', en: 'Labels could not load.', zh: '标签加载失败。',
  },

  // ── Shared across the new screens ────────────────────────────────────────
  'ui.print': { id: 'Cetak', en: 'Print', zh: '打印' },
  'ui.back': { id: 'Kembali', en: 'Back', zh: '返回' },
  'ui.none': { id: 'tidak ada', en: 'none', zh: '无' },
  'ui.today': { id: 'Hari ini', en: 'Today', zh: '今天' },

  // ══ The canvas as the specification (2026-08-31) ═════════════════════════
  // Steven: the artboards' own Indonesian is the id-ID catalogue. English and
  // Simplified Chinese are written to match it, and all three move together
  // (CLAUDE.md §10). Where a string is a RULE the customer is held to — the
  // cut-off, credit expiry, the exact-amount suffix — the translation says the
  // same thing, not a softer version of it.

  // ── Back-office rail, the ten items S1 draws ─────────────────────────────
  'bo.foods': { id: 'Makanan & gizi', en: 'Dishes & nutrition', zh: '菜品与营养' },
  'bo.orders': { id: 'Pesanan', en: 'Orders', zh: '订单' },
  'bo.customers': { id: 'Pelanggan', en: 'Customers', zh: '客户' },
  'bo.packages': { id: 'Paket & kredit', en: 'Packages & credit', zh: '套餐与余额' },

  // ── 01 · menu calendar ───────────────────────────────────────────────────
  'c01.title': { id: 'Menu minggu ini', en: "This week's menu", zh: '本周菜单' },
  'c01.published_until': {
    id: 'Terbit sampai {0}', en: 'Published through {0}', zh: '已发布至 {0}',
  },
  'c01.cutoff': {
    id: 'Batas pesan untuk besok {0} lagi — pukul 15.00 WIB',
    en: 'Ordering for tomorrow closes in {0} — 15.00 WIB',
    zh: '明日订单将于 {0} 后截止 —— 15:00（WIB）',
  },
  'c01.add': { id: 'Tambah', en: 'Add', zh: '加入' },
  'c01.incl_tax': { id: 'termasuk PPN', en: 'VAT included', zh: '含增值税' },
  'c01.cart': { id: 'Keranjang', en: 'Cart', zh: '购物车' },
  'c01.meals_total': { id: '{0} meal · {1}', en: '{0} meals · {1}', zh: '{0} 份 · {1}' },
  'c01.summary': {
    id: 'Komponen dan gizi', en: 'Components and nutrition', zh: '组成与营养',
  },

  // ── 02 · meal detail ─────────────────────────────────────────────────────
  'c02.contents': { id: 'Isi meal', en: 'What is in it', zh: '餐品内容' },
  'c02.role_main': { id: 'Utama', en: 'Main', zh: '主食' },
  'c02.role_side': { id: 'Pendamping', en: 'Side', zh: '配菜' },
  'c02.role_dessert': { id: 'Penutup', en: 'Dessert', zh: '甜点' },
  'c02.role_drink': { id: 'Minuman', en: 'Drink', zh: '饮品' },
  'c02.nutrition': { id: 'Informasi gizi', en: 'Nutrition', zh: '营养信息' },
  'c02.per_portion': { id: 'per porsi', en: 'per portion', zh: '每份' },
  'c02.energy': { id: 'Energi', en: 'Energy', zh: '能量' },
  'c02.protein': { id: 'Protein', en: 'Protein', zh: '蛋白质' },
  'c02.carbs': { id: 'Karbohidrat', en: 'Carbohydrate', zh: '碳水化合物' },
  'c02.fat': { id: 'Lemak', en: 'Fat', zh: '脂肪' },
  'c02.fibre': { id: 'Serat', en: 'Fibre', zh: '膳食纤维' },
  'c02.sodium': { id: 'Natrium', en: 'Sodium', zh: '钠' },
  'c02.sum_note': {
    id: 'Jumlah dari {0} komponen di atas.',
    en: 'The sum of the {0} components above.',
    zh: '以上 {0} 项组成的合计。',
  },
  'c02.allergens': { id: 'Alergen', en: 'Allergens', zh: '过敏原' },
  'c02.no_allergens': { id: 'Tidak ada alergen tercatat', en: 'No allergens recorded', zh: '未记录过敏原' },
  'c02.add_for': { id: 'Tambah · {0}', en: 'Add · {0}', zh: '加入 · {0}' },
  'c02.estimated_note': {
    id: 'Gizi belum lengkap untuk semua komponen, jadi angka ini perkiraan.',
    en: 'Nutrition is incomplete for some components, so these figures are an estimate.',
    zh: '部分组成的营养数据尚不完整，因此以上数值为估算值。',
  },

  // ── 03 · cart ────────────────────────────────────────────────────────────
  'c03.title': { id: 'Keranjang', en: 'Cart', zh: '购物车' },
  'c03.empty': {
    id: 'Keranjang masih kosong.', en: 'Your cart is empty.', zh: '购物车是空的。',
  },
  'c03.per_portion_price': { id: 'Harga per porsi', en: 'Price per portion', zh: '每份价格' },
  'c03.tier_band': { id: '{0} porsi', en: '{0} portions', zh: '{0} 份' },
  'c03.tier_active': { id: 'berlaku', en: 'applies', zh: '适用' },
  'c03.upsell': {
    id: 'Tambah {0} porsi lagi, semua turun ke {1}',
    en: 'Add {0} more and every portion drops to {1}',
    zh: '再加 {0} 份，每份将降至 {1}',
  },
  'c03.topup': { id: 'Isi ulang', en: 'Top up', zh: '去凑单' },
  'c03.subtotal': { id: 'Subtotal · {0} porsi', en: 'Subtotal · {0} portions', zh: '小计 · {0} 份' },
  'c03.shipping': {
    id: 'Ongkos kirim · {0} pengantaran',
    en: 'Delivery · {0} drops', zh: '配送费 · {0} 次',
  },
  'c03.vat_incl': {
    id: 'PPN {0}% (sudah termasuk)', en: 'VAT {0}% (included)', zh: '增值税 {0}%（已含）',
  },
  'c03.total': { id: 'Total', en: 'Total', zh: '合计' },
  'c03.checkout': { id: 'Checkout', en: 'Checkout', zh: '去结算' },
  'c03.remove': { id: 'Hapus', en: 'Remove', zh: '移除' },

  // ── 04 · delivery ────────────────────────────────────────────────────────
  'c04.title': { id: 'Pengiriman', en: 'Delivery', zh: '配送' },
  'c04.move_pin': { id: 'Geser pin', en: 'Move pin', zh: '移动定位' },
  'c04.map_label': { id: 'peta pengantaran', en: 'delivery map', zh: '配送地图' },
  'c04.primary': { id: 'Utama', en: 'Primary', zh: '默认' },
  'c04.change': { id: 'Ubah', en: 'Change', zh: '更改' },
  'c04.slot_for': { id: 'Jam antar · {0}', en: 'Delivery time · {0}', zh: '配送时段 · {0}' },
  'c04.slot_full': {
    id: '{0} sudah penuh untuk area ini.',
    en: '{0} is full for this area.',
    zh: '{0} 在该区域已约满。',
  },
  'c04.kitchen_confirm': { id: 'Konfirmasi dapur', en: 'Kitchen confirmation', zh: '厨房确认' },
  'c04.kitchen_note': {
    id: 'Pesanan masuk antrean dapur setelah pembayaran diverifikasi. Kalau satu meal batal, kami hubungi lewat WhatsApp dan kreditkan penuh.',
    en: 'Your order enters the kitchen queue once payment is verified. If a meal falls through we contact you on WhatsApp and credit it back in full.',
    zh: '付款核验通过后，订单即进入厨房队列。若某份餐无法供应，我们会通过 WhatsApp 联系您并全额返还余额。',
  },
  'c04.courier_note': { id: 'Catatan untuk kurir', en: 'Note for the courier', zh: '给骑手的备注' },
  'c04.courier_placeholder': {
    id: 'Tulis patokan atau pesan singkat',
    en: 'A landmark, or a short message',
    zh: '填写地标或简短说明',
  },
  'c04.pay': { id: 'Bayar', en: 'Pay', zh: '应付' },
  'c04.continue': { id: 'Lanjut bayar', en: 'Continue to payment', zh: '去付款' },
  'c04.no_address': {
    id: 'Tambah alamat dulu sebelum checkout.',
    en: 'Add an address before checking out.',
    zh: '请先添加地址再结算。',
  },

  // ── 05 · manual transfer ─────────────────────────────────────────────────
  'c05.title': { id: 'Pembayaran', en: 'Payment', zh: '付款' },
  'c05.order': { id: 'Pesanan {0}', en: 'Order {0}', zh: '订单 {0}' },
  'c05.transfer': { id: 'Transfer {0}', en: 'Transfer {0}', zh: '转账 {0}' },
  'c05.deadline': {
    id: 'Bayar dalam {0}. Lewat batas itu pesanan dibatalkan otomatis dan slot dilepas.',
    en: 'Pay within {0}. After that the order is cancelled automatically and the slot is released.',
    zh: '请在 {0} 内付款。逾期订单将自动取消，所占时段一并释放。',
  },
  'c05.account': { id: 'Rekening tujuan', en: 'Transfer to', zh: '收款账户' },
  'c05.exact': {
    id: 'Nominal tepat sampai 3 digit terakhir',
    en: 'Exact amount, to the last three digits',
    zh: '请精确到最后三位数字',
  },
  'c05.copy': { id: 'Salin', en: 'Copy', zh: '复制' },
  'c05.proof': { id: 'Bukti transfer', en: 'Proof of transfer', zh: '转账凭证' },
  'c05.upload': {
    id: 'Unggah tangkapan layar mutasi',
    en: 'Upload a screenshot of the transfer',
    zh: '上传转账截图',
  },
  'c05.formats': { id: 'JPG atau PNG, maks 5 MB', en: 'JPG or PNG, max 5 MB', zh: 'JPG 或 PNG，最大 5 MB' },
  'c05.waiting': { id: 'Menunggu', en: 'Waiting', zh: '待核验' },
  'c05.waiting_note': {
    id: 'Tim keuangan memverifikasi manual pada jam kerja, rata-rata di bawah 30 menit. Status dikirim ke WhatsApp.',
    en: 'Finance verifies by hand during business hours, typically under 30 minutes. We send the result to WhatsApp.',
    zh: '财务在工作时间人工核验，通常不超过 30 分钟，结果将通过 WhatsApp 发送。',
  },
  'c05.done': { id: 'Saya sudah transfer', en: "I've transferred", zh: '我已完成转账' },
  'c05.suffix_note': {
    id: 'Tiga digit terakhir membedakan pembayaranmu dari yang lain. Kirim persis angka ini.',
    en: 'The last three digits tell your payment apart from everyone else\u2019s. Send exactly this amount.',
    zh: '最后三位数字用于区分您的付款，请务必按此金额转账。',
  },

  // ── 06 · package credit ──────────────────────────────────────────────────
  'c06.title': { id: 'Paket saya', en: 'My packages', zh: '我的套餐' },
  'c06.balance': { id: '{0} dari {1} porsi', en: '{0} of {1} portions', zh: '{1} 份中剩余 {0} 份' },
  // The artboard sets the number large and the rest small, so the suffix is
  // its own string rather than something sliced out of the sentence above.
  'c06.of_total': { id: 'dari {0} porsi', en: 'of {0} portions', zh: '共 {0} 份' },
  'credit.purchase': { id: 'Pembelian paket', en: 'Package purchased', zh: '购买套餐' },
  'credit.used': { id: 'Terpakai', en: 'Used', zh: '已使用' },
  'credit.refunded': { id: 'Dikembalikan', en: 'Refunded', zh: '已返还' },
  'credit.expired': { id: 'Hangus', en: 'Expired', zh: '已过期' },
  'packages.book_failed': {
    id: 'Slot tidak bisa dikunci. Coba slot lain.',
    en: 'The slot could not be locked. Try another.',
    zh: '该时段无法锁定，请选择其他时段。',
  },
  'c06.valid_until': {
    id: 'Berlaku sampai {0} · {1} hari lagi',
    en: 'Valid until {0} · {1} days left',
    zh: '有效期至 {0} · 剩余 {1} 天',
  },
  'c06.use_credit': {
    id: 'Pakai kredit untuk minggu depan',
    en: 'Use credit for next week',
    zh: '用余额预订下周',
  },
  'c06.history': { id: 'Riwayat kredit', en: 'Credit history', zh: '余额记录' },
  'c06.expiry_note': {
    id: 'Kredit hangus setelah masa berlaku dan tidak bisa diuangkan.',
    en: 'Credit expires at the end of its term and cannot be refunded as cash.',
    zh: '余额到期作废，且不可折现。',
  },
  'c06.need_more': { id: 'Butuh lebih banyak?', en: 'Need more?', zh: '需要更多？' },
  'c06.buy': { id: 'Beli paket', en: 'Buy a package', zh: '购买套餐' },
  'c06.none': {
    id: 'Belum ada paket aktif.', en: 'No active package yet.', zh: '暂无有效套餐。',
  },

  // ── M2 · buy a package ───────────────────────────────────────────────────
  'm2.title': { id: 'Paket', en: 'Packages', zh: '套餐' },
  'm2.headline': {
    id: 'Bayar sekali, pakai 90 hari',
    en: 'Pay once, use it for 90 days',
    zh: '一次付费，90 天内使用',
  },
  'm2.sub': {
    id: 'Kredit dipakai pada menu apa pun yang sudah terbit.',
    en: 'Credit works on any menu that has been published.',
    zh: '余额可用于任何已发布的菜单。',
  },
  'm2.portions': { id: '{0} porsi', en: '{0} portions', zh: '{0} 份' },
  'm2.per_portion': { id: '{0} per porsi', en: '{0} per portion', zh: '每份 {0}' },
  'm2.valid_days': { id: 'berlaku {0} hari', en: 'valid {0} days', zh: '有效期 {0} 天' },
  'm2.bestseller': { id: 'Paling laris', en: 'Most popular', zh: '最受欢迎' },
  'm2.saving': {
    id: 'Hemat {0} dibanding harga satuan',
    en: 'Saves {0} against the single-portion price',
    zh: '较单份价格节省 {0}',
  },
  'm2.terms': {
    id: 'Kredit tidak bisa diuangkan dan hangus setelah masa berlaku.',
    en: 'Credit cannot be refunded as cash and expires at the end of its term.',
    zh: '余额不可折现，到期即作废。',
  },
  'm2.buy': { id: 'Beli paket', en: 'Buy package', zh: '购买套餐' },

  // ── M3 · pick a slot from credit ─────────────────────────────────────────
  'm3.title': { id: 'Pilih slot', en: 'Pick a slot', zh: '选择时段' },
  'm3.credits': { id: '{0} kredit', en: '{0} credits', zh: '{0} 份余额' },
  'm3.pick_time': { id: '{0} · pilih jam', en: '{0} · pick a time', zh: '{0} · 选择时间' },
  'm3.area_full': {
    id: 'Kapasitas penuh untuk wilayahmu',
    en: 'At capacity for your area',
    zh: '您所在区域已约满',
  },
  'm3.change_note': {
    id: 'Slot bisa diubah sampai 15.00 sehari sebelumnya. Setelah itu kredit terkunci pada jadwal ini.',
    en: 'You can change the slot until 15.00 the day before. After that the credit is locked to this schedule.',
    zh: '可在前一日 15:00 前更改时段，此后余额将锁定于该安排。',
  },
  'm3.selected': { id: '{0} slot dipilih', en: '{0} slot selected', zh: '已选 {0} 个时段' },
  'm3.remaining': { id: 'sisa {0} kredit', en: '{0} credits left', zh: '剩余 {0} 份' },
  'm3.lock': { id: 'Kunci jadwal', en: 'Lock the schedule', zh: '锁定安排' },
  'm3.none_published': {
    id: 'Belum ada menu terbit untuk minggu ini.',
    en: 'No menu published for this week yet.',
    zh: '本周尚未发布菜单。',
  },

  // ── S1–S5, the canvas's own back-office wording ─────────────────────────
  'dash.revenue_sub': { id: 'termasuk PPN', en: 'VAT included', zh: '含增值税' },
  'dash.deliveries_sub': { id: '{0} terkirim · {1} di jalan', en: '{0} delivered · {1} en route', zh: '{0} 已送达 · {1} 配送中' },
  'dash.meals_sub': { id: '{0} dapur · {1} slot', en: '{0} kitchens · {1} slots', zh: '{0} 间厨房 · {1} 个时段' },
  'dash.action_draft': {
    id: 'Menu {0} masih DRAFT', en: 'Menu for {0} is still DRAFT', zh: '{0} 的菜单仍为草稿',
  },
  'dash.action_price': {
    id: 'Harga {0} belum diisi', en: 'No price set for {0}', zh: '{0} 尚未设置价格',
  },
  'dash.action_price_sub': {
    id: 'blokir checkout — PRICE_NOT_CONFIGURED',
    en: 'blocks checkout — PRICE_NOT_CONFIGURED',
    zh: '将阻止结算 —— PRICE_NOT_CONFIGURED',
  },
  'dash.review': { id: 'Tinjau', en: 'Review', zh: '查看' },
  'dash.fill': { id: 'Isi', en: 'Fill in', zh: '补充' },
  'dash.arrange': { id: 'Atur', en: 'Arrange', zh: '调整' },

  'pay.tab_waiting': { id: 'Menunggu · {0}', en: 'Waiting · {0}', zh: '待核验 · {0}' },
  'pay.tab_rejected': { id: 'Ditolak · {0}', en: 'Rejected · {0}', zh: '已驳回 · {0}' },
  'pay.tab_verified': { id: 'Terverifikasi hari ini · {0}', en: 'Verified today · {0}', zh: '今日已核验 · {0}' },
  'pay.audit_note': {
    id: 'Setiap keputusan tercatat di audit log dengan aktor dan alasan.',
    en: 'Every decision is written to the audit log with its actor and reason.',
    zh: '每次决定都会连同操作人与原因写入审计日志。',
  },
  'pay.col_order': { id: 'Pesanan', en: 'Order', zh: '订单' },
  'pay.col_customer': { id: 'Pelanggan', en: 'Customer', zh: '客户' },
  'pay.col_amount': { id: 'Nominal', en: 'Amount', zh: '金额' },
  'pay.col_waiting': { id: 'Menunggu', en: 'Waiting', zh: '已等待' },
  'pay.col_match': { id: 'Kecocokan', en: 'Match', zh: '匹配' },
  'pay.match_ok': { id: 'Cocok', en: 'Match', zh: '一致' },
  'pay.match_none': { id: 'Belum ada bukti', en: 'No proof yet', zh: '暂无凭证' },
  'pay.proof_label': { id: 'bukti transfer · unggahan pelanggan', en: 'transfer proof · customer upload', zh: '转账凭证 · 客户上传' },
  'pay.billed': { id: 'Tagihan', en: 'Billed', zh: '应收' },
  'pay.suffix_match': {
    id: 'Tiga digit terakhir {0} cocok dengan sufiks pesanan. Sufiks tidak masuk basis pajak.',
    en: 'The last three digits {0} match the order suffix. The suffix is not part of the tax base.',
    zh: '末三位 {0} 与订单后缀一致。该后缀不计入税基。',
  },
  'pay.retail': { id: 'Retail', en: 'Retail', zh: '零售' },
  'pay.select_row': { id: 'Pilih pesanan untuk ditinjau', en: 'Select an order to review', zh: '请选择要审核的订单' },

  'cov.save_area': { id: 'Simpan wilayah', en: 'Save area', zh: '保存配送区' },
  'cov.deactivate': { id: 'Nonaktifkan', en: 'Deactivate', zh: '停用' },
  'cov.legend': { id: 'Legenda', en: 'Legend', zh: '图例' },
  'cov.not_editable': {
    id: 'Wilayah masih diubah lewat data dapur — belum ada endpoint tulis.',
    en: 'Coverage is still edited through the kitchen record — there is no write endpoint yet.',
    zh: '配送区目前仍需通过厨房数据修改 —— 尚无写入接口。',
  },
  'cov.legend_radius': { id: 'Radius layanan', en: 'Service radius', zh: '服务半径' },
  'cov.legend_polygon': { id: 'Poligon aktif', en: 'Active polygon', zh: '启用的多边形' },
  'cov.legend_rejected': { id: 'Checkout ditolak', en: 'Checkout refused', zh: '结算被拒' },

  'price.save_row': { id: 'Simpan baris', en: 'Save row', zh: '保存该行' },
  'price.validation': { id: 'Validasi tier', en: 'Tier validation', zh: '档位校验' },
  'price.no_end': { id: 'tanpa akhir', en: 'no end date', zh: '无结束日期' },

  'cal.publish_all': { id: 'Terbitkan minggu ini', en: 'Publish this week', zh: '发布本周' },
  'cal.d_minus': { id: 'D-{0}', en: 'D-{0}', zh: 'D-{0}' },

  // ── M1, the home page's own navigation ──────────────────────────────────
  'm1.nav_menu': { id: 'Menu minggu ini', en: "This week's menu", zh: '本周菜单' },
  'm1.nav_how': { id: 'Cara kerja', en: 'How it works', zh: '服务流程' },
  'm1.nav_packages': { id: 'Paket', en: 'Packages', zh: '套餐' },
  'm1.nav_corporate': { id: 'Untuk kantor', en: 'For offices', zh: '企业订餐' },
  'm1.nav_areas': { id: 'Wilayah antar', en: 'Delivery areas', zh: '配送范围' },
  'm1.sign_in': { id: 'Masuk', en: 'Sign in', zh: '登录' },
  'm1.see_menu': { id: 'Lihat menu', en: 'See the menu', zh: '查看菜单' },

  // ── Meal photo upload ───────────────────────────────────────────────────
  'photo.title': { id: 'Foto meal', en: 'Meal photo', zh: '餐品照片' },
  'photo.upload': { id: 'Unggah foto', en: 'Upload photo', zh: '上传照片' },
  'photo.replace': { id: 'Ganti foto', en: 'Replace photo', zh: '更换照片' },
  'photo.remove': { id: 'Hapus foto', en: 'Remove photo', zh: '删除照片' },
  'photo.none': {
    id: 'Belum ada foto — kartu memakai warna tipe diet.',
    en: 'No photo yet — the card uses the diet-type tint.',
    zh: '尚无照片 —— 卡片将使用饮食类型底色。',
  },
  'photo.formats': { id: 'JPG atau PNG, maks 5 MB', en: 'JPG or PNG, max 5 MB', zh: 'JPG 或 PNG，最大 5 MB' },
  'photo.failed': { id: 'Unggahan foto gagal.', en: 'The photo upload failed.', zh: '照片上传失败。' },
} satisfies Record<string, Entry>

export type MessageKey = keyof typeof messages
