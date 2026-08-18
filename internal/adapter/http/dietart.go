package http

import "html/template"

// dietArt is the contextual corner mark on each diet-type card (Steven,
// 2026-08-18: "give background image in up right corner contextual to the
// subject").
//
// Drawn rather than photographed, for two reasons. There is no photograph for
// any of the six, and `diet_type.hero_image_key` — the column meant for one —
// has no object storage behind it yet (M9 is 🟡 for exactly that). Flat
// single-colour marks in currentColor also tint themselves from the card's ink
// token, so they cannot drift from the palette.
//
// They are DECORATION: aria-hidden, no meaning that the card's heading does
// not already carry, and drawn at low opacity behind the text. Anything a
// customer needs to know is in the words.
//
// Keyed by slug. An unknown slug — a diet type added later in the admin —
// falls back to the bowl rather than rendering a hole, which is why the
// fallback is a generic "food" mark rather than one of the six.
var dietArtBySlug = map[string]template.HTML{
	// A bowl with a leaf rising out of it.
	"healthy": template.HTML(`
		<path d="M6,34 h52 a26,26 0 0 1 -52,0 Z"/>
		<path d="M34,30 C34,15 45,7 56,6 C56,19 47,30 34,30 Z"/>
		<path d="M34,32 C36,24 42,17 50,12" fill="none" stroke="currentColor"
		      stroke-width="3" stroke-linecap="round"/>`),

	// Arrow up off a baseline.
	"weight-gain": template.HTML(`
		<path d="M32,8 L52,32 H40 V48 H24 V32 H12 Z"/>
		<rect x="12" y="52" width="40" height="6" rx="3"/>`),

	// Arrow down onto a baseline.
	"weight-loss": template.HTML(`
		<rect x="12" y="6" width="40" height="6" rx="3"/>
		<path d="M32,58 L12,34 H24 V18 H40 V34 H52 Z"/>`),

	// A dumbbell.
	"high-protein": template.HTML(`
		<rect x="20" y="29" width="24" height="6" rx="3"/>
		<rect x="12" y="21" width="8" height="22" rx="3"/>
		<rect x="44" y="21" width="8" height="22" rx="3"/>
		<rect x="5" y="26" width="6" height="12" rx="2"/>
		<rect x="53" y="26" width="6" height="12" rx="2"/>`),

	// A heart with a medical cross cut out of it. fill-rule evenodd is what
	// makes the cross a HOLE rather than a second solid shape on top.
	"special-diet": template.HTML(`
		<path fill-rule="evenodd" d="M32,56 C9,39 7,24 17,16 C25,10 31,16 32,20
		      C33,16 39,10 47,16 C57,24 55,39 32,56 Z
		      M28,22 h8 v7 h7 v8 h-7 v7 h-8 v-7 h-7 v-8 h7 Z"/>`),

	// An avocado half: silhouette with the stone as a hole.
	"keto": template.HTML(`
		<path fill-rule="evenodd" d="M32,5 C45,5 53,21 53,34 C53,48 44,59 32,59
		      C20,59 11,48 11,34 C11,21 19,5 32,5 Z
		      M32,25 a10,10 0 1 0 0.1,0 Z"/>`),
}

// fallbackDietArt is the bowl on its own — a diet type the admin adds later
// still gets a mark rather than an empty corner.
var fallbackDietArt = template.HTML(`
		<path d="M6,34 h52 a26,26 0 0 1 -52,0 Z"/>
		<path d="M22,26 h20 a10,10 0 0 0 -20,0 Z"/>`)

func dietArt(slug string) template.HTML {
	if art, ok := dietArtBySlug[slug]; ok {
		return art
	}
	return fallbackDietArt
}
