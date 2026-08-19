package http

import (
	"regexp"
	"sort"
	"testing"
)

// Every catalogue key the templates ask for must exist.
//
// This is the gap TestPublicMessagesComplete cannot see: that test proves the
// keys IN the catalogue are fully translated, but says nothing about a key a
// template uses and the catalogue does not have. Catalog.T echoes an unknown
// key back, so the failure ships as the literal string "price.col_amount"
// rendered in a table header — which is exactly what happened on 2026-08-19,
// when a string replacement silently no-opped after gofmt realigned the map
// and nothing noticed until the page was read.
func TestEveryTemplateKeyExists(t *testing.T) {
	// Matches {{t .L "key"}}, {{t $.L "key"}}, {{c .Copy .L "key"}},
	// {{chtml $.Copy $.L "key"}} — i.e. the last quoted argument of a lookup.
	re := regexp.MustCompile(`\{\{-?\s*(?:t|c|chtml)\s+[^"}]*"([a-z0-9_.]+)"`)

	missing := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(publicTemplates, -1) {
		key := m[1]
		if _, ok := publicMessages[key]; !ok {
			missing[key] = true
		}
	}
	if len(missing) == 0 {
		return
	}
	keys := make([]string, 0, len(missing))
	for k := range missing {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Errorf("the templates use %q but publicMessages has no such key — "+
			"it would render as the literal key text", k)
	}
}
