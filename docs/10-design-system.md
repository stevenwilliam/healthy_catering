# Design System — Evermore

**Version:** 0.1 (brand received, product undefined)
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

- Asset: `docs/design_guideline/logo.png` (2000×525, transparent)
- Ink: **Nourish Green deep, `#1C3D34`** — the same value as the primary token
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
| `--nourish-deep` | `#1C3D34` | the logo ink; the workhorse |
| `--nourish` | `#468973` | mid green |

### 2.2 Secondary — Restore Beige

| Token | Hex | Note |
|---|---|---|
| `--beige` | `#FFFAE0` | page canvas |
| `--beige-deep` | `#CCBDAA` | borders, muted surfaces |

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
   label, a link, a price or a button. If an "Energize Orange" button is
   wanted, it needs dark ink on the orange, not white — verify before shipping.
3. **The light tertiaries are backgrounds, not inks.** Dark ink on them is fine;
   they are never the foreground.
4. Colour is **never the only signal**. A state carries an icon or a word too.

The four passing colours give a full semantic set without inventing anything:
Berry for destructive, Blue for informational, Brown for warning-ish/neutral
emphasis, Nourish deep for primary.

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

- [ ] **Reversed-out logo** for dark and Nourish-Green fills. The supplied PNG
      is dark ink only and vanishes on the primary colour.
- [ ] **Is Evermore the customer-facing name**, or an internal brand for the
      `healthy_catering` product? The repo and the brand do not match.
- [ ] **Erode licence** confirmed for web embedding.
- [ ] Dark mode: the guidelines describe one light palette. A dark theme needs
      its own token set and its own contrast pass, not an inversion.
- [ ] Logo clear-space and minimum size — not in the supplied pages.
- [ ] Page 13 of the guidelines ("Logo on Color Palette") is referenced by the
      colour page for recommended combinations, and was not supplied.
