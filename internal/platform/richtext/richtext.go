// Package richtext sanitises the HTML that the back office's WYSIWYG editor
// produces.
//
// This is the one place in the product where a stored value is rendered into a
// page WITHOUT escaping, which is the whole point of rich text and also the
// only way to get script into these pages. So the rules are strict and stated
// here rather than spread across call sites:
//
//   - An ALLOWLIST, never a blocklist. Anything not named is removed. A
//     blocklist is a list of the attacks somebody thought of.
//   - No <script>, no <style>, no <iframe>, no event handlers, no inline
//     styles. Formatting comes from the site's stylesheet, so an editor cannot
//     paste a colour that fails contrast or a font that is not the brand's.
//   - Links are http(s) or mailto only, and every external one is rewritten
//     with rel="nofollow noopener noreferrer" — target=_blank without noopener
//     hands the opened tab a reference back into this page.
//   - Sanitised on the way IN and again on the way OUT (CLAUDE.md §4). The
//     second pass is cheap for a handful of short strings and means a value
//     written by an older, looser build — or straight into the database — can
//     still not reach a page.
//
// bluemonday rather than a hand-written cleaner: HTML sanitisation is a
// problem with a long tail of parser-confusion bugs, and hand-rolling one is
// how those bugs get shipped.
package richtext

import (
	"html/template"
	"strings"
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

var (
	once   sync.Once
	policy *bluemonday.Policy
)

// Policy builds the allowlist once and reuses it; bluemonday policies are
// safe for concurrent use.
func Policy() *bluemonday.Policy {
	once.Do(func() {
		p := bluemonday.NewPolicy()

		// Block-level structure an editor genuinely needs.
		p.AllowElements("p", "br", "h2", "h3", "h4",
			"ul", "ol", "li", "blockquote")
		// Inline emphasis. Both the semantic and the presentational tags,
		// because contenteditable emits <b>/<i> whatever the toolbar asks for.
		p.AllowElements("strong", "b", "em", "i", "u", "span")

		// Links, tightly bounded.
		p.AllowAttrs("href").OnElements("a")
		p.AllowURLSchemes("http", "https", "mailto")
		p.RequireParseableURLs(true)
		p.RequireNoFollowOnLinks(true)
		p.AddTargetBlankToFullyQualifiedLinks(true)
		p.RequireNoReferrerOnFullyQualifiedLinks(true)

		// NOT allowed, and each for a reason:
		//   style/class — an editor must not be able to override the design
		//                 system's colours, which are contrast-checked
		//   img/iframe  — an image is an upload with a size, a type and alt
		//                 text, not a URL typed into a text box
		//   id          — a duplicate id breaks the page's own anchors
		policy = p
	})
	return policy
}

// Clean returns HTML safe to store and to render.
func Clean(in string) string {
	return Policy().Sanitize(in)
}

// Render sanitises and marks the result as trusted for html/template.
//
// This is the ONLY function in the codebase that should produce a
// template.HTML from stored input. Anything else handing user text to
// template.HTML is a cross-site-scripting hole.
func Render(in string) template.HTML {
	return template.HTML(Clean(in))
}

// PlainText strips every tag, for places that need the words without the
// markup — a meta description, a search index, an email subject.
func PlainText(in string) string {
	return strings.TrimSpace(bluemonday.StrictPolicy().Sanitize(in))
}
