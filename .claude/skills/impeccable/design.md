# Evermore — design reference

The working numbers. `docs/10-design-system.md` carries the reasoning and the
brand-guideline reading; this is the sheet to have open while building, so
nothing here is a judgement call — every ratio below was measured with
`scripts/contrast.py` and can be re-measured.

**This file is per project.** The `impeccable` skill beside it is portable; a
brand is not. A new project gets its own `design.md` in the same place.

---

## 1. Type

| Role | Family | Notes |
| --- | --- | --- |
| Display | **Erode** | titles, headlines, the wordmark's character |
| Body / UI | **Inter** (variable) | everything a customer reads |
| CJK | platform fonts | PingFang SC · Hiragino Sans GB · Microsoft YaHei · Noto Sans CJK SC |

Both are **self-hosted** in `web/public/fonts/`. Never a font CDN — it hands a
third party every visitor's IP and the page they are on.

- **Erode ships four discrete cuts: 400 / 500 / 600 / 700.** Ask for anything
  else and the browser synthesises a fake bold, which smears the letterforms
  the wordmark is built from. Display weights land on 600 or 700 exactly.
- **Erode must not be subset.** The Fontshare licence forbids modifying the
  font software; the usual latin-subsetting trim is a modification.
- **Neither brand face covers CJK.** A Chinese headline is not set in Erode and
  cannot be — it falls through to the platform stack. Chinese pages carry the
  brand's colour, layout and wordmark, not its display voice.

### Scale and weight tokens

| Token | Size | | Token | Weight |
| --- | --- | --- | --- | --- |
| `--text-xs` | 13px | | `--w-display` | 700 (Erode) |
| `--text-sm` | 15px | | `--w-display-sub` | 600 (Erode) |
| `--text-base` | 17px | | `--w-body` | 450 (Inter) |
| `--text-lg` | 20px | | `--w-strong` | 600 |
| `--text-xl` | 24px | | `--w-ui` | 600 |
| `--text-2xl` | 32px | | `--w-eyebrow` | 700 |
| `--text-3xl` | 44px | | | |
| `--text-4xl` | 64px | | | |
| `--text-bar` | **19px** | the bar floor — see §3 | | |

Body is 17px rather than 16 because Inter at 450 sits on a coloured ground and
needs the extra pixel to hold its weight.

---

## 2. Palette

| Token | Hex | Role |
| --- | --- | --- |
| `--nourish-deep` | `#1C3D34` | **the page ground**, the ink on light surfaces, the logo ink |
| `--nourish` | `#468973` | **the bars** — masthead and footer |
| `--beige` | `#FFFAE0` | ink on both greens; the sheet panels sit on |
| `--beige-deep` | `#CCBDAA` | muted ink **on the deep ground only** |
| `--surface` | `#FFFFFF` | cards |
| `--ink-muted` | `#4A5D56` | secondary text on the sheet |
| `--brown-deep` | `#613F37` | warning / neutral emphasis |
| `--blue-deep` | `#2E55A3` | informational |
| `--berry-deep` | `#91253D` | destructive |
| `--blue-light` | `#B6DAFA` | tint; takes deep ink |
| `--orange-light` | `#FFBC8F` | tint; takes deep ink |
| `--orange-deep` | `#E0782D` | **decoration only** |
| `--berry-light` | `#CC6883` | background only |
| `--brown-light` | `#A36E50` | background only |
| gold | `#D4AF37` | the corner ribbon |

---

## 3. Measured contrast — check here before choosing a colour

**On the deep ground `#1C3D34`** — comfortable at any size:

| Ink | Ratio | |
| --- | ---: | --- |
| `#FFFFFF` | 11.89 | AAA |
| `#FFFAE0` beige | 11.32 | AAA |
| `#B6DAFA` blue light | 8.15 | AAA — links on dark |
| `#FFBC8F` orange light | 7.27 | AAA — this is how the orange earns its place |
| `#CCBDAA` beige deep | 6.47 | AA — muted text |

**On the mid-green bars `#468973`** — nothing reaches AA at reading size:

| Ink | Ratio | |
| --- | ---: | --- |
| `#FFFFFF` | 4.13 | **large text only** |
| `#FFFAE0` beige | 3.93 | **large text only** |
| `#1C3D34` deep ink | 2.88 | ✗ never |
| `#CCBDAA` | 2.25 | ✗ never — it is a deep-ground token |

> **The bar rule.** Every string on a bar is `--text-bar` (19px) at weight 700 —
> the first whole pixel above WCAG's 18.66px "large text" threshold. A 15px nav
> label on the bar looks fine and is an AA failure. This is why the header type
> is chunkier than it looks like it needs to be.

**On the beige sheet `#FFFAE0`** and on white cards:

| Ink | on beige | on white |
| --- | ---: | ---: |
| `#1C3D34` | 11.32 | 11.89 |
| `#613F37` brown deep | 8.79 | — |
| `#91253D` berry deep | 7.89 | — |
| `#2E55A3` blue deep | 6.79 | — |
| `#4A5D56` ink muted | 6.69 | — |
| `#E0782D` orange deep | **2.90 ✗** | — |

**Other pairings in use:** deep ink on the ribbon gold `#D4AF37` is **5.65**;
deep ink on blue-light **8.15** and on orange-light **7.27**.

---

## 4. Rules that are not taste

- **Two grounds.** The page is deep green and body copy sits straight on it.
  The bars are mid green and everything on them is large text.
- **The CTA inverts by surface.** On the ground: beige fill, deep ink. On a
  beige panel: deep-green fill, beige ink. A deep-green button on the ground
  measures 2.88 and fails 1.4.11 as a control you must be able to find.
- **Focus rings**: deep ink normally, **beige on the ground** — deep ink there
  is an indicator you cannot see.
- **Colour is never the only signal.** The current nav item carries an
  underline *and* `aria-current`; the current language carries a tick *and*
  `aria-checked`.
- **AA is the floor, and contrast is calculated.** State the ratio next to the
  token. `python3 scripts/contrast.py '#RRGGBB' '#RRGGBB'`.
- **44px minimum touch target**; motion is one subtle tier and
  `prefers-reduced-motion` zeroes delays as well as durations.
- Mobile-first, designed at 360px.

---

## 5. Brand assets

- **Wordmark**: lowercase "evermore" with a **reversed final `e`** — the mark's
  one distinctive move. It is not a typo and must never be corrected. Both
  `r`s are normal orientation. Never recolour, restretch or redraw it.
  Generated in both cuts by `scripts/mklogo.py` from
  `docs/design_guideline/logo.png`.
- **Favicon** is the wordmark's leading `e` on the masthead fill — the full
  lockup is 104:11 and turns to mush at 16px.
- **Background**: Steven's tile, extracted to beige-on-transparent by
  `scripts/mkpattern`, painted at 36% of native over the ground colour. Peak
  14.9% alpha, which leaves beige text at 7.29 where a glyph crosses a line.
- **Certification badges** are the issuer's own files when supplied — never
  redrawn, never sourced from image search. Defaults are Evermore's own seals
  from `scripts/mkseals.py`.
