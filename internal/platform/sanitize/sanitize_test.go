package sanitize

import (
	"strings"
	"testing"
)

func TestTextNormalizesThenBounds(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
		err  bool
	}{
		{"trims", "  Kemang  ", 50, "Kemang", false},
		{"collapses whitespace", "Jakarta    Selatan", 50, "Jakarta Selatan", false},
		{"newlines become spaces", "line one\nline two", 50, "line one line two", false},
		{"strips control characters", "Kemang\x00\x07", 50, "Kemang", false},
		{"empty is allowed", "", 50, "", false},
		{"over the limit is rejected", strings.Repeat("a", 51), 50, "", true},
		{"exactly the limit passes", strings.Repeat("a", 50), 50, strings.Repeat("a", 50), false},
		{"invalid UTF-8 is rejected", string([]byte{0xff, 0xfe}), 50, "", true},
		{"no limit", strings.Repeat("a", 5000), 0, strings.Repeat("a", 5000), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Text("field", tc.in, tc.max)
			if (err != nil) != tc.err {
				t.Fatalf("err = %v, wantErr %v", err, tc.err)
			}
			if !tc.err && got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// A bidi override can make a stored value display as something it is not.
func TestTextStripsBidiOverrides(t *testing.T) {
	got, err := Text("label", "Home‮evil", 50)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(got, '‮') {
		t.Errorf("bidi override survived: %q", got)
	}
	if got != "Homeevil" {
		t.Errorf("got %q, want the visible characters preserved", got)
	}
}

// The rule is reject, never silently repair — the length case must error rather
// than truncate, or a customer's address quietly loses its house number.
func TestTextRejectsRatherThanTruncates(t *testing.T) {
	_, err := Text("address_line", strings.Repeat("x", 300), 200)
	if err == nil {
		t.Fatal("an over-long value must be rejected, not truncated")
	}
	var se *Error
	if !asSanitizeError(err, &se) || se.Field != "address_line" {
		t.Errorf("error must name the field, got %v", err)
	}
}

func TestRequired(t *testing.T) {
	if _, err := Required("label", "   ", 50); err == nil {
		t.Error("whitespace-only must fail a required field")
	}
	if got, err := Required("label", " Office ", 50); err != nil || got != "Office" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		in   string
		want string
		err  bool
	}{
		{"Steven@Example.COM", "steven@example.com", false},
		{"  ven@evermore.co.id ", "ven@evermore.co.id", false},
		{"no-at-sign", "", true},
		{"two@@at.com", "", true},
		{"@nolocal.com", "", true},
		{"nodomain@", "", true},
		{"no@tld", "", true},
		{"spa ce@x.com", "", true},
		{"", "", true},
	}
	for _, tc := range tests {
		got, err := Email("email", tc.in, 254)
		if (err != nil) != tc.err {
			t.Errorf("%q: err = %v, wantErr %v", tc.in, err, tc.err)
			continue
		}
		if !tc.err && got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPhoneNormalizesIndonesianNumbers(t *testing.T) {
	tests := []struct {
		in   string
		want string
		err  bool
	}{
		{"081234567890", "+6281234567890", false},
		{"+62 812-3456-7890", "+6281234567890", false},
		{"6281234567890", "+6281234567890", false},
		{"0812 3456 7890", "+6281234567890", false},
		{"(0812) 3456-7890", "+6281234567890", false},
		{"12345", "", true},              // too short, and no known prefix
		{"081234567890123456", "", true}, // too long
		{"0812-ABCD", "", true},          // letters
		{"", "", true},
	}
	for _, tc := range tests {
		got, err := Phone("phone", tc.in)
		if (err != nil) != tc.err {
			t.Errorf("%q: err = %v, wantErr %v", tc.in, err, tc.err)
			continue
		}
		if !tc.err && got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEnumIsAnAllowList(t *testing.T) {
	if _, err := Enum("source", "WIDGET", "WIDGET", "ADDRESS_FORM", "CHECKOUT"); err != nil {
		t.Errorf("a listed value must pass: %v", err)
	}
	if _, err := Enum("source", "DROP TABLE", "WIDGET", "CHECKOUT"); err == nil {
		t.Error("an unlisted value must be rejected")
	}
	if _, err := Enum("source", "widget", "WIDGET"); err == nil {
		t.Error("the allow-list is exact — case must matter")
	}
}

func TestSlug(t *testing.T) {
	tests := []struct {
		in  string
		err bool
	}{
		{"weight-loss", false},
		{"paket-20", false},
		{"Weight Loss", true},
		{"-leading", true},
		{"trailing-", true},
		{"under_score", true},
		{"", true},
	}
	for _, tc := range tests {
		if _, err := Slug("slug", tc.in, 60); (err != nil) != tc.err {
			t.Errorf("%q: err = %v, wantErr %v", tc.in, err, tc.err)
		}
	}
}

// Every report in this system exports to Excel, so a delivery note is an
// attack surface.
func TestCSVCellNeutralisesFormulas(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"=cmd|'/c calc'!A1", "'=cmd|'/c calc'!A1"},
		{"+1+1", "'+1+1"},
		{"-1+1", "'-1+1"},
		{"@SUM(A1)", "'@SUM(A1)"},
		{"Jl. Kemang Raya 12", "Jl. Kemang Raya 12"},
		{"", ""},
		{"grey gate, ring the bell", "grey gate, ring the bell"},
	}
	for _, tc := range tests {
		if got := CSVCell(tc.in); got != tc.want {
			t.Errorf("CSVCell(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func asSanitizeError(err error, target **Error) bool {
	e, ok := err.(*Error)
	if ok {
		*target = e
	}
	return ok
}
