package notify

import "testing"

// Every message must exist in every language. A blank subject or body is
// worse than the wrong language: the customer gets an empty email.
func TestCopyBookCoversEveryLocale(t *testing.T) {
	for _, locale := range []string{"id-ID", "en", "zh"} {
		for name, pair := range copyBook {
			if pair.subject[locale] == "" {
				t.Errorf("%s: no subject for %s", name, locale)
			}
			if pair.text[locale] == "" {
				t.Errorf("%s: no body for %s", name, locale)
			}
		}
	}
}

// An unknown locale must render Indonesian, not nothing.
func TestUnknownLocaleFallsBack(t *testing.T) {
	tpl := NewTemplates()
	msg, err := tpl.Render(TplVerifyEmail, "fr", map[string]any{
		"Name": "Ven", "URL": "https://example.test/x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Subject == "" || msg.Body == "" {
		t.Fatalf("empty message for an unknown locale: %+v", msg)
	}
	if msg.Locale != "id-ID" {
		t.Errorf("locale = %q, want the id-ID fallback", msg.Locale)
	}
}
