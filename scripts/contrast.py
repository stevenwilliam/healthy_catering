#!/usr/bin/env python3
"""WCAG contrast for the Evermore palette.

CLAUDE.md §7: contrast is calculated, not eyeballed. Run this after any colour
change and paste the numbers into web/public/css/tokens.css and
docs/10-design-system.md — the comment beside a token IS the record.

    python3 scripts/contrast.py            # the standing pairings
    python3 scripts/contrast.py '#FFFAE0' '#468973'
"""
import sys

# The ground, the surfaces, and the inks. Keep in step with tokens.css.
#
# 2026-08-18: Steven SWAPPED the two greens. The GROUND is now deep #1C3D34 and
# the BARS (masthead + footer) are mid #468973. The names below were left
# describing the old arrangement for nine days, which is how design.md came to
# say "a deep-green button on the ground measures 2.88" — a pairing that no
# longer exists. The ratios were never wrong; the labels were.
GROUND = '#1C3D34'   # THE PAGE GROUND — deep green
BAR = '#468973'      # masthead + footer fill — mid green
SHEET = '#FFFAE0'    # beige content sheet
WHITE = '#FFFFFF'
MUTED = '#4A5D56'
BEIGE_DEEP = '#CCBDAA'
WA_TEAL = '#128C7E'  # WhatsApp's darker brand teal
GOLD = '#D4AF37'     # the corner ribbon's base
GOLD_HI = '#FFF3C4'  # the ribbon shimmer, mid sweep
GOLD_PEAK = '#FFFDF0'  # the ribbon shimmer, peak
BORDER = '#778979'   # --border rgba(28,61,52,.60) composited over the sheet
BORDER_SUBTLE = '#BFC5B0'  # --border-subtle rgba(28,61,52,.28) over the sheet


def _lin(c):
    c /= 255
    return c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4


def luminance(hexcolour):
    h = hexcolour.lstrip('#')
    r, g, b = (int(h[i:i + 2], 16) for i in (0, 2, 4))
    return 0.2126 * _lin(r) + 0.7152 * _lin(g) + 0.0722 * _lin(b)


def ratio(a, b):
    la, lb = luminance(a), luminance(b)
    hi, lo = max(la, lb), min(la, lb)
    return (hi + 0.05) / (lo + 0.05)


def verdict(v):
    if v >= 7:      return 'AAA body'
    if v >= 4.5:    return 'AA body'
    if v >= 3:      return 'AA large / UI boundary'
    return 'FAILS'


PAIRS = [
    # ── On the deep ground, where body copy may sit directly ──────────────
    ('beige on the deep ground', SHEET, GROUND),
    ('white on the deep ground', WHITE, GROUND),
    ('beige-deep (muted) on the deep ground', BEIGE_DEEP, GROUND),
    ('the beige sheet as a surface on the ground', SHEET, GROUND),
    ('the white card as a surface on the ground', WHITE, GROUND),

    # ── On the BARS. Nothing here reaches AA at reading size, which is what
    #    --text-bar (19px/700) exists to work around. ───────────────────────
    ('beige on a bar  [LARGE TEXT ONLY]', SHEET, BAR),
    ('white on a bar  [LARGE TEXT ONLY]', WHITE, BAR),
    ('deep ink on a bar  [never]', GROUND, BAR),
    ('beige-deep on a bar  [never — deep-ground token]', BEIGE_DEEP, BAR),
    ('a bar against the ground (the seam)', BAR, GROUND),

    # ── On the beige sheet and white cards ────────────────────────────────
    ('deep ink on the beige sheet', GROUND, SHEET),
    ('muted ink on the beige sheet', MUTED, SHEET),
    ('deep ink on a white card', GROUND, WHITE),

    # ── Non-text contrast, WCAG 1.4.11 — the 3:1 boundary ─────────────────
    ('--border on the sheet  [control boundary]', BORDER, SHEET),
    ('--border-subtle on the sheet  [DECORATIVE ONLY]', BORDER_SUBTLE, SHEET),
    ('beige-deep on the sheet  [never a border]', BEIGE_DEEP, SHEET),

    # ── The corner ribbon. The shimmer only ever LIFTS the ratio, so the
    #    base gold is the worst case and the one that governs. ─────────────
    ('ribbon ink on the base gold  [worst case]', GROUND, GOLD),
    ('ribbon ink on the shimmer, mid', GROUND, GOLD_HI),
    ('ribbon ink on the shimmer, peak', GROUND, GOLD_PEAK),

    # ── The WhatsApp float. Its fill is 1.00 against the ground by design;
    #    the BEIGE RING is the affordance, and it is what must clear 3:1. ──
    ('WhatsApp teal against the ground  [the ring carries it]', WA_TEAL, GROUND),
    ('white glyph on WhatsApp teal', WHITE, WA_TEAL),
    ('the beige ring against WhatsApp teal', SHEET, WA_TEAL),
    ('the beige ring against the ground', SHEET, GROUND),
]


if len(sys.argv) == 3:
    a, b = sys.argv[1], sys.argv[2]
    v = ratio(a, b)
    print(f'{a} on {b}  {v:.2f}  {verdict(v)}')
else:
    width = max(len(n) for n, _, _ in PAIRS)
    for name, a, b in PAIRS:
        v = ratio(a, b)
        print(f'{name:<{width}}  {a} on {b}  {v:5.2f}  {verdict(v)}')
