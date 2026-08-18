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
CANVAS = '#468973'   # page ground (Nourish Green)
DEEPGREEN = '#1C3D34'  # masthead + footer fill, and the ink everywhere else
SHEET = '#FFFAE0'    # beige content sheet
WHITE = '#FFFFFF'
MUTED = '#4A5D56'
BEIGE_DEEP = '#CCBDAA'
WA_TEAL = '#128C7E'  # WhatsApp's darker brand teal


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
    ('body copy on the green ground — beige', SHEET, CANVAS),
    ('body copy on the green ground — white', WHITE, CANVAS),
    ('body copy on the green ground — deep ink', DEEPGREEN, CANVAS),
    ('beige sheet as a surface on the ground', SHEET, CANVAS),
    ('white card as a surface on the ground', WHITE, CANVAS),
    ('deep ink on the beige sheet', DEEPGREEN, SHEET),
    ('muted ink on the beige sheet', MUTED, SHEET),
    ('deep ink on a white card', DEEPGREEN, WHITE),
    ('beige on the masthead/footer fill', SHEET, DEEPGREEN),
    ('beige-deep on the masthead/footer fill', BEIGE_DEEP, DEEPGREEN),
    ('masthead fill against the ground', DEEPGREEN, CANVAS),
    ('WhatsApp teal against the ground', WA_TEAL, CANVAS),
    ('white glyph on WhatsApp teal', WHITE, WA_TEAL),
    ('beige ring against WhatsApp teal', SHEET, WA_TEAL),
    ('beige ring against the ground', SHEET, CANVAS),
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
