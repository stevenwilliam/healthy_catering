package i18n

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in    string
		want  Locale
		known bool
	}{
		{"id", ID, true},
		{"en", EN, true},
		{"zh", ZH, true},
		{"EN", EN, true},
		{"id-ID", ID, true},
		{"en-GB", EN, true},
		{"zh-Hans", ZH, true},
		// A Traditional-Chinese browser still lands on Chinese rather than
		// falling through to Indonesian.
		{"zh-TW", ZH, true},
		{"", ID, false},
		{"fr", ID, false},
		{"nonsense", ID, false},
	}
	for _, c := range cases {
		got, known := Parse(c.in)
		if got != c.want || known != c.known {
			t.Errorf("Parse(%q) = %v,%v; want %v,%v", c.in, got, known, c.want, c.known)
		}
	}
}

func TestFromPath(t *testing.T) {
	cases := []struct {
		in       string
		want     Locale
		wantRest string
	}{
		{"/", ID, "/"},
		{"/menu/keto", ID, "/menu/keto"},
		{"/en/", EN, "/"},
		{"/en/menu/keto", EN, "/menu/keto"},
		{"/zh/menu/keto", ZH, "/menu/keto"},
		// "english" is not the prefix "en" — a diet slug that merely starts
		// with those letters must not be eaten as a locale.
		{"/english/menu", ID, "/english/menu"},
		{"/menu/en", ID, "/menu/en"},
	}
	for _, c := range cases {
		got, rest := FromPath(c.in)
		if got != c.want || rest != c.wantRest {
			t.Errorf("FromPath(%q) = %v,%q; want %v,%q", c.in, got, rest, c.want, c.wantRest)
		}
	}
}

func TestPathRoundTrips(t *testing.T) {
	for _, l := range Supported {
		for _, rest := range []string{"/", "/menu/keto"} {
			p := Path(l, rest)
			gotL, gotRest := FromPath(p)
			if gotL != l || gotRest != rest {
				t.Errorf("Path(%v,%q)=%q then FromPath=%v,%q", l, rest, p, gotL, gotRest)
			}
		}
	}
}

func TestFromAcceptLanguage(t *testing.T) {
	cases := []struct {
		in    string
		want  Locale
		known bool
	}{
		{"", ID, false},
		{"fr-FR,fr;q=0.9", ID, false},
		{"en-US,en;q=0.9", EN, true},
		{"zh-CN,zh;q=0.9,en;q=0.8", ZH, true},
		// q-values decide, not order.
		{"en;q=0.3,zh;q=0.9", ZH, true},
		// q=0 is an explicit refusal, so the next best wins.
		{"en;q=0,zh;q=0.5", ZH, true},
		{"id", ID, true},
	}
	for _, c := range cases {
		got, known := FromAcceptLanguage(c.in)
		if got != c.want || known != c.known {
			t.Errorf("FromAcceptLanguage(%q) = %v,%v; want %v,%v", c.in, got, known, c.want, c.known)
		}
	}
}

func TestCatalogFallsBackRatherThanBlanking(t *testing.T) {
	c := Catalog{
		"full":    {ID: "halo", EN: "hello", ZH: "你好"},
		"partial": {ID: "hanya id"},
	}
	if got := c.T(ZH, "full"); got != "你好" {
		t.Errorf("T(ZH, full) = %q", got)
	}
	// A missing translation shows the default language, never an empty label.
	if got := c.T(ZH, "partial"); got != "hanya id" {
		t.Errorf("T(ZH, partial) = %q; want the Indonesian fallback", got)
	}
	// An unknown key is loud, not silent.
	if got := c.T(EN, "nope"); got != "nope" {
		t.Errorf("T(EN, nope) = %q; want the key echoed back", got)
	}
}

func TestCatalogMissing(t *testing.T) {
	c := Catalog{
		"full":    {ID: "a", EN: "b", ZH: "c"},
		"partial": {ID: "a", EN: "  "},
	}
	missing := c.Missing()
	if _, ok := missing["full"]; ok {
		t.Error("a fully translated key was reported missing")
	}
	got := missing["partial"]
	if len(got) != 2 {
		t.Fatalf("partial: want EN and ZH missing, got %v", got)
	}
}
