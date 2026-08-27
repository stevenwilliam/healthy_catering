#!/usr/bin/env python3
"""Generate the two drifting produce layers for the public background.

Evermore sells healthy catering, so the page's texture is produce: leaves,
citrus, avocado, broccoli, carrot, wheat, berries — drawn as LINEWORK, not
silhouettes, because a filled shape at a usable alpha reads as a smudge and an
outline reads as a drawing.

Two layers at different scales and speeds give depth without either one having
to be strong enough to notice on its own.

    python3 scripts/mkfoodpattern.py

Writes web/public/images/produce-far.svg and produce-near.svg.

THE ALPHA IS NOT SET HERE. These are drawn at full strength in the brand beige;
public.css applies the opacity, because the budget is a CONTRAST decision that
belongs beside the tokens it constrains, not inside an art asset. See the
measured table in public.css — the binding constraint is --beige-deep, which
needs 4.50 and which the tile this replaces had already pushed to 4.16.

Deterministic: a fixed seed, so re-running produces byte-identical files and a
diff means somebody changed the drawing.
"""
import random
from pathlib import Path

INK = "#FFFAE0"
OUT = Path(__file__).resolve().parent.parent / "web" / "public" / "images"


def leaf(s):
    return (f'<path d="M0,{-s} C{s*0.72},{-s*0.45} {s*0.72},{s*0.45} 0,{s} '
            f'C{-s*0.72},{s*0.45} {-s*0.72},{-s*0.45} 0,{-s} Z"/>'
            f'<path d="M0,{-s} L0,{s}"/>')


def citrus(s):
    seg = "".join(f'<path d="M0,0 L{s*0.78*__import__("math").cos(a):.1f},'
                  f'{s*0.78*__import__("math").sin(a):.1f}"/>'
                  for a in [i * 3.14159 / 3 for i in range(6)])
    return f'<circle cx="0" cy="0" r="{s}"/><circle cx="0" cy="0" r="{s*0.82:.1f}"/>{seg}'


def avocado(s):
    return (f'<path d="M0,{-s} C{s*0.62},{-s*0.5} {s*0.7},{s*0.55} 0,{s} '
            f'C{-s*0.7},{s*0.55} {-s*0.62},{-s*0.5} 0,{-s} Z"/>'
            f'<circle cx="0" cy="{s*0.28:.1f}" r="{s*0.3:.1f}"/>')


def broccoli(s):
    return (f'<circle cx="{-s*0.42:.1f}" cy="{-s*0.3:.1f}" r="{s*0.36:.1f}"/>'
            f'<circle cx="{s*0.42:.1f}" cy="{-s*0.3:.1f}" r="{s*0.36:.1f}"/>'
            f'<circle cx="0" cy="{-s*0.62:.1f}" r="{s*0.38:.1f}"/>'
            f'<path d="M0,{s*0.05:.1f} L0,{s:.1f}"/>')


def carrot(s):
    return (f'<path d="M{-s*0.42:.1f},{-s*0.5:.1f} L{s*0.42:.1f},{-s*0.5:.1f} L0,{s} Z"/>'
            f'<path d="M0,{-s*0.5:.1f} L0,{-s:.1f}"/>'
            f'<path d="M0,{-s*0.62:.1f} L{-s*0.45:.1f},{-s:.1f}"/>'
            f'<path d="M0,{-s*0.62:.1f} L{s*0.45:.1f},{-s:.1f}"/>')


def wheat(s):
    g = "".join(f'<ellipse cx="{d*s*0.26:.1f}" cy="{-s*0.55 + i*s*0.34:.1f}" '
                f'rx="{s*0.17:.1f}" ry="{s*0.26:.1f}"/>'
                for i in range(3) for d in (-1, 1))
    return f'<path d="M0,{-s} L0,{s}"/>{g}'


def berries(s):
    return (f'<circle cx="{-s*0.38:.1f}" cy="{s*0.22:.1f}" r="{s*0.38:.1f}"/>'
            f'<circle cx="{s*0.38:.1f}" cy="{s*0.22:.1f}" r="{s*0.38:.1f}"/>'
            f'<circle cx="0" cy="{-s*0.4:.1f}" r="{s*0.38:.1f}"/>')


GLYPHS = [leaf, citrus, avocado, broccoli, carrot, wheat, berries]


def tile(size, count, glyph_r, stroke, seed):
    """One seamless tile.

    Every glyph is drawn NINE times — at its position and shifted by ±one tile
    on each axis. That is what makes the repeat seamless: a shape near an edge
    appears on the opposite edge automatically, so nothing is clipped at a seam
    and the glyphs do not have to be kept away from the edges (which is what
    makes a scatter look like a grid).
    """
    rnd = random.Random(seed)
    body = []
    for _ in range(count):
        fn = rnd.choice(GLYPHS)
        cx, cy = rnd.uniform(0, size), rnd.uniform(0, size)
        rot = rnd.uniform(0, 360)
        s = glyph_r * rnd.uniform(0.78, 1.22)
        shape = fn(s)
        for dx in (-size, 0, size):
            for dy in (-size, 0, size):
                body.append(
                    f'<g transform="translate({cx+dx:.1f},{cy+dy:.1f}) rotate({rot:.1f})">{shape}</g>')
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" '
            f'viewBox="0 0 {size} {size}">'
            f'<g fill="none" stroke="{INK}" stroke-width="{stroke}" '
            f'stroke-linecap="round" stroke-linejoin="round">'
            + "".join(body) + '</g></svg>')


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    # Coverage, not opacity, is the lever for visibility. The per-pixel peak is
    # what costs contrast, and a thicker stroke does not raise it — it only
    # puts more of the drawing on screen. So the strokes are heavy and the
    # glyphs large, and the alpha in public.css stays at the bound.
    specs = [("produce-far.svg", 460, 9, 40.0, 2.6, 20260827),
             ("produce-near.svg", 300, 6, 25.0, 2.0, 19730411)]
    for name, size, count, r, stroke, seed in specs:
        p = OUT / name
        p.write_text(tile(size, count, r, stroke, seed), encoding="utf-8")
        print(f"{p}  {p.stat().st_size} bytes")


if __name__ == "__main__":
    main()
