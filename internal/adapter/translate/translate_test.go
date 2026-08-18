package translate

import (
	"context"
	"testing"
)

// With nothing configured the service must still be usable: Available() says
// no, and Translate returns ErrUnavailable rather than panicking or blocking.
func TestNoopIsSafe(t *testing.T) {
	var tr Translator = Noop{}
	if tr.Available() {
		t.Error("Noop reported itself available")
	}
	if _, err := tr.Translate(context.Background(), "halo", "id", "en"); err != ErrUnavailable {
		t.Errorf("err = %v; want ErrUnavailable", err)
	}
}

// An empty or unknown provider must fall back to Noop, never to a half-built
// client that can only fail at request time.
func TestNewFallsBackToNoop(t *testing.T) {
	for _, c := range []struct{ provider, key string }{
		{"", ""},
		{"", "a-key"},
		{"google", ""},   // named provider, no key
		{"deepl", "key"}, // provider we do not implement
	} {
		if got := New(c.provider, c.key); got.Available() {
			t.Errorf("New(%q, %q) is available; want the no-op", c.provider, c.key)
		}
	}
	if got := New("google", "a-key"); !got.Available() {
		t.Error("New(google, a-key) should be available")
	}
	if got := New("GOOGLE", "a-key"); !got.Available() {
		t.Error("provider matching should be case-insensitive")
	}
}

// "zh" is ambiguous to the API as well as to a reader; it has to go out as
// Simplified, matching what docs/11 says the UI means by zh.
func TestChineseIsPinnedToSimplified(t *testing.T) {
	if got := googleTarget("zh"); got != "zh-CN" {
		t.Errorf("googleTarget(zh) = %q; want zh-CN", got)
	}
	for _, l := range []string{"id", "en"} {
		if got := googleTarget(l); got != l {
			t.Errorf("googleTarget(%q) = %q; want it unchanged", l, got)
		}
	}
}
