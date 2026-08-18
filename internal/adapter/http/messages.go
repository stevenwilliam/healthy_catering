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
	"home.hero_alt": {
		i18n.ID: "Tiga orang bermeditasi dengan tenang di taman",
		i18n.EN: "Three people meditating peacefully in a garden",
		i18n.ZH: "三个人在花园里静心冥想",
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
