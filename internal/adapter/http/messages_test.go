package http

import (
	"strings"
	"testing"

	"github.com/stevenwilliam/healthy_catering/internal/platform/i18n"
)

// The catalogue is the contract: a key that reaches production without a
// Chinese string renders Indonesian to a Chinese reader, which looks like a
// bug and is invisible in review. Fail here instead.
func TestPublicMessagesComplete(t *testing.T) {
	missing := publicMessages.Missing()
	if len(missing) == 0 {
		return
	}
	for key, locales := range missing {
		names := make([]string, 0, len(locales))
		for _, l := range locales {
			names = append(names, string(l))
		}
		t.Errorf("publicMessages[%q] has no translation for: %s", key, strings.Join(names, ", "))
	}
}

// Every locale needs a flag, or the selector renders a label with a hole
// beside it.
func TestEveryLocaleHasAFlag(t *testing.T) {
	for _, l := range i18n.Supported {
		if strings.TrimSpace(string(flagSVG[l])) == "" {
			t.Errorf("no flag for locale %q", l)
		}
	}
}

// Format keys carry verbs, and a mismatch between languages means one locale
// renders "%!s(MISSING)" in a page title.
func TestFormatKeysAgreeOnVerbs(t *testing.T) {
	for _, key := range []string{"menu.title"} {
		want := strings.Count(publicMessages.T(i18n.Default, key), "%s")
		for _, l := range i18n.Supported {
			if got := strings.Count(publicMessages.T(l, key), "%s"); got != want {
				t.Errorf("%s[%s] has %d %%s verbs; the default has %d", key, l, got, want)
			}
		}
	}
}
