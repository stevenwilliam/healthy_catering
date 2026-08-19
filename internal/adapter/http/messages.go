package http

import (
	"html/template"

	"github.com/stevenwilliam/healthy_catering/internal/platform/i18n"
)

// publicMessages is the catalogue for the server-rendered public pages.
//
// One place, three languages, keyed by a stable identifier — CLAUDE.md §10
// ("message catalogues from the first string, never inline"). A key missing a
// translation falls back to Indonesian rather than rendering blank, and
// TestPublicMessagesComplete fails the build if any key is short of a locale,
// so a half-translated string cannot ship quietly.
//
// What this catalogue does NOT cover: diet-type names and descriptions, meal
// names and food items. Those are rows in `diet_type`, `meal` and `food`, each
// with a single `name`/`description` column, so they render in whatever
// language they were entered in regardless of the selected locale. Translating
// them needs translated columns plus an admin surface to type them into —
// raised in docs/03-open-questions.md as Q-24.
var publicMessages = i18n.Catalog{
	// ── Chrome ──────────────────────────────────────────────────────────────
	"nav.main": {
		i18n.ID: "Menu utama",
		i18n.EN: "Main menu",
		i18n.ZH: "主菜单",
	},
	"nav.home_aria": {
		i18n.ID: "Evermore — beranda",
		i18n.EN: "Evermore — home",
		i18n.ZH: "Evermore — 首页",
	},
	"lang.label": {
		i18n.ID: "Bahasa",
		i18n.EN: "Language",
		i18n.ZH: "语言",
	},
	"lang.choose": {
		i18n.ID: "Pilih bahasa",
		i18n.EN: "Choose a language",
		i18n.ZH: "选择语言",
	},
	"wa.aria": {
		i18n.ID: "Hubungi kami di WhatsApp",
		i18n.EN: "Contact us on WhatsApp",
		i18n.ZH: "通过 WhatsApp 联系我们",
	},

	// ── Home ────────────────────────────────────────────────────────────────
	"home.title": {
		i18n.ID: "Evermore — katering sehat harian di Jakarta",
		i18n.EN: "Evermore — daily healthy catering in Jakarta",
		i18n.ZH: "Evermore — 雅加达每日健康餐外送",
	},
	"home.description": {
		i18n.ID: "Makanan sehat harian diantar ke rumah atau kantor Anda di Jakarta. " +
			"Pilih menu sesuai kebutuhan: Healthy, Weight Loss, High Protein dan lainnya.",
		i18n.EN: "Healthy daily meals delivered to your home or office in Jakarta. " +
			"Choose the menu that fits: Healthy, Weight Loss, High Protein and more.",
		i18n.ZH: "健康的每日餐点，配送到您在雅加达的家或办公室。" +
			"按需选择菜单：Healthy、Weight Loss、High Protein 等。",
	},
	"home.eyebrow": {
		i18n.ID: "Katering sehat harian · Jakarta",
		i18n.EN: "Daily healthy catering · Jakarta",
		i18n.ZH: "每日健康餐 · 雅加达",
	},
	"home.h1": {
		i18n.ID: "Makan sehat, setiap hari, tanpa repot.",
		i18n.EN: "Eat well, every day, without the hassle.",
		i18n.ZH: "健康饮食，天天如此，轻松无忧。",
	},
	"home.cta": {
		i18n.ID: "Lihat menu minggu ini",
		i18n.EN: "See this week's menu",
		i18n.ZH: "查看本周菜单",
	},
	"home.diets_h2": {
		i18n.ID: "Pilih sesuai kebutuhan Anda",
		i18n.EN: "Choose what fits your goal",
		i18n.ZH: "选择适合您的方案",
	},
	// Steven, 2026-08-18: use "evermore homepage" rather than a description of
	// the picture. Worth knowing what that costs — alt text is what a screen
	// reader announces IN PLACE of the image, so a reader who cannot see it now
	// learns the name of the page they are already on rather than what is
	// shown. Same string in all three languages because it is the brand name
	// plus one English word, not a sentence to translate.
	"home.hero_alt": {
		i18n.ID: "Evermore homepage",
		i18n.EN: "Evermore homepage",
		i18n.ZH: "Evermore homepage",
	},
	"home.check_h2": {
		i18n.ID: "Kami antar ke tempat Anda?",
		i18n.EN: "Do we deliver to you?",
		i18n.ZH: "我们能送到您那里吗？",
	},
	"home.check_body": {
		i18n.ID: "Masukkan titik lokasi Anda saat mendaftar — kami langsung memberi tahu " +
			"dapur mana yang melayani, sebelum Anda memesan.",
		i18n.EN: "Drop your location pin when you sign up — we tell you which kitchen " +
			"serves you before you order.",
		i18n.ZH: "注册时标记您的位置——下单前我们会立即告知由哪个厨房为您服务。",
	},

	// ── Menu ────────────────────────────────────────────────────────────────
	"menu.eyebrow": {
		i18n.ID: "Menu",
		i18n.EN: "Menu",
		i18n.ZH: "菜单",
	},
	// %s is the diet-type name, which comes from the database and is therefore
	// not itself translated — see the note at the top of this file.
	"menu.title": {
		i18n.ID: "%s — menu minggu ini | Evermore",
		i18n.EN: "%s — this week's menu | Evermore",
		i18n.ZH: "%s — 本周菜单 | Evermore",
	},
	"menu.h2": {
		i18n.ID: "Menu tujuh hari ke depan",
		i18n.EN: "The next seven days",
		i18n.ZH: "未来七天菜单",
	},
	"menu.empty": {
		i18n.ID: "Menu untuk minggu ini sedang disiapkan. Silakan cek kembali besok.",
		i18n.EN: "This week's menu is being prepared. Please check back tomorrow.",
		i18n.ZH: "本周菜单正在准备中，请明天再来查看。",
	},
	"menu.kcal": {
		i18n.ID: "kkal",
		i18n.EN: "kcal",
		i18n.ZH: "千卡",
	},
	"menu.protein": {
		i18n.ID: "g protein",
		i18n.EN: "g protein",
		i18n.ZH: "克蛋白质",
	},
	"menu.estimated": {
		i18n.ID: "perkiraan",
		i18n.EN: "estimated",
		i18n.ZH: "估算值",
	},

	// ── 404 ─────────────────────────────────────────────────────────────────
	"notfound.title": {
		i18n.ID: "Halaman tidak ditemukan — Evermore",
		i18n.EN: "Page not found — Evermore",
		i18n.ZH: "页面未找到 — Evermore",
	},
	"notfound.h1": {
		i18n.ID: "Halaman tidak ditemukan",
		i18n.EN: "Page not found",
		i18n.ZH: "页面未找到",
	},
	"notfound.body": {
		i18n.ID: "Tautan yang Anda buka tidak ada atau sudah dipindahkan.",
		i18n.EN: "The link you opened doesn't exist or has moved.",
		i18n.ZH: "您打开的链接不存在或已被移动。",
	},
	"notfound.cta": {
		i18n.ID: "Kembali ke beranda",
		i18n.EN: "Back to home",
		i18n.ZH: "返回首页",
	},
	// ── Header navigation ───────────────────────────────────────────────────
	"nav.pricelist": {
		i18n.ID: "Daftar harga", i18n.EN: "Price list", i18n.ZH: "价格表",
	},
	"nav.contact":  {i18n.ID: "Kontak", i18n.EN: "Contact", i18n.ZH: "联系我们"},
	"nav.about":    {i18n.ID: "Tentang kami", i18n.EN: "About us", i18n.ZH: "关于我们"},
	"nav.career":   {i18n.ID: "Karier", i18n.EN: "Career", i18n.ZH: "招聘"},
	"nav.category": {i18n.ID: "Kategori", i18n.EN: "Category", i18n.ZH: "菜单分类"},

	// ── Price list ──────────────────────────────────────────────────────────
	"price.title": {
		i18n.ID: "Daftar harga — Evermore",
		i18n.EN: "Price list — Evermore",
		i18n.ZH: "价格表 — Evermore",
	},
	"price.description": {
		i18n.ID: "Harga menu Evermore dan pilihan paket harian untuk makan sehat di Jakarta.",
		i18n.EN: "Evermore's menu pricing and daily packages for eating well in Jakarta.",
		i18n.ZH: "Evermore 的菜单价格与按天套餐，助您在雅加达吃得健康。",
	},
	"price.h1": {
		i18n.ID: "Harga", i18n.EN: "Prices", i18n.ZH: "价格",
	},
	"price.lede": {
		i18n.ID: "Harga per porsi. Semakin banyak porsi dalam satu pesanan, semakin murah harganya.",
		i18n.EN: "Prices are per portion. The more portions in one order, the lower the price.",
		i18n.ZH: "价格以每份计算。单次订购份数越多，单价越低。",
	},
	"price.meals_h2": {
		i18n.ID: "Harga per porsi", i18n.EN: "Price per portion", i18n.ZH: "每份价格",
	},
	"price.meals_note": {
		i18n.ID: "Harga sudah termasuk wadah dan pengantaran ke area layanan kami.",
		i18n.EN: "Prices include packaging and delivery inside our service area.",
		i18n.ZH: "价格已包含餐盒及配送区域内的配送费用。",
	},
	"price.col_category": {i18n.ID: "Kategori", i18n.EN: "Category", i18n.ZH: "类别"},
	"price.col_tier":     {i18n.ID: "Jumlah porsi", i18n.EN: "Portions", i18n.ZH: "份数"},
	"price.col_price":    {i18n.ID: "Harga per porsi", i18n.EN: "Per portion", i18n.ZH: "每份价格"},
	"price.tiers_h3": {
		i18n.ID: "Tingkatan jumlah", i18n.EN: "Quantity tiers", i18n.ZH: "数量分档",
	},
	"price.empty": {
		i18n.ID: "Harga sedang diperbarui. Silakan hubungi kami untuk penawaran terbaru.",
		i18n.EN: "Prices are being updated. Please contact us for a current quote.",
		i18n.ZH: "价格正在更新中，请联系我们获取最新报价。",
	},
	"price.lede_quote": {
		i18n.ID: "Harga menyesuaikan kategori, jumlah porsi, dan lama berlangganan. Hubungi kami untuk penawaran yang pas.",
		i18n.EN: "Prices depend on the category, how many portions, and how long you order for. Contact us for a quote that fits.",
		i18n.ZH: "价格取决于菜单类别、订购份数与订购周期。请联系我们获取合适的报价。",
	},
	// Fallbacks only — the live wording is public_content, edited in the back
	// office. These exist so the section still renders against a database that
	// has never been seeded.
	"benefit.title": {
		i18n.ID: "Kenapa Evermore", i18n.EN: "Why Evermore", i18n.ZH: "为什么选择 Evermore",
	},
	"benefit.body": {
		i18n.ID: "<ul><li>Menu disusun bersama ahli gizi.</li><li>Dimasak pagi, diantar hari itu juga.</li><li>Kalori dan protein tercantum di setiap porsi.</li></ul>",
		i18n.EN: "<ul><li>Menus planned with nutritionists.</li><li>Cooked in the morning, delivered the same day.</li><li>Calories and protein on every portion.</li></ul>",
		i18n.ZH: "<ul><li>菜单由营养师共同制定。</li><li>清晨烹制，当天送达。</li><li>每份标明热量与蛋白质。</li></ul>",
	},
	"price.quote_h2": {
		i18n.ID: "Minta penawaran", i18n.EN: "Ask for a quote", i18n.ZH: "获取报价",
	},
	"price.quote_body": {
		i18n.ID: "Ceritakan kebutuhan Anda — kategori menu, berapa porsi per hari, dan untuk berapa lama — dan kami kirimkan penawaran beserta pilihan paketnya.",
		i18n.EN: "Tell us what you need — which menu, how many portions a day, and for how long — and we will send a quote with the package options.",
		i18n.ZH: "请告诉我们您的需求——所需菜单、每日份数以及订购时长——我们将为您发送报价及套餐选项。",
	},
	"price.quote_cta": {
		i18n.ID: "Hubungi kami", i18n.EN: "Contact us", i18n.ZH: "联系我们",
	},
	"price.packages_h2": {
		i18n.ID: "Paket harian", i18n.EN: "Daily packages", i18n.ZH: "按天套餐",
	},
	// A package is sold as a number of DAYS rather than of credits (Steven,
	// 2026-08-18). Only the label changed: the column, the ledger and every
	// balance underneath are still meal credits, because that is what is
	// actually decremented when a meal is booked.
	"price.credits": {i18n.ID: "hari", i18n.EN: "days", i18n.ZH: "天"},
	"price.on_request": {
		i18n.ID: "Hubungi kami", i18n.EN: "On request", i18n.ZH: "请咨询",
	},

	// ── Contact ─────────────────────────────────────────────────────────────
	"contact.title": {
		i18n.ID: "Kontak — Evermore",
		i18n.EN: "Contact — Evermore",
		i18n.ZH: "联系我们 — Evermore",
	},
	"contact.description": {
		i18n.ID: "Hubungi Evermore untuk pertanyaan menu, pesanan kantor, atau kerja sama.",
		i18n.EN: "Get in touch with Evermore about the menu, office orders or partnerships.",
		i18n.ZH: "如需咨询菜单、企业订餐或合作事宜，请联系 Evermore。",
	},
	"contact.h1": {i18n.ID: "Hubungi kami", i18n.EN: "Contact us", i18n.ZH: "联系我们"},
	"contact.lede": {
		i18n.ID: "Ada pertanyaan tentang menu, pesanan kantor, atau area pengantaran? Kami senang membantu.",
		i18n.EN: "Questions about the menu, an office order, or where we deliver? We are glad to help.",
		i18n.ZH: "对菜单、企业订餐或配送范围有疑问？我们很乐意为您解答。",
	},
	"contact.reach_h2": {
		i18n.ID: "Cara menghubungi kami", i18n.EN: "How to reach us", i18n.ZH: "联系方式",
	},
	"contact.email": {i18n.ID: "Email", i18n.EN: "Email", i18n.ZH: "邮箱"},
	"contact.phone": {i18n.ID: "Telepon", i18n.EN: "Phone", i18n.ZH: "电话"},
	"contact.hours": {
		i18n.ID: "Kami membalas pada hari kerja, Senin sampai Sabtu, pukul 08.00–17.00 WIB.",
		i18n.EN: "We reply on working days, Monday to Saturday, 08:00–17:00 WIB.",
		i18n.ZH: "我们在工作日回复：周一至周六 08:00–17:00（西部印尼时间）。",
	},

	// ── About ───────────────────────────────────────────────────────────────
	"about.title": {
		i18n.ID: "Tentang kami — Evermore",
		i18n.EN: "About us — Evermore",
		i18n.ZH: "关于我们 — Evermore",
	},
	"about.description": {
		i18n.ID: "Evermore memasak makanan sehat harian di Jakarta dan mengantarnya dari dapur terdekat.",
		i18n.EN: "Evermore cooks healthy daily meals in Jakarta and delivers from the nearest kitchen.",
		i18n.ZH: "Evermore 在雅加达制作每日健康餐，并由最近的厨房配送。",
	},
	"about.h1": {i18n.ID: "Tentang Evermore", i18n.EN: "About Evermore", i18n.ZH: "关于 Evermore"},
	"about.lede": {
		i18n.ID: "Kami memasak makanan sehat harian di Jakarta dan mengantarnya dari dapur yang paling dekat dengan Anda.",
		i18n.EN: "We cook healthy daily meals in Jakarta and deliver them from the kitchen closest to you.",
		i18n.ZH: "我们在雅加达制作每日健康餐，并由离您最近的厨房配送。",
	},
	"about.body": {
		i18n.ID: "Evermore berawal dari satu pertanyaan sederhana: bagaimana caranya makan sehat setiap hari tanpa harus memasak sendiri. Kami menyusun menu bersama ahli gizi, memasaknya pagi hari di beberapa dapur di Jakarta, dan mengantarnya ke rumah atau kantor Anda pada hari yang sama. Setiap porsi mencantumkan kalori dan proteinnya, karena Anda berhak tahu apa yang Anda makan.",
		i18n.EN: "Evermore started from one plain question: how do you eat well every day without having to cook. We plan the menu with nutritionists, cook it each morning across several kitchens in Jakarta, and deliver it to your home or office the same day. Every portion lists its calories and protein, because you should know what you are eating.",
		i18n.ZH: "Evermore 源于一个朴素的问题：如何在不亲自下厨的情况下每天吃得健康。我们与营养师共同制定菜单，每天清晨在雅加达的多个厨房烹制，并于当天送达您的家中或办公室。每一份餐点都标明热量与蛋白质含量——您有权知道自己吃的是什么。",
	},

	// ── Career ──────────────────────────────────────────────────────────────
	"career.title": {
		i18n.ID: "Karier — Evermore",
		i18n.EN: "Career — Evermore",
		i18n.ZH: "招聘 — Evermore",
	},
	"career.description": {
		i18n.ID: "Peluang karier di dapur, pengantaran, dan tim kantor Evermore di Jakarta.",
		i18n.EN: "Careers in Evermore's kitchens, delivery and office teams in Jakarta.",
		i18n.ZH: "Evermore 在雅加达的厨房、配送与办公团队招聘机会。",
	},
	"career.h1": {i18n.ID: "Karier", i18n.EN: "Careers", i18n.ZH: "加入我们"},
	"career.lede": {
		i18n.ID: "Kami sedang tumbuh, dan kami mencari orang yang peduli pada makanan yang baik.",
		i18n.EN: "We are growing, and we are looking for people who care about good food.",
		i18n.ZH: "我们正在成长，并寻找真正在意好食物的伙伴。",
	},
	"career.body": {
		i18n.ID: "Kami membuka kesempatan untuk juru masak, staf dapur, kurir, dan tim layanan pelanggan. Kirimkan CV Anda beserta posisi yang diminati, dan ceritakan sedikit mengapa Anda ingin bergabung. Kami membalas setiap lamaran yang masuk.",
		i18n.EN: "We have openings for cooks, kitchen staff, couriers and customer-service team members. Send us your CV with the role you are interested in, and a few words on why you want to join. We reply to every application we receive.",
		i18n.ZH: "我们正在招聘厨师、厨房员工、配送员以及客服团队成员。请发送您的简历，注明意向职位，并简要说明您希望加入的原因。我们会回复每一份收到的申请。",
	},
	// Fallback only — the live wording is public_content, edited on the Content
	// screen, so the offer can change without a deploy.
	// Fallbacks; the live wording is public_content.
	"cert.heading": {
		i18n.ID: "Standar keamanan pangan kami",
		i18n.EN: "Our food-safety standards",
		i18n.ZH: "我们的食品安全标准",
	},
	"cert.haccp_full": {
		i18n.ID: "Hazard Analysis and Critical Control Points",
		i18n.EN: "Hazard Analysis and Critical Control Points",
		i18n.ZH: "危害分析与关键控制点",
	},

	"ribbon.text": {
		i18n.ID: "Gratis ongkir", i18n.EN: "Free delivery", i18n.ZH: "免费配送",
	},

	"nav.benefits": {i18n.ID: "Keunggulan", i18n.EN: "Benefits", i18n.ZH: "我们的优势"},

	// ── Benefits page ───────────────────────────────────────────────────────
	"benefits.title": {
		i18n.ID: "Keunggulan — Evermore",
		i18n.EN: "Benefits — Evermore",
		i18n.ZH: "我们的优势 — Evermore",
	},
	"benefits.description": {
		i18n.ID: "Kenapa memilih Evermore untuk katering sehat harian di Jakarta.",
		i18n.EN: "Why choose Evermore for daily healthy catering in Jakarta.",
		i18n.ZH: "为什么选择 Evermore 的雅加达每日健康餐。",
	},

	// ── Career form ─────────────────────────────────────────────────────────
	"career.openings_h2": {
		i18n.ID: "Posisi yang sedang dibuka", i18n.EN: "Open positions", i18n.ZH: "正在招聘的职位",
	},
	"career.no_openings": {
		i18n.ID: "Belum ada posisi yang dibuka saat ini. Anda tetap boleh mengirim data — kami simpan untuk lowongan berikutnya.",
		i18n.EN: "No positions are open at the moment. You are still welcome to write — we keep it on file for the next vacancy.",
		i18n.ZH: "目前暂无空缺职位。您仍然可以提交信息，我们会保留至下次招聘。",
	},
	"career.form_h2": {
		i18n.ID: "Kirim lamaran", i18n.EN: "Apply", i18n.ZH: "提交申请",
	},
	"career.f_name":     {i18n.ID: "Nama lengkap", i18n.EN: "Full name", i18n.ZH: "姓名"},
	"career.f_email":    {i18n.ID: "Email", i18n.EN: "Email", i18n.ZH: "邮箱"},
	"career.f_phone":    {i18n.ID: "Nomor HP (opsional)", i18n.EN: "Mobile number (optional)", i18n.ZH: "手机号码（选填）"},
	"career.f_position": {i18n.ID: "Posisi yang dilamar", i18n.EN: "Position", i18n.ZH: "申请职位"},
	"career.f_position_choose": {
		i18n.ID: "Pilih posisi", i18n.EN: "Choose a position", i18n.ZH: "请选择职位",
	},
	"career.f_message": {
		i18n.ID: "Ceritakan tentang Anda", i18n.EN: "Tell us about yourself", i18n.ZH: "请介绍一下您自己",
	},
	"career.submit": {i18n.ID: "Kirim", i18n.EN: "Send", i18n.ZH: "提交"},
	"career.no_file_note": {
		i18n.ID: "Formulir ini tidak menerima lampiran. Bila kami tertarik, kami akan membalas email Anda dan meminta CV di sana.",
		i18n.EN: "This form does not accept attachments. If we are interested we will reply to your email and ask for your CV there.",
		i18n.ZH: "本表单不接受附件。如果我们感兴趣，会回复您的邮件并在邮件中索取简历。",
	},
	"career.thanks": {
		i18n.ID: "Terima kasih — lamaran Anda sudah kami terima. Kami membalas setiap lamaran yang masuk.",
		i18n.EN: "Thank you — we have your application. We reply to every one we receive.",
		i18n.ZH: "感谢您——我们已收到您的申请。我们会回复每一份申请。",
	},
	"career.e_required": {i18n.ID: "Wajib diisi.", i18n.EN: "Required.", i18n.ZH: "此项必填。"},
	"career.e_email": {
		i18n.ID: "Alamat email tidak valid.", i18n.EN: "That email address is not valid.", i18n.ZH: "邮箱地址无效。",
	},
	"career.e_phone": {
		i18n.ID: "Nomor HP tidak valid.", i18n.EN: "That mobile number is not valid.", i18n.ZH: "手机号码无效。",
	},
	"career.e_position": {
		i18n.ID: "Pilih salah satu posisi yang dibuka.", i18n.EN: "Choose one of the open positions.", i18n.ZH: "请选择一个正在招聘的职位。",
	},
	"career.err_toomany": {
		i18n.ID: "Terlalu banyak pengiriman dari koneksi ini. Coba lagi nanti.",
		i18n.EN: "Too many submissions from this connection. Please try again later.",
		i18n.ZH: "此连接提交次数过多，请稍后再试。",
	},
	"career.err_toolarge": {
		i18n.ID: "Isian terlalu panjang.", i18n.EN: "That submission is too long.", i18n.ZH: "提交内容过长。",
	},
	"career.err_unsupported": {
		i18n.ID: "Formulir ini hanya menerima teks, tanpa lampiran.",
		i18n.EN: "This form accepts text only, with no attachments.",
		i18n.ZH: "本表单仅接受文本，不接受附件。",
	},
	"career.err_failed": {
		i18n.ID: "Lamaran gagal terkirim. Silakan coba lagi.",
		i18n.EN: "The application could not be sent. Please try again.",
		i18n.ZH: "申请提交失败，请重试。",
	},

	"career.apply": {
		i18n.ID: "Kirim lamaran", i18n.EN: "Send an application", i18n.ZH: "投递简历",
	},
	"career.subject": {
		i18n.ID: "Lamaran kerja", i18n.EN: "Job application", i18n.ZH: "求职申请",
	},
}

// flagSVG is the flag shown beside each language in the selector.
//
// Inline SVG rather than the flag emoji (🇮🇩 🇬🇧 🇨🇳): Windows ships no glyphs
// for regional-indicator pairs, so on the browser Steven actually uses, emoji
// flags render as the letter boxes "ID", "GB", "CN" — the one place the
// control must not be ambiguous. These are decorative and marked aria-hidden,
// because the language NAME beside them is what carries the meaning: a flag is
// a country, not a language, and English is not the flag of one country.
var flagSVG = map[i18n.Locale]template.HTML{
	// Indonesia — red over white.
	i18n.ID: template.HTML(`<svg class="flag" viewBox="0 0 6 4" aria-hidden="true" focusable="false">` +
		`<rect width="6" height="2" fill="#CE1126"/>` +
		`<rect y="2" width="6" height="2" fill="#F5F5F5"/></svg>`),

	// The Union Flag, for English.
	i18n.EN: template.HTML(`<svg class="flag" viewBox="0 0 60 40" aria-hidden="true" focusable="false">` +
		`<rect width="60" height="40" fill="#012169"/>` +
		`<path d="M0,0 60,40 M60,0 0,40" stroke="#F5F5F5" stroke-width="8"/>` +
		`<path d="M0,0 60,40 M60,0 0,40" stroke="#C8102E" stroke-width="4"/>` +
		`<path d="M30,0 V40 M0,20 H60" stroke="#F5F5F5" stroke-width="13"/>` +
		`<path d="M30,0 V40 M0,20 H60" stroke="#C8102E" stroke-width="8"/></svg>`),

	// The People's Republic of China — one large star and four small ones,
	// each of the small ones rotated to point at the large one.
	i18n.ZH: template.HTML(`<svg class="flag" viewBox="0 0 30 20" aria-hidden="true" focusable="false">` +
		`<rect width="30" height="20" fill="#EE1C25"/>` +
		`<g fill="#FFDE00">` +
		`<path transform="translate(5,5) scale(3)" d="M0,-1 .225,-.309 .951,-.309 .363,.118 .588,.809 0,.382 -.588,.809 -.363,.118 -.951,-.309 -.225,-.309Z"/>` +
		`<path transform="translate(10,2) scale(1) rotate(23)" d="M0,-1 .225,-.309 .951,-.309 .363,.118 .588,.809 0,.382 -.588,.809 -.363,.118 -.951,-.309 -.225,-.309Z"/>` +
		`<path transform="translate(12,4) scale(1) rotate(46)" d="M0,-1 .225,-.309 .951,-.309 .363,.118 .588,.809 0,.382 -.588,.809 -.363,.118 -.951,-.309 -.225,-.309Z"/>` +
		`<path transform="translate(12,7) scale(1) rotate(70)" d="M0,-1 .225,-.309 .951,-.309 .363,.118 .588,.809 0,.382 -.588,.809 -.363,.118 -.951,-.309 -.225,-.309Z"/>` +
		`<path transform="translate(10,9) scale(1) rotate(21)" d="M0,-1 .225,-.309 .951,-.309 .363,.118 .588,.809 0,.382 -.588,.809 -.363,.118 -.951,-.309 -.225,-.309Z"/>` +
		`</g></svg>`),
}

func flagFor(l i18n.Locale) template.HTML { return flagSVG[l] }
