# Design System — Evermore

**Version:** 0.2 (brand received and re-verified against the source PNGs; product undefined)
**Source:** `docs/design_guideline/` — "evermore Mini Brand Guidelines", pages
17 (colour) and 19 (typeface), supplied by Steven.

This document is the **engineering reading** of those guidelines: the same
decisions expressed as tokens, with the contrast maths done. The PNGs are the
authority on intent; this file is the authority on what ships.

---

## 1. Brand

The wordmark is **"evermore"**, set lowercase in a single weight of deep green,
with a **reversed final `e`** — the mark's one distinctive move. It is not a
typo and must never be "corrected".

- Asset: `docs/design_guideline/logo.png` — **7582×1989**, RGBA, transparent
- Ink: **Nourish Green deep, `#1C3D34`** — sampled from the file as
  `rgba(28, 61, 52, 255)`, the same value as the primary token
- Verified by decoding the PNG glyph by glyph: eight letterforms, and **only the
  final `e` is mirrored**. Both `r`s are in normal orientation — at small sizes
  the second `r` reads as mirrored next to the flipped `e`, and it is not. Do
  not "fix" either one.
- Never recolour it to a third colour, never restretch it, never redraw the
  letterforms.
- **On a dark or Nourish-Green fill the mark disappears**, so a reversed-out
  version is required. It now exists: `scripts/mklogo.py` derives it from the
  supplied artwork — see §3.2.

## 2. Palette

The guidelines structure colour in three levels and say application across them
is "intentionally dynamic and flexible". That flexibility is a design licence,
not an accessibility one — see the ratios below.

### 2.1 Primary — Nourish Green

| Token | Hex | Note |
|---|---|---|
| `--nourish-deep` | `#1C3D34` | the logo ink; **the page ground** since 2026-08-18 — see §2.7 |
| `--nourish` | `#468973` | **the bars** — masthead and footer — see §2.7 |

### 2.2 Secondary — Restore Beige

| Token | Hex | Note |
|---|---|---|
| `--beige` | `#FFFAE0` | primary ink on both greens, and the **sheet** for panels and the app |
| `--beige-deep` | `#CCBDAA` | borders and muted text **on the deep ground only** — 2.25 on a bar |

### 2.3 Tertiary

| Name | Dark | Light |
|---|---|---|
| Ground Brown | `#613F37` | `#A36E50` |
| Hydrate Blue | `#2E55A3` | `#B6DAFA` |
| Energize Orange | `#E0782D` | `#FFBC8F` |
| Revive Berry | `#91253D` | `#CC6883` |

### 2.4 Contrast — read this before using a colour

Calculated, not eyeballed. **AA body text needs 4.5:1.**

| Colour | on `#FFFAE0` | on `#FFFFFF` | White text on it |
|---|---:|---:|---:|
| `#1C3D34` Nourish deep | **11.32** | **11.89** | **11.89** |
| `#613F37` Ground Brown | **8.79** | **9.23** | **9.23** |
| `#91253D` Revive Berry | **7.89** | **8.28** | **8.28** |
| `#2E55A3` Hydrate Blue | **6.79** | **7.13** | **7.13** |
| `#468973` Nourish mid | ✗ 3.93 | ✗ 4.13 | ✗ 4.13 |
| `#A36E50` Brown light | ✗ 4.08 | ✗ 4.28 | — |
| `#CC6883` Berry light | ✗ 3.40 | ✗ 3.57 | — |
| `#E0782D` Energize Orange | ✗ 2.90 | ✗ 3.05 | ✗ 3.05 |

**Consequences, decided now so they are not rediscovered later:**

1. **`#468973` is not a text colour and not a button fill.** It fails in both
   directions — 4.13 as ink on white, and 4.13 for white ink on it. Use
   `#1C3D34` for green text and green fills. Reserve `#468973` for large
   decorative areas, illustration and non-informational shapes.
2. **`#E0782D` fails badly (2.90–3.05).** It is a highlight colour, not a
   label, a link, a price or a button. Dark ink on the orange was checked
   rather than assumed: `#1C3D34` on `#E0782D` is **3.90** — enough for large
   text (≥24px, or ≥19px bold) and for non-text UI under 1.4.11, **not** enough
   for body text or a small button label. `#000000` on `#E0782D` reaches
   **6.89**. So: an orange button is allowed only with near-black ink at a large
   size, and never with white ink.
3. **The light tertiaries are backgrounds, not inks.** Dark ink on them is fine;
   they are never the foreground. Measured with `#1C3D34` ink: Blue light
   **8.15** ✓, Orange light **7.27** ✓, Berry light ✗ 3.33, Brown light ✗ 2.78.
   Only the blue and orange tints take body text.
4. Colour is **never the only signal**. A state carries an icon or a word too.

### 2.5 On a Nourish-Green surface

The primary green is going to be the header, the footer and every dark panel,
so the inks that survive on it are decided here rather than per component.
Ink on `#1C3D34`:

| Ink | Ratio | Verdict |
|---|---:|---|
| `#FFFAE0` Beige | **11.32** | ✓ primary text on dark |
| `#B6DAFA` Blue light | **8.15** | ✓ links / informational on dark |
| `#FFBC8F` Orange light | **7.27** | ✓ **this is how Energize Orange earns its place** — as its tint on the dark green, not as the dark orange on light |
| `#CCBDAA` Beige deep | **6.47** | ✓ secondary/muted text on dark |
| `#CC6883` Berry light | ✗ 3.33 | large text only |
| `#468973` Nourish mid | ✗ 2.88 | never text on dark green |

### 2.6 Borders and non-text contrast (1.4.11 — 3:1)

`--beige-deep` `#CCBDAA` on the beige canvas is **1.75**. That is invisible to a
low-vision user, so **`#CCBDAA` is not an input border, a focus ring or any
boundary that carries meaning** — it is a divider between filled surfaces only.
Form field outlines, focus rings and control boundaries use `#1C3D34`, solid or
at an opacity that still measures ≥3:1. `#468973` on beige is 3.93, so it is a
legal *non-text* boundary even though it is not a legal text colour.

The four passing colours give a full semantic set without inventing anything:
Berry for destructive, Blue for informational, Brown for warning-ish/neutral
emphasis, Nourish deep for primary.

### 2.7 The two grounds (Steven, 2026-08-18)

The page ground moved from Restore Beige to Nourish Green `#468973`, and then
the two greens were **swapped**: the PAGE is deep `#1C3D34` and the BARS — the
masthead and the fixed footer — are mid `#468973`.

That swap matters because `#468973` sits at relative luminance 0.204, almost
exactly midway between black and white, and **nothing in the brand reaches AA
4.5 on it**:

| On mid green `#468973` | Ratio | Verdict |
|---|---|---|
| white `#FFFFFF` | **4.13** | AA large text / UI boundary only |
| beige `#FFFAE0` | **3.93** | AA large text / UI boundary only |
| deep ink `#1C3D34` | **2.88** | fails everything |
| `#CCBDAA` | **2.25** | fails everything — never use it on a bar |
| black `#000000` | 5.09 | passes, but is not a brand colour |

On the deep ground everything is comfortable:

| On deep green `#1C3D34` | Ratio | Verdict |
|---|---|---|
| beige `#FFFAE0` | **11.32** | AAA |
| white `#FFFFFF` | **11.89** | AAA |

So the rules are:

> **The page is deep green and body copy sits straight on it in beige.** Panels
> and cards are design decisions here, not accessibility crutches.
>
> **The bars are mid green and everything on them is LARGE text** — `--text-bar`
> is 19px, the first whole pixel above WCAG's 18.66px threshold, at weight 700.
> A 15px nav link on a bar looks fine and is an AA failure.

Consequences worth knowing before editing anything:

| Element | Rule | Why |
|---|---|---|
| masthead nav, footer, language selector | `--text-bar` @700 | 3.93 is AA for large text only |
| `--on-dark-muted` `#CCBDAA` | never on a bar | 2.25 there; it is a deep-ground token |
| the CTA | beige fill, deep ink | 11.32 on the page ✓ |
| focus rings on the page | beige | 11.32 ✓ |
| the WhatsApp float | keeps its beige ring | see below |

**The WhatsApp float has needed its ring under both grounds, for two different
reasons.** On the mid-green ground `#128C7E` measured **1.00:1** — the two were
within 0.0003 of the same luminance, so the button was a shape findable only by
hue. On the deep ground it measures **2.87**, still under the 3:1 that 1.4.11
requires of a control. The permanent 3px beige ring bounds it either way: 11.32
against the page and 3.94 against the teal. Its focus state is **white**, not
beige, because the resting ring is already beige and an outer deep-green ring
would be invisible against the page — focus and resting would look identical.
The ring is an accessibility mechanism, not styling. Do not remove it.

### 2.8 The background — drifting produce (Steven, 2026-08-27)

**Supersedes the tiled motif of 2026-08-19.** That tile was Steven's supplied
artwork, extracted to beige-on-transparent by `scripts/mkpattern` and painted
at 36% of native, peaking at 14.9% alpha. It is retired here; the file and the
generator stay, because **the SPA (`web/src/index.css`) still uses it** and
restyling the app was not part of this change.

**Why it was replaced, beyond the brief.** Its 14.9% peak left beige text at
7.29 — measured, recorded, and correct. But the muted token was never checked
at the same point: `--beige-deep` `#CCBDAA` on that composite is **4.16, below
AA's 4.50**, everywhere the linework crossed muted text. The comment in
`public.css` claiming "6.47 on the ground ✓" was true of the bare ground and
false of the page as it actually rendered.

**What replaced it.** Two fixed layers of produce linework — citrus, leaves,
berries, wheat, avocado, broccoli, carrot — generated by
`scripts/mkfoodpattern.py` from a fixed seed, so re-running produces
byte-identical files and a diff means somebody changed the drawing.

| Layer | Tile | Opacity | Drift |
| --- | --- | --- | --- |
| `body::before` far | 340px | 0.06 | 150s, down-right |
| `body::after` near | 220px | 0.05 | 95s, down-left |

The ground moved from `body` to `html`, because a body background propagates to
the canvas and would paint over anything at `z-index: -1`.

**The alpha is a contrast decision.** Public-page text sits directly on the
ground, so the binding token is `--beige-deep`, not beige:

| Ink | arithmetic bound | lightest pixel rendered | |
| --- | ---: | ---: | --- |
| beige `#FFFAE0` | 8.29 | 8.43 | AAA |
| white `#FFFFFF` | 8.71 | 8.85 | AAA |
| `--beige-deep` `#CCBDAA` | 4.74 | **4.82** | AA — binds |

"Rendered" is the lightest pixel found in a 400×400 capture of the background
with all content hidden (`#335044`). Bound and measurement agree to a
hundredth, and that agreement is the point. An earlier pass at 0.09/0.07
measured 4.89 and looked safe — but that was one paused frame in which no two
strokes crossed. **The bound governs**, because over a 150-second drift across
a whole viewport the crossing does happen.

**Visibility is bought with coverage, not opacity.** The per-pixel peak is what
costs contrast; heavier strokes and larger glyphs put more drawing on screen
without moving the peak. That is why the generator draws at 2.6/2.0 stroke
weight rather than hairlines.

Motion is transform-only on a fixed layer, oversized by exactly one tile and
translating by exactly one tile so the loop is seamless.
`background-position` is deliberately **not** animated — it repaints a
full-viewport image every frame, on a page mostly read on mid-range Android.
`prefers-reduced-motion` stops both layers, leaving the drawing as a static
texture.

## 3. Typography

Two faces, both **self-hosted** — never a font CDN, which leaks every visitor's
IP and page to a third party.

| Role | Family | Use |
|---|---|---|
| Display | **Erode** | titles, headlines, subtitles — "prominent typographic elements" |
| Body / UI | **Inter** | body copy, longer text, smaller-scale applications |

Weights shown in the guidelines: Regular, Medium, Semi Bold, Bold — for both.

**Erode is not on Google Fonts.** It is an Indian Type Foundry face distributed
through Fontshare; download the web package and check its licence terms before
first use. Inter is SIL OFL and can come from the Google Fonts static files, as
in ruuma. Both go in `web/public/fonts/` with their licence beside them.

Serif display against a sans body is the same structure ruuma landed on, and the
reason holds here: the wordmark has character, and setting headings in the body
sans would leave nothing on screen echoing it.

### 3.1 How the two are mixed (reworked 2026-08-18)

The pairing works by **contrast, not blend**: Erode heavy and tight, Inter open
and even, and one small uppercase Inter label bridging them. Erode ships four
discrete cuts, so a display weight must land on 600 or 700 exactly — ask for
650 and the browser synthesises a fake bold, which smears the letterforms the
wordmark is built from.

| Role | Face | Size | Weight | Tracking |
|---|---|---|---|---|
| `h1` | Erode | 44px → 64px ≥768px | 700 | −0.022em |
| `h2` | Erode | 32px | 700 | −0.015em |
| `h3`, card titles | Erode | 24px | 600 | −0.01em |
| `.lede` | Inter | 20px | 600 | — |
| body | Inter | 17px | 450 | — |
| `.eyebrow`, `.when` | Inter | 13px | 700 | +0.1em, uppercase |
| nav, buttons, labels | Inter | 15–17px | 600–700 | — |

Sizes and weights went up a step across the board on Steven's instruction
("more bold and bigger in overall design"). Body is 17px rather than 16 because
Inter at 450 sits on a coloured ground here and needs the extra pixel to hold
its weight. Two of these numbers are **load-bearing for accessibility**, not
taste: the `.lede` at 20px/600 and the `h2` at 32px are what make beige-on-green
legal as large text (§2.7). Reducing either size or weight silently drops that
text below AA.

### 3.2 The wordmark

The supplied artwork is `docs/design_guideline/logo.png` — 7582×1989, flat
`#1C3D34` lettering. `scripts/mklogo.py` reduces it to a coverage mask and
re-emits it in any colour, which is what produces both cuts from one source:

- `web/public/images/evermore-wordmark-light.png` — beige, for the deep-green
  masthead. **This answers §5's reversed-out logo question**: derived, not
  supplied.
- `web/public/images/evermore-wordmark-deep.png` — for light surfaces.
- `web/public/images/og-default.png` — 1200×630 share card.

One trap is recorded in the script: the artwork's ink is `#1C3D34`, whose luma
is 50, so naive darkness-as-coverage caps at 205 and emits a solid stroke at 80%
alpha. The wordmark would then render as a *tint* of whatever sits behind it,
quietly losing the 11.32:1 this document claims for it. Coverage is normalised
against the darkest pixel present so a solid stroke is genuinely solid.

### 2.9 Home motion — the expressive tier (2026-08-27)

The home page ran the subtle tier first and Steven could not see it, which was
the correct outcome of a 12px/400ms rule. **On the home page only**, travel is
now 28px and durations 400–560ms. Every other page keeps the subtle tier in §4.

Still **no JavaScript**. The CSP would permit a same-origin script; nothing here
needs one, and the home page is the LCP path.

| What | Motion | Driven by |
| --- | --- | --- |
| `.hero-copy` children | rise 28px + fade, 90ms apart | time, 460ms |
| `.hero-art` (the frame) | fade | time, 560ms |
| `.hero-art img` | parallax pan ±3.2% inside a 1.08 scale | scroll, `view()` |
| price table `tbody tr` | rise 28px, no fade | scroll, `cover 25%..70%` |
| `.check .panel`, `.diets .card-diet` | rise 28px, no fade | scroll, `cover 25%..70%` |
| section-head rule | `scaleX` wipe from the left | scroll, `cover 20%..60%` |
| `.diets .card-diet` hover/focus-within | `scale(1.02)` + deeper shadow | interaction |

**Not animated, deliberately: the price figures.** A count-up renders a money
value that is briefly wrong and starts at zero, and this project has already
shipped `Rp 0` against real prices once. The table reveals by row instead.

Three things here are one word away from silently not working, and each was
found by measurement rather than by looking:

- **`overflow: clip`, not `hidden`, on `.hero-art`.** Both crop the oversized
  image identically, but `hidden` makes the element a scroll container, and
  `view()` resolves against the nearest scrollport. With `hidden` the image's
  timeline was measured against a frame that never scrolls: pinned at
  `currentTime` 49.9967%, parallax frozen at every scroll offset.
- **A scroll-driven animation needs a non-zero `animation-duration`.**
  `animation: hero-pan linear both` leaves the shorthand's duration at its
  initial `0s`, and there is then no progress for a range to map onto.
- **Individual `translate`/`scale`, not the `transform` shorthand.** A card
  carries both a scroll reveal and a hover lift; written as `transform` the
  animation's fill state wins permanently and the hover does nothing at all.

**Ranges are swept, never guessed.** `entry` percentages play entirely at the
viewport's bottom edge. Measured at 390×844, card top in px as the motion runs:
`cover 5%..35%` → 785→616 (invisible), `cover 25%..70%` → 587→336 (the middle
of the screen). Re-sweep after any change.

**Motion is never the reason content is visible.** Base state painted,
animation additive, all of it inside `prefers-reduced-motion: no-preference`.
Anything on a *scroll* timeline moves but never fades: a scroll timeline only
advances while active, so a faded element would be stranded invisible if the
range were mis-scoped or the container did not scroll. Time-based animations
always finish, so those may fade. Measured 2026-08-27 at 390px: under
`prefers-reduced-motion: reduce` the first viewport differs from the
pre-animation build by **113 pixels of 329,160, max channel delta 2** — the
ribbon shimmer at a different phase, nothing else.

The section-head rule is a **static** design element declared outside the motion
block, so with motion off it is simply present; only its wipe is animated.

`scripts/verify-motion.js` asserts the invariant on every element.

## 4. Conventions carried in

From `99-steven-preference.md`, applied here:

- **Search box on every list.**
- Focus rings always visible; **minimum 44×44px touch targets**.
- Motion is one subtle tier: 200–400ms, ease-out, ≤12px travel, transform and
  opacity only, and `prefers-reduced-motion` zeroes **delays as well as
  durations**. **Exception, home page only (§2.9):** 400–560ms and 28px, plus
  a scroll-driven parallax and hover lift. Nowhere else.
- Anything that writes to the server is a button that **disables itself for the
  life of the request** and **confirms before anything irreversible**.
- Every public page ships the SEO baseline (§13 of the preference file).

## 5. Open — needs Steven or a designer

- [x] **Reversed-out logo** for dark and Nourish-Green fills. Derived from the
      supplied artwork by `scripts/mklogo.py` rather than waiting on one —
      see §3.2. Worth a designer's eye, but it is no longer a blocker.
- [x] **Is Evermore the customer-facing name** — yes; `healthy_catering` is the
      repo codename only (CLAUDE.md §10).
- [x] **Erode licence** confirmed for web embedding (Steven, 2026-08-13); the
      terms and the no-subsetting constraint are in `web/public/fonts/fonts.css`.
- [ ] **Logo clear-space** is being approximated. The masthead gives the
      wordmark its own flex cell and the artwork is trimmed to its bounding box,
      so clear space is whatever the layout gap happens to be — page 13 of the
      guidelines would settle it.
- [ ] Dark mode: the guidelines describe one light palette. A dark theme needs
      its own token set and its own contrast pass, not an inversion.
- [ ] Logo clear-space and minimum size — not in the supplied pages.
- [ ] Page 13 of the guidelines ("Logo on Color Palette") is referenced by the
      colour page for recommended combinations, and was not supplied.
