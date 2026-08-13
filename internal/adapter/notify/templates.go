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
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nTerima kasih sudah mendaftar di Evermore.\n" +
				"Klik tautan berikut untuk mengonfirmasi email Anda:\n\n{{.URL}}\n\n" +
				"Tautan ini berlaku 24 jam.",
			"en": "Hello {{.Name}},\n\nThanks for signing up to Evermore.\n" +
				"Click the link below to confirm your email address:\n\n{{.URL}}\n\n" +
				"This link is valid for 24 hours.",
		},
	},
	TplOrderPlaced: {
		subject: map[string]string{
			"id-ID": "Pesanan {{.OrderCode}} — menunggu pembayaran",
			"en":    "Order {{.OrderCode}} — awaiting payment",
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
		},
	},
	TplPaymentVerified: {
		subject: map[string]string{
			"id-ID": "Pembayaran diterima — {{.OrderCode}}",
			"en":    "Payment received — {{.OrderCode}}",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nPembayaran untuk {{.OrderCode}} sudah kami terima. " +
				"Pesanan Anda masuk ke dapur.\n\n{{.Extra}}",
			"en": "Hello {{.Name}},\n\nWe have received your payment for {{.OrderCode}}. " +
				"Your order is with the kitchen.\n\n{{.Extra}}",
		},
	},
	TplPaymentRejected: {
		subject: map[string]string{
			"id-ID": "Bukti transfer perlu diperiksa — {{.OrderCode}}",
			"en":    "We need another look at your transfer — {{.OrderCode}}",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nBukti transfer untuk {{.OrderCode}} belum bisa kami " +
				"verifikasi.\n\nAlasan: {{.Reason}}\n\nSilakan unggah ulang sebelum {{.Deadline}}.",
			"en": "Hello {{.Name}},\n\nWe could not verify the transfer proof for {{.OrderCode}}.\n\n" +
				"Reason: {{.Reason}}\n\nPlease upload it again before {{.Deadline}}.",
		},
	},
	TplDeliveryReminder: {
		subject: map[string]string{
			"id-ID": "Pengiriman besok — {{.ServiceDate}}",
			"en":    "Your delivery tomorrow — {{.ServiceDate}}",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nBesok ({{.ServiceDate}}) kami antar {{.Slot}} ke:\n" +
				"{{.Address}}\n\nSampai jumpa besok.",
			"en": "Hello {{.Name}},\n\nTomorrow ({{.ServiceDate}}) we deliver {{.Slot}} to:\n" +
				"{{.Address}}\n\nSee you then.",
		},
	},
	TplCreditsLow: {
		subject: map[string]string{
			"id-ID": "Kredit Anda tinggal {{.Remaining}}",
			"en":    "{{.Remaining}} credits left",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nPaket {{.PackageName}} Anda tinggal {{.Remaining}} kredit, " +
				"berlaku sampai {{.ExpiresAt}}.",
			"en": "Hello {{.Name}},\n\nYour {{.PackageName}} has {{.Remaining}} credits left, " +
				"valid until {{.ExpiresAt}}.",
		},
	},
	TplPackageExpiring: {
		subject: map[string]string{
			"id-ID": "Paket Anda berakhir {{.ExpiresAt}}",
			"en":    "Your package expires on {{.ExpiresAt}}",
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
		},
	},
	TplPackageExpired: {
		subject: map[string]string{
			"id-ID": "Paket {{.PackageName}} sudah berakhir",
			"en":    "Your {{.PackageName}} has expired",
		},
		text: map[string]string{
			"id-ID": "Halo {{.Name}},\n\nPaket {{.PackageName}} berakhir pada {{.ExpiresAt}}.\n\n" +
				"Terima kasih sudah makan bersama kami — silakan hubungi kami bila ada pertanyaan.",
			"en": "Hello {{.Name}},\n\nYour {{.PackageName}} expired on {{.ExpiresAt}}.\n\n" +
				"Thank you for eating with us — do get in touch if you have any questions.",
		},
	},
	TplOutOfRangeNotify: {
		subject: map[string]string{
			"id-ID": "Kami sudah mengantar ke daerah Anda",
			"en":    "We now deliver to your area",
		},
		text: map[string]string{
			"id-ID": "Halo,\n\nKabar baik — Evermore kini melayani {{.District}}.\n\n" +
				"Silakan pesan di {{.URL}}",
			"en": "Hello,\n\nGood news — Evermore now delivers to {{.District}}.\n\n" +
				"Order at {{.URL}}",
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
	if locale != "en" {
		locale = "id-ID"
	}

	subject, err := t.exec(name+".subject", c.subject[locale], data)
	if err != nil {
		return Message{}, err
	}
	body, err := t.exec(name+".text", c.text[locale], data)
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
