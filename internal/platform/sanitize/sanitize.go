// Package sanitize normalises and bounds untrusted input, and encodes it safely
// on the way out.
//
// Steven's rule (99 §7): validate and sanitize on BOTH sides. The frontend does
// it for feedback; this package exists because the frontend can be bypassed
// with curl, so the server re-checks everything from scratch and treats the
// client as hostile.
//
// Two halves, and both are needed:
//
//   - On the way IN: normalize, then validate, then REJECT — never silently
//     repair, because a silent repair hides an attack and surprises the user.
//   - On the way OUT: encode for the context the value lands in. Sanitizing
//     input alone does not stop XSS or a spreadsheet formula; encoding at the
//     point of output does.
package sanitize

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Error is a rejected input, naming the field so the API can point at it.
type Error struct {
	Field  string
	Reason string
}

func (e *Error) Error() string { return e.Field + ": " + e.Reason }

func reject(field, reason string) error { return &Error{Field: field, Reason: reason} }

// Text normalizes and bounds a free-text field.
//
// Order matters: normalize first, then validate. Otherwise the same value
// passes one check and fails another — a trailing space or a decomposed
// accent makes two spellings of one string.
//
// It removes control characters (including the bidirectional-override runes
// used to make a string display as something other than what it is), collapses
// runs of whitespace, NFC-normalizes, and enforces a maximum rune count. It
// does NOT strip HTML: the value is stored as the user typed it and escaped at
// the point of output, which is the only place the correct escaping is known.
func Text(field, in string, maxRunes int) (string, error) {
	if !utf8.ValidString(in) {
		return "", reject(field, "contains invalid UTF-8")
	}

	var b strings.Builder
	b.Grow(len(in))
	lastSpace := false
	for _, r := range in {
		switch {
		case r == '\n' || r == '\r' || r == '\t' || unicode.IsSpace(r):
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
		case unicode.IsControl(r), isBidiOverride(r):
			// Dropped rather than rejected: they are invisible, so a user
			// pasting from Word cannot see what to remove. Everything
			// meaningful survives.
			continue
		default:
			b.WriteRune(r)
			lastSpace = false
		}
	}

	out := norm.NFC.String(strings.TrimSpace(b.String()))
	if maxRunes > 0 && utf8.RuneCountInString(out) > maxRunes {
		return "", reject(field, fmt.Sprintf("must be %d characters or fewer", maxRunes))
	}
	return out, nil
}

// Required is Text plus a non-empty check.
func Required(field, in string, maxRunes int) (string, error) {
	out, err := Text(field, in, maxRunes)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", reject(field, "is required")
	}
	return out, nil
}

// Email normalizes an address for storage and comparison: trimmed and
// lower-cased, so one person cannot register twice with different casing.
// The column is CITEXT as well — the database is the last line, this is the
// first.
func Email(field, in string, maxRunes int) (string, error) {
	out, err := Required(field, in, maxRunes)
	if err != nil {
		return "", err
	}
	out = strings.ToLower(out)
	at := strings.IndexByte(out, '@')
	if at <= 0 || at == len(out)-1 || strings.Contains(out, " ") {
		return "", reject(field, "is not a valid email address")
	}
	if strings.Count(out, "@") != 1 {
		return "", reject(field, "is not a valid email address")
	}
	dot := strings.LastIndexByte(out, '.')
	if dot < at {
		return "", reject(field, "is not a valid email address")
	}
	return out, nil
}

// Phone normalizes an Indonesian mobile number to +62 form.
//
// It rejects rather than guesses when the number is not plausible: a delivery
// to the wrong phone number is a failed delivery, and "repairing" a number
// into something dialable but wrong is worse than refusing it.
func Phone(field, in string) (string, error) {
	var digits strings.Builder
	plus := false
	for i, r := range strings.TrimSpace(in) {
		switch {
		case r == '+' && i == 0:
			plus = true
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')' || r == '.':
			// Formatting a human typed; ignored.
		default:
			return "", reject(field, "may only contain digits and + - ( ) spaces")
		}
	}
	d := digits.String()
	switch {
	case plus && strings.HasPrefix(d, "62"):
		// already international
	case strings.HasPrefix(d, "62"):
		// bare 62…
	case strings.HasPrefix(d, "0"):
		d = "62" + strings.TrimPrefix(d, "0")
	default:
		return "", reject(field, "must start with 0, 62 or +62")
	}
	if len(d) < 10 || len(d) > 15 {
		return "", reject(field, "is not a plausible phone number")
	}
	return "+" + d, nil
}

// Enum checks a value against an allow-list. Allow-list, never deny-list:
// a deny-list is a guess about what is dangerous, an allow-list is a statement
// about what is valid.
func Enum(field, in string, allowed ...string) (string, error) {
	v := strings.TrimSpace(in)
	for _, a := range allowed {
		if v == a {
			return v, nil
		}
	}
	return "", reject(field, "must be one of: "+strings.Join(allowed, ", "))
}

// Slug bounds a URL-safe identifier.
func Slug(field, in string, maxRunes int) (string, error) {
	v := strings.ToLower(strings.TrimSpace(in))
	if v == "" {
		return "", reject(field, "is required")
	}
	if maxRunes > 0 && utf8.RuneCountInString(v) > maxRunes {
		return "", reject(field, fmt.Sprintf("must be %d characters or fewer", maxRunes))
	}
	for _, r := range v {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return "", reject(field, "may only contain lowercase letters, digits and hyphens")
		}
	}
	if strings.HasPrefix(v, "-") || strings.HasSuffix(v, "-") {
		return "", reject(field, "may not start or end with a hyphen")
	}
	return v, nil
}

// CSVCell makes a value safe to write into a spreadsheet export.
//
// A cell beginning =, +, - or @ is a FORMULA in Excel, Sheets and LibreOffice.
// A customer whose delivery note is `=cmd|'/c calc'!A1` becomes code execution
// on the machine of whoever opens the export — and every report in this system
// exports to Excel (PROMPT §12). Prefixing with an apostrophe forces the cell
// to be read as text, and the value the user typed is preserved.
func CSVCell(in string) string {
	if in == "" {
		return in
	}
	switch in[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + in
	}
	return in
}

// isBidiOverride reports the Unicode direction-override runes, which can make a
// stored string render as something other than what it is — an "admin" label on
// a value that is not.
func isBidiOverride(r rune) bool {
	switch r {
	case '‪', '‫', '‬', '‭', '‮',
		'⁦', '⁧', '⁨', '⁩', '‏', '‎':
		return true
	}
	return false
}
