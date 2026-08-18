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
- **On a dark or Nourish-Green fill the mark disappears.** A reversed-out
  (beige or white) version is needed before the first dark header is built.
  It does not exist yet — see §5.

## 2. Palette

The guidelines structure colour in three levels and say application across them
is "intentionally dynamic and flexible". That flexibility is a design licence,
not an accessibility one — see the ratios below.

### 2.1 Primary — Nourish Green

| Token | Hex | Note |
|---|---|---|
| `--nourish-deep` | `#1C3D34` | the logo ink; the workhorse; masthead + footer fill |
| `--nourish` | `#468973` | **the page ground** since 2026-08-18 — see §2.7 |

### 2.2 Secondary — Restore Beige

| Token | Hex | Note |
|---|---|---|
| `--beige` | `#FFFAE0` | the content **sheet** — where body copy is allowed |
| `--beige-deep` | `#CCBDAA` | borders, muted surfaces; the footer's muted ink |

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

### 2.7 The green ground (Steven, 2026-08-18)

The page ground moved from Restore Beige to **Nourish Green `#468973`**. That
one change re-derives the layout rules, because `#468973` sits at relative
luminance 0.204 — almost exactly midway between black and white — and **nothing
reaches AA 4.5 on it**:

| On `#468973` | Ratio | Verdict |
|---|---|---|
| white `#FFFFFF` | **4.13** | AA large text / UI boundary only |
| beige `#FFFAE0` | **3.93** | AA large text / UI boundary only |
| deep ink `#1C3D34` | **2.88** | fails everything |

White is the best case available and it is still short of 4.5. So the rule is:

> **The green is a ground, not a surface for reading.** Display type sits on it.
> Every paragraph sits on a beige sheet (`.panel`, ink 11.32) or a white card
> (ink 11.89).

What that licenses, and what it forbids:

| Element | On the ground? | Why |
|---|---|---|
| `h1` 44–64px, `h2` 32px, beige | yes | ≥24px is WCAG "large text", 3.93 ✓ |
| `.lede` 20px **at weight 600** | yes | ≥18.66px bold is large text, 3.93 ✓ |
| the 13px `.eyebrow` | **no** — it is a beige pill | 13px needs 4.5; as a pill it is ink-on-beige at 11.32 ✓ |
| any `<p>` at body size | **no** — wrap it in `.panel` | 3.93 is not AA at 17px |
| the CTA button | inverted: beige fill, deep ink | a deep-green fill is 2.88 against the ground and fails 1.4.11 |
| focus rings | beige, not deep ink | deep ink is 2.88 there — an indicator you cannot see |

**The WhatsApp float needed rescuing.** Its `#128C7E` measures **1.00:1**
against `#468973` — the two luminances differ by 0.0003, so the button was one
colour change away from being a shape only a hue-discriminating eye could find.
(WhatsApp's brighter `#25D366` is no better; it was already ruled out at 1.89
on beige.) The fix is a permanent 3px beige ring: 3.93 against the ground and
3.94 against the teal, so the control is bounded whichever it overlaps. That
ring is an accessibility mechanism, not styling — do not remove it.

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

## 4. Conventions carried in

From `99-steven-preference.md`, applied here:

- **Search box on every list.**
- Focus rings always visible; **minimum 44×44px touch targets**.
- Motion is one subtle tier: 200–400ms, ease-out, ≤12px travel, transform and
  opacity only, and `prefers-reduced-motion` zeroes **delays as well as
  durations**.
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
