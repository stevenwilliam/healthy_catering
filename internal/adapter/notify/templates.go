package notify

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"sync"
)

// Templates are the nine transactional messages PROMPT §11 lists, in both
// languages (id-ID default).
//
// Bodies are rendered with html/template for the HTML part, which escapes by
// default — a customer whose name is <script>alert(1)</script> must not turn a
// confirmation email into an attack on whoever opens it (CLAUDE.md §4: encode
// on the way OUT, for the context).
type Templates struct {
	mu     sync.RWMutex
	cached map[string]*template.Template
}

// Template names.
const (
	TplVerifyEmail      = "verify_email"
	TplOrderPlaced      = "order_placed"
	TplPaymentVerified  = "payment_verified"
	TplPaymentRejected  = "payment_rejected"
	TplDeliveryReminder = "delivery_reminder"
	TplCreditsLow       = "credits_low"
	TplPackageExpiring  = "package_expiring"
	TplPackageExpired   = "package_expired"
	TplOutOfRangeNotify = "coverage_reached"
)

// Copy is one rendered message in one language.
type copyPair struct {
	subject map[string]string
	text    map[string]string
}

var copyBook = map[string]copyPair{
	TplVerifyEmail: {
		subject: map[string]string{
			"id-ID": "Konfirmasi email Anda — Evermore",
			"en":    "Confirm your email — Evermore",
			"zh":    "请确认您的邮箱 — Evermore",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nTerima kasih sudah mendaftar di Evermore.\n" +
				"Klik tautan berikut untuk mengonfirmasi email Anda:\n\n{{.URL}}\n\n" +
				"Tautan ini berlaku 24 jam.",
			"en": "Hello {{.Name}},\n\nThanks for signing up to Evermore.\n" +
				"Click the link below to confirm your email address:\n\n{{.URL}}\n\n" +
				"This link is valid for 24 hours.",
			"zh": "您好 {{.Name}}，\n\n感谢您注册 Evermore。\n" +
				"请点击以下链接确认您的邮箱地址：\n\n{{.URL}}\n\n该链接 24 小时内有效。",
		},
	},
	TplOrderPlaced: {
		subject: map[string]string{
			"id-ID": "Pesanan {{.OrderCode}} — menunggu pembayaran",
			"en":    "Order {{.OrderCode}} — awaiting payment",
			"zh":    "订单 {{.OrderCode}} — 待付款",
		},
		text: map[string]string{
			// The unique suffix is explained rather than just shown: a customer
			// who rounds it off breaks the matching it exists for.
			"id-ID": "Halo {{.Name}},\n\nPesanan {{.OrderCode}} sudah kami terima.\n\n" +
				"Silakan transfer TEPAT sebesar:\n\n  {{.PaymentAmount}}\n\n" +
				"Tiga digit terakhir adalah kode unik Anda — mohon jangan dibulatkan, " +
				"karena angka itulah yang kami pakai untuk mencocokkan pembayaran.\n\n" +
				"Rekening: {{.BankName}} {{.BankAccount}} a.n. {{.BankHolder}}\n" +
				"Batas waktu: {{.Deadline}}\n\n" +
				"Setelah transfer, unggah bukti di halaman pesanan Anda.",
			"en": "Hello {{.Name}},\n\nWe have your order {{.OrderCode}}.\n\n" +
				"Please transfer EXACTLY:\n\n  {{.PaymentAmount}}\n\n" +
				"The last three digits are your unique code — please do not round it, " +
				"as that is how we match your payment.\n\n" +
				"Account: {{.BankName}} {{.BankAccount}} ({{.BankHolder}})\n" +
				"Deadline: {{.Deadline}}\n\n" +
				"After transferring, upload your proof on the order page.",
			"zh": "您好 {{.Name}}，\n\n我们已收到您的订单 {{.OrderCode}}。\n\n" +
				"请转账准确金额：\n\n  {{.PaymentAmount}}\n\n" +
				"末尾三位是您的专属识别码，请勿四舍五入——我们依靠它核对您的付款。\n\n" +
				"账户：{{.BankName}} {{.BankAccount}}（户名 {{.BankHolder}}）\n" +
				"截止时间：{{.Deadline}}\n\n转账后，请在订单页面上传付款凭证。",
		},
	},
	TplPaymentVerified: {
		subject: map[string]string{
			"id-ID": "Pembayaran diterima — {{.OrderCode}}",
			"en":    "Payment received — {{.OrderCode}}",
			"zh":    "已收到付款 — {{.OrderCode}}",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nPembayaran untuk {{.OrderCode}} sudah kami terima. " +
				"Pesanan Anda masuk ke dapur.\n\n{{.Extra}}",
			"en": "Hello {{.Name}},\n\nWe have received your payment for {{.OrderCode}}. " +
				"Your order is with the kitchen.\n\n{{.Extra}}",
			"zh": "您好 {{.Name}}，\n\n我们已收到 {{.OrderCode}} 的付款，" +
				"您的订单已进入厨房。\n\n{{.Extra}}",
		},
	},
	TplPaymentRejected: {
		subject: map[string]string{
			"id-ID": "Bukti transfer perlu diperiksa — {{.OrderCode}}",
			"en":    "We need another look at your transfer — {{.OrderCode}}",
			"zh":    "您的转账凭证需要重新核对 — {{.OrderCode}}",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nBukti transfer untuk {{.OrderCode}} belum bisa kami " +
				"verifikasi.\n\nAlasan: {{.Reason}}\n\nSilakan unggah ulang sebelum {{.Deadline}}.",
			"en": "Hello {{.Name}},\n\nWe could not verify the transfer proof for {{.OrderCode}}.\n\n" +
				"Reason: {{.Reason}}\n\nPlease upload it again before {{.Deadline}}.",
			"zh": "您好 {{.Name}}，\n\n我们无法核实 {{.OrderCode}} 的转账凭证。\n\n" +
				"原因：{{.Reason}}\n\n请在 {{.Deadline}} 前重新上传。",
		},
	},
	TplDeliveryReminder: {
		subject: map[string]string{
			"id-ID": "Pengiriman besok — {{.ServiceDate}}",
			"en":    "Your delivery tomorrow — {{.ServiceDate}}",
			"zh":    "您明天的配送 — {{.ServiceDate}}",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nBesok ({{.ServiceDate}}) kami antar {{.Slot}} ke:\n" +
				"{{.Address}}\n\nSampai jumpa besok.",
			"en": "Hello {{.Name}},\n\nTomorrow ({{.ServiceDate}}) we deliver {{.Slot}} to:\n" +
				"{{.Address}}\n\nSee you then.",
			"zh": "您好 {{.Name}}，\n\n明天（{{.ServiceDate}}）我们将把 {{.Slot}} 送至：\n" +
				"{{.Address}}\n\n明天见。",
		},
	},
	TplCreditsLow: {
		subject: map[string]string{
			"id-ID": "Kredit Anda tinggal {{.Remaining}}",
			"en":    "{{.Remaining}} credits left",
			"zh":    "您还剩 {{.Remaining}} 份餐额",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nPaket {{.PackageName}} Anda tinggal {{.Remaining}} kredit, " +
				"berlaku sampai {{.ExpiresAt}}.",
			"en": "Hello {{.Name}},\n\nYour {{.PackageName}} has {{.Remaining}} credits left, " +
				"valid until {{.ExpiresAt}}.",
			"zh": "您好 {{.Name}}，\n\n您的 {{.PackageName}} 还剩 {{.Remaining}} 份餐额，" +
				"有效期至 {{.ExpiresAt}}。",
		},
	},
	TplPackageExpiring: {
		subject: map[string]string{
			"id-ID": "Paket Anda berakhir {{.ExpiresAt}}",
			"en":    "Your package expires on {{.ExpiresAt}}",
			"zh":    "您的套餐将于 {{.ExpiresAt}} 到期",
		},
		text: map[string]string{
			// Forfeiture is stated plainly and early, because D-31 means unused
			// credits are lost and nobody should learn that on the day.
			"id-ID": "Halo {{.Name}},\n\nPaket {{.PackageName}} berakhir pada {{.ExpiresAt}} " +
				"dan Anda masih punya {{.Remaining}} kredit.\n\n" +
				"Kredit yang tidak terpakai tidak dapat diuangkan atau diperpanjang otomatis, " +
				"jadi silakan pesan sebelum tanggal tersebut.",
			"en": "Hello {{.Name}},\n\nYour {{.PackageName}} expires on {{.ExpiresAt}} " +
				"and you still have {{.Remaining}} credits.\n\n" +
				"Unused credits cannot be refunded or automatically extended, " +
				"so please book before then.",
			"zh": "您好 {{.Name}}，\n\n您的 {{.PackageName}} 将于 {{.ExpiresAt}} 到期，" +
				"目前还剩 {{.Remaining}} 份餐额。\n\n" +
				"未使用的餐额不可退款，也不会自动延期，请在到期前预订。",
		},
	},
	TplPackageExpired: {
		subject: map[string]string{
			"id-ID": "Paket {{.PackageName}} sudah berakhir",
			"en":    "Your {{.PackageName}} has expired",
			"zh":    "您的 {{.PackageName}} 已到期",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nPaket {{.PackageName}} berakhir pada {{.ExpiresAt}}.\n\n" +
				"Terima kasih sudah makan bersama kami — silakan hubungi kami bila ada pertanyaan.",
			"en": "Hello {{.Name}},\n\nYour {{.PackageName}} expired on {{.ExpiresAt}}.\n\n" +
				"Thank you for eating with us — do get in touch if you have any questions.",
			"zh": "您好 {{.Name}}，\n\n您的 {{.PackageName}} 已于 {{.ExpiresAt}} 到期。\n\n" +
				"感谢您选择我们的餐食——如有任何疑问，欢迎随时联系。",
		},
	},
	TplOutOfRangeNotify: {
		subject: map[string]string{
			"id-ID": "Kami sudah mengantar ke daerah Anda",
			"en":    "We now deliver to your area",
			"zh":    "我们现已配送到您所在的区域",
		},
		text: map[string]string{
			"id-ID": "Halo,\n\nKabar baik — Evermore kini melayani {{.District}}.\n\n" +
				"Silakan pesan di {{.URL}}",
			"en": "Hello,\n\nGood news — Evermore now delivers to {{.District}}.\n\n" +
				"Order at {{.URL}}",
			"zh": "您好，\n\n好消息——Evermore 现已为 {{.District}} 提供配送服务。\n\n" +
				"立即订购：{{.URL}}",
		},
	},
}

// NewTemplates prepares the renderer.
func NewTemplates() *Templates {
	return &Templates{cached: map[string]*template.Template{}}
}

// Render produces a message for a template, locale and data set.
func (t *Templates) Render(name, locale string, data map[string]any) (Message, error) {
	c, ok := copyBook[name]
	if !ok {
		return Message{}, fmt.Errorf("notify: no template %q", name)
	}
	// Three locales now (Steven, 2026-08-18). Anything unrecognised falls back
	// to Indonesian, and a locale with a missing string falls back with it —
	// see localised() below.
	switch locale {
	case "en", "zh":
	default:
		locale = "id-ID"
	}

	subject, err := t.exec(name+".subject", localised(c.subject, locale), data)
	if err != nil {
		return Message{}, err
	}
	body, err := t.exec(name+".text", localised(c.text, locale), data)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Subject: subject, Body: body, HTML: htmlWrap(subject, body),
		Template: name, Locale: locale,
	}, nil
}

// exec renders one string. html/template escapes every interpolated value, so
// a customer's name cannot carry markup into the message.
//
// Each string is its OWN template, cached by name. Sharing one root template
// fails with "cannot Parse after Execute" the moment a second message is
// rendered — which would have shipped as "the first email works and the rest
// silently error".
func (t *Templates) exec(name, src string, data map[string]any) (string, error) {
	t.mu.RLock()
	tpl, ok := t.cached[name]
	t.mu.RUnlock()

	if !ok {
		parsed, err := template.New(name).Parse(src)
		if err != nil {
			return "", fmt.Errorf("notify: parse %s: %w", name, err)
		}
		t.mu.Lock()
		t.cached[name] = parsed
		t.mu.Unlock()
		tpl = parsed
	}
	var b bytes.Buffer
	if err := tpl.Execute(&b, data); err != nil {
		return "", fmt.Errorf("notify: render %s: %w", name, err)
	}
	// The plain-text part is escaped by html/template, which turns an
	// apostrophe into &#39;. Undo the entities that matter for plain text
	// while keeping the tags neutralised.
	out := b.String()
	out = strings.NewReplacer("&#39;", "'", "&#34;", `"`, "&amp;", "&").Replace(out)
	return out, nil
}

// htmlWrap gives the message a minimal HTML part in the brand colours.
//
// Nourish deep on Restore Beige measures 11.32:1 (docs/10 §2.4), which clears
// AA comfortably — email clients strip CSS unpredictably, so the colours are
// inline and the layout survives without them.
func htmlWrap(subject, body string) string {
	var b strings.Builder
	b.WriteString(`<div style="font-family:Georgia,serif;background:#FFFAE0;color:#1C3D34;padding:24px">`)
	b.WriteString(`<h1 style="font-size:20px;margin:0 0 16px">`)
	b.WriteString(template.HTMLEscapeString(subject))
	b.WriteString(`</h1><div style="font-family:Helvetica,Arial,sans-serif;font-size:15px;line-height:1.6;white-space:pre-wrap">`)
	b.WriteString(template.HTMLEscapeString(body))
	b.WriteString(`</div><p style="margin-top:24px;font-size:12px;color:#4A5D56">Evermore · evermore.co.id</p></div>`)
	return b.String()
}

// localised picks a string for a locale, falling back to Indonesian.
//
// Without the fallback a template missing one language would render an EMPTY
// subject and an empty body, and the customer would receive a blank email —
// strictly worse than receiving it in the wrong language.
func localised(m map[string]string, locale string) string {
	if s := m[locale]; s != "" {
		return s
	}
	return m["id-ID"]
}
