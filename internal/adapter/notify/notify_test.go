package notify

import (
	"strings"
	"testing"
)

// A customer's name reaches an email body; markup in it must not reach the
// recipient's mail client as markup (CLAUDE.md §4 — encode on the way out).
func TestRenderEscapesCustomerInput(t *testing.T) {
	tpl := NewTemplates()
	m, err := tpl.Render(TplVerifyEmail, "en", map[string]any{
		"Name": `<script>alert(1)</script>`,
		"URL":  "https://www.evermore.co.id/verify?token=abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.HTML, "<script>") {
		t.Errorf("unescaped script tag reached the HTML part:\n%s", m.HTML)
	}
	if strings.Contains(m.Body, "<script>") {
		t.Errorf("unescaped script tag reached the text part:\n%s", m.Body)
	}
}

// Indonesian is the default; an unknown locale must not produce an empty
// message.
func TestRenderFallsBackToIndonesian(t *testing.T) {
	tpl := NewTemplates()
	for _, locale := range []string{"", "fr", "id-ID"} {
		m, err := tpl.Render(TplOrderPlaced, locale, map[string]any{
			"Name": "Budi", "OrderCode": "ABCD1234", "PaymentAmount": "Rp 500.123",
			"BankName": "BCA", "BankAccount": "123", "BankHolder": "PT",
			"Deadline": "18:00",
		})
		if err != nil {
			t.Fatalf("locale %q: %v", locale, err)
		}
		if m.Subject == "" || m.Body == "" {
			t.Errorf("locale %q produced an empty message", locale)
		}
		if !strings.Contains(m.Body, "Rp 500.123") {
			t.Errorf("locale %q lost the payment amount", locale)
		}
	}
}

// The unique suffix only works if customers do not round it off, so the
// message has to say so.
func TestOrderPlacedExplainsTheUniqueCode(t *testing.T) {
	tpl := NewTemplates()
	for _, locale := range []string{"id-ID", "en"} {
		m, _ := tpl.Render(TplOrderPlaced, locale, map[string]any{
			"Name": "Budi", "OrderCode": "X", "PaymentAmount": "Rp 500.123",
			"BankName": "BCA", "BankAccount": "1", "BankHolder": "PT", "Deadline": "18:00",
		})
		low := strings.ToLower(m.Body)
		if !strings.Contains(low, "bulatkan") && !strings.Contains(low, "round") {
			t.Errorf("locale %s does not warn against rounding the amount", locale)
		}
	}
}

// D-31: unused credits are forfeited, so the expiry warning must say so before
// the day it happens.
func TestExpiryWarningStatesForfeiture(t *testing.T) {
	tpl := NewTemplates()
	m, _ := tpl.Render(TplPackageExpiring, "en", map[string]any{
		"Name": "Budi", "PackageName": "Paket 20", "ExpiresAt": "2026-09-30", "Remaining": 4,
	})
	if !strings.Contains(strings.ToLower(m.Body), "cannot be refunded") {
		t.Errorf("the expiry warning must state that unused credits are lost:\n%s", m.Body)
	}
}

// A recipient or subject carrying CRLF would inject extra mail headers.
func TestHeaderInjectionRefused(t *testing.T) {
	s := NewSMTPSender(SMTPConfig{Host: "127.0.0.1", Port: 1025, FromEmail: "a@b.c"})
	err := s.Send(t.Context(), Message{
		Recipient: "victim@example.com\r\nBcc: attacker@evil.test",
		Subject:   "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "line break") {
		t.Errorf("expected the CRLF recipient to be refused, got %v", err)
	}
}

func TestUnknownTemplateIsAnError(t *testing.T) {
	if _, err := NewTemplates().Render("no_such_template", "en", nil); err == nil {
		t.Error("an unknown template must error rather than send an empty message")
	}
}
