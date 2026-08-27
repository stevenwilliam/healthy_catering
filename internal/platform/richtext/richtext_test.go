package richtext

import "testing"

func TestStripsScriptAndHandlers(t *testing.T) {
	cases := map[string]string{
		`<p>hi</p><script>alert(1)</script>`:        `<p>hi</p>`,
		`<img src=x onerror="alert(1)">`:            ``,
		`<p onclick="evil()">text</p>`:              `<p>text</p>`,
		`<a href="javascript:alert(1)">x</a>`:       `x`,
		`<p style="color:red">x</p>`:                `<p>x</p>`,
		`<iframe src="https://evil.test"></iframe>`: ``,
	}
	for in, want := range cases {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestKeepsTheFormattingAnEditorNeeds(t *testing.T) {
	in := `<ul><li><strong>a</strong> and <em>b</em></li></ul>`
	if got := Clean(in); got != in {
		t.Errorf("Clean(%q) = %q; the allowlist dropped legitimate formatting", in, got)
	}
}

func TestExternalLinksAreDefanged(t *testing.T) {
	got := Clean(`<a href="https://example.test">x</a>`)
	for _, want := range []string{`nofollow`, `noopener`, `noreferrer`} {
		if !contains(got, want) {
			t.Errorf("Clean() = %q; missing %s", got, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
