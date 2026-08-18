// Package i18n is the message catalogue and locale negotiation.
//
// Business-agnostic and portable (CLAUDE.md §2): it knows about locales,
// negotiation and lookup, and nothing about catering. The catalogues
// themselves live with the surface that renders them.
//
// Three locales as of 2026-08-18 (Steven): Indonesian (default), English and
// Simplified Chinese. CLAUDE.md §10 requires "message catalogues from the
// first string, never inline", which is what Catalog is for.
package i18n

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Locale is a supported UI language.
type Locale string

const (
	ID Locale = "id" // Bahasa Indonesia — the default
	EN Locale = "en" // English
	ZH Locale = "zh" // 中文 (Simplified)
)

// Default is what an unrecognised or absent preference falls back to, and what
// a missing translation falls back to. The audience is Jakarta first.
const Default = ID

// Supported is in the order a language selector should list them.
var Supported = []Locale{ID, EN, ZH}

// Info is everything a selector or a <html lang> needs.
type Info struct {
	Locale Locale
	// Tag is BCP-47, for lang= and hreflang=. Chinese carries the script
	// because "zh" alone does not say Simplified or Traditional, and the two
	// are not mutually readable at a glance.
	Tag string
	// Endonym is the language's name IN that language. A selector must show
	// this: someone who has landed on the wrong language cannot read a list
	// written in the language they cannot read.
	Endonym string
	// English is the name in English, for logs, admin screens and alt text.
	English string
	// Prefix is the URL path segment. The default locale has none, so the
	// canonical Indonesian URLs stay exactly what they have always been and no
	// existing link breaks.
	Prefix string
}

var meta = map[Locale]Info{
	ID: {Locale: ID, Tag: "id-ID", Endonym: "Bahasa Indonesia", English: "Indonesian", Prefix: ""},
	EN: {Locale: EN, Tag: "en", Endonym: "English", English: "English", Prefix: "en"},
	ZH: {Locale: ZH, Tag: "zh-Hans", Endonym: "中文", English: "Chinese (Simplified)", Prefix: "zh"},
}

// Meta returns the descriptor for a locale, falling back to the default.
func Meta(l Locale) Info {
	if info, ok := meta[l]; ok {
		return info
	}
	return meta[Default]
}

// All returns every supported locale's descriptor, in selector order.
func All() []Info {
	out := make([]Info, 0, len(Supported))
	for _, l := range Supported {
		out = append(out, meta[l])
	}
	return out
}

// Parse recognises a locale from a code such as "en", "EN", "zh-Hans" or
// "id-ID". It deliberately matches on the primary subtag so that a browser
// sending "zh-TW" or "en-GB" still lands somewhere sensible.
func Parse(s string) (Locale, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return Default, false
	}
	if i := strings.IndexAny(s, "-_"); i >= 0 {
		s = s[:i]
	}
	for _, l := range Supported {
		if string(l) == s {
			return l, true
		}
	}
	return Default, false
}

// FromPath splits a leading locale prefix off a URL path.
//
//	"/en/menu/keto" -> EN, "/menu/keto"
//	"/menu/keto"    -> ID, "/menu/keto"
//
// The remainder always keeps its leading slash, so callers can route on it
// unchanged.
func FromPath(p string) (Locale, string) {
	trimmed := strings.TrimPrefix(p, "/")
	seg, rest, _ := strings.Cut(trimmed, "/")
	for _, l := range Supported {
		if pre := meta[l].Prefix; pre != "" && seg == pre {
			return l, "/" + rest
		}
	}
	return Default, p
}

// Path builds the URL for a page in a given locale. `rest` is the path without
// any locale prefix, e.g. "/menu/keto".
func Path(l Locale, rest string) string {
	if !strings.HasPrefix(rest, "/") {
		rest = "/" + rest
	}
	pre := Meta(l).Prefix
	if pre == "" {
		return rest
	}
	if rest == "/" {
		return "/" + pre + "/"
	}
	return "/" + pre + rest
}

// FromAcceptLanguage picks the best supported locale from an Accept-Language
// header, honouring q-values. Returns false when the header expresses no
// preference we can serve, so the caller can tell "asked for Indonesian" apart
// from "did not ask".
func FromAcceptLanguage(header string) (Locale, bool) {
	if strings.TrimSpace(header) == "" {
		return Default, false
	}
	type pref struct {
		locale Locale
		q      float64
		order  int
	}
	var prefs []pref
	for i, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag, params, _ := strings.Cut(part, ";")
		q := 1.0
		if params != "" {
			if _, qs, ok := strings.Cut(params, "q="); ok {
				if v, err := strconv.ParseFloat(strings.TrimSpace(qs), 64); err == nil {
					q = v
				}
			}
		}
		// q=0 means "explicitly not this one".
		if q <= 0 {
			continue
		}
		if l, ok := Parse(tag); ok {
			prefs = append(prefs, pref{l, q, i})
		}
	}
	if len(prefs) == 0 {
		return Default, false
	}
	sort.SliceStable(prefs, func(a, b int) bool {
		if prefs[a].q != prefs[b].q {
			return prefs[a].q > prefs[b].q
		}
		return prefs[a].order < prefs[b].order
	})
	return prefs[0].locale, true
}

// Catalog maps a message key to its translations.
//
// Lookup falls back to the default locale rather than returning empty, because
// a missing Chinese string should show the Indonesian one, not a blank space
// where a button label belongs. A key with no entry at all returns the key
// itself, which is ugly on purpose — it shows up immediately in review instead
// of shipping as silence.
type Catalog map[string]map[Locale]string

// T looks up a key.
func (c Catalog) T(l Locale, key string) string {
	m, ok := c[key]
	if !ok {
		return key
	}
	if s := m[l]; s != "" {
		return s
	}
	if s := m[Default]; s != "" {
		return s
	}
	return key
}

// Tf looks up a key and formats it.
func (c Catalog) Tf(l Locale, key string, args ...any) string {
	return fmt.Sprintf(c.T(l, key), args...)
}

// Missing reports keys that lack a translation for any supported locale.
// Tests assert this is empty, so an untranslated string cannot reach
// production quietly — the catalogue is the contract.
func (c Catalog) Missing() map[string][]Locale {
	out := map[string][]Locale{}
	for key, m := range c {
		for _, l := range Supported {
			if strings.TrimSpace(m[l]) == "" {
				out[key] = append(out[key], l)
			}
		}
	}
	return out
}
