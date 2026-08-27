#!/usr/bin/env python3
"""Derive web logo assets from the supplied brand artwork.

No ImageMagick or Pillow on this box, so PNG is read and written directly:
8-bit RGBA, non-interlaced, which is what docs/design_guideline/logo.png is.

The artwork is flat single-colour lettering, so it is reduced to a COVERAGE
mask (how much ink is at this pixel) and then re-emitted in whatever colour the
surface needs. That is what lets one source file produce both the reversed-out
wordmark for the dark-green masthead and the deep-green one for light fills,
without ever recolouring by hand.
"""
import struct
import sys
import zlib

SRC = 'docs/design_guideline/logo.png'


def decode(path):
    d = open(path, 'rb').read()
    assert d[:8] == b'\x89PNG\r\n\x1a\n', 'not a PNG'
    idat, i = [], 8
    w = h = bd = ct = None
    while i < len(d):
        (ln,), typ = struct.unpack('>I', d[i:i + 4]), d[i + 4:i + 8]
        body = d[i + 8:i + 8 + ln]
        if typ == b'IHDR':
            w, h, bd, ct, _, _, il = struct.unpack('>IIBBBBB', body)
            assert (bd, ct, il) == (8, 6, 0), f'unsupported PNG {bd=} {ct=} {il=}'
        elif typ == b'IDAT':
            idat.append(body)
        elif typ == b'IEND':
            break
        i += 12 + ln
    raw = zlib.decompress(b''.join(idat))

    # Undo the per-scanline filters. bpp is 4 for RGBA8.
    bpp, stride = 4, w * 4
    out = bytearray(h * stride)
    prev = bytearray(stride)
    pos = 0
    for y in range(h):
        ft = raw[pos]
        pos += 1
        line = bytearray(raw[pos:pos + stride])
        pos += stride
        if ft == 1:
            for x in range(bpp, stride):
                line[x] = (line[x] + line[x - bpp]) & 0xFF
        elif ft == 2:
            for x in range(stride):
                line[x] = (line[x] + prev[x]) & 0xFF
        elif ft == 3:
            for x in range(stride):
                a = line[x - bpp] if x >= bpp else 0
                line[x] = (line[x] + ((a + prev[x]) >> 1)) & 0xFF
        elif ft == 4:
            for x in range(stride):
                a = line[x - bpp] if x >= bpp else 0
                b = prev[x]
                c = prev[x - bpp] if x >= bpp else 0
                p = a + b - c
                pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
                pr = a if (pa <= pb and pa <= pc) else (b if pb <= pc else c)
                line[x] = (line[x] + pr) & 0xFF
        elif ft != 0:
            raise ValueError(f'bad filter {ft}')
        out[y * stride:(y + 1) * stride] = line
        prev = line
    return w, h, out


def coverage(w, h, px):
    """Ink coverage 0..255 per pixel.

    Handles both ways this artwork can arrive: transparent background (use the
    alpha channel) or opaque white background (use darkness). Multiplying the
    two is correct for either, because the absent one is a constant 255.

    Then NORMALISE against the darkest pixel actually present. The artwork is
    drawn in Nourish Green #1C3D34, whose luma is 50, so raw darkness caps at
    205 and a solid stroke would be emitted at 80% alpha — the recoloured
    wordmark would render as a washed-out tint of whatever it sits on, quietly
    losing the 11.32:1 the design system claims for it. Scaling so the ink
    colour maps to a full 255 keeps the anti-aliased edges linear and makes the
    solid interior genuinely solid.
    """
    raw = bytearray(w * h)
    for i in range(w * h):
        o = i * 4
        r, g, b, a = px[o], px[o + 1], px[o + 2], px[o + 3]
        # Rec.601 luma is plenty for deciding ink-vs-paper.
        lum = (r * 299 + g * 587 + b * 114) // 1000
        raw[i] = (a * (255 - lum)) // 255
    peak = max(raw)
    print(f'peak raw coverage {peak} -> normalised to 255', file=sys.stderr)
    if peak == 0:
        return raw
    return bytearray(min(255, v * 255 // peak) for v in raw)


def bbox(w, h, cov, thresh=10):
    x0, y0, x1, y1 = w, h, -1, -1
    for y in range(h):
        row = cov[y * w:(y + 1) * w]
        if max(row) < thresh:
            continue
        if y < y0:
            y0 = y
        y1 = y
        for x in range(w):
            if row[x] >= thresh:
                if x < x0:
                    x0 = x
                if x > x1:
                    x1 = x
    return x0, y0, x1 + 1, y1 + 1


def downscale(cov, w, box, out_w):
    """Area-average the coverage mask down to out_w, preserving aspect."""
    x0, y0, x1, y1 = box
    src_w, src_h = x1 - x0, y1 - y0
    out_h = max(1, round(src_h * out_w / src_w))
    acc = [0] * (out_w * out_h)
    cnt = [0] * (out_w * out_h)
    for y in range(src_h):
        oy = min(out_h - 1, y * out_h // src_h)
        base = (y0 + y) * w + x0
        obase = oy * out_w
        for x in range(src_w):
            ox = min(out_w - 1, x * out_w // src_w)
            acc[obase + ox] += cov[base + x]
            cnt[obase + ox] += 1
    mask = bytearray(out_w * out_h)
    for i in range(out_w * out_h):
        if cnt[i]:
            mask[i] = acc[i] // cnt[i]
    return out_w, out_h, mask


def encode_bytes(w, h, rgba):
    def chunk(typ, body):
        return (struct.pack('>I', len(body)) + typ + body
                + struct.pack('>I', zlib.crc32(typ + body) & 0xFFFFFFFF))

    raw = bytearray()
    for y in range(h):
        raw.append(0)                       # filter 0; the art compresses fine
        raw += rgba[y * w * 4:(y + 1) * w * 4]
    return (b'\x89PNG\r\n\x1a\n'
            + chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 6, 0, 0, 0))
            + chunk(b'IDAT', zlib.compress(bytes(raw), 9))
            + chunk(b'IEND', b''))


def encode(path, w, h, rgba):
    png = encode_bytes(w, h, rgba)
    open(path, 'wb').write(png)
    return len(png)


def tint(mask, w, h, rgb):
    r, g, b = rgb
    out = bytearray(w * h * 4)
    for i in range(w * h):
        o = i * 4
        out[o], out[o + 1], out[o + 2], out[o + 3] = r, g, b, mask[i]
    return out


def hexrgb(s):
    s = s.lstrip('#')
    return int(s[0:2], 16), int(s[2:4], 16), int(s[4:6], 16)


def first_glyph_box(cov, w, box, thresh=12):
    """Bounding box of the leading 'e'.

    The brand supplies a horizontal lockup and no icon mark, and a 6238x663
    wordmark shrunk into a 16px tab is an illegible smear. The leading 'e' is
    the distinctive letterform in this face — the flat crossbar — so it stands
    in as the mark. Found by ink columns rather than hard-coded pixels, so it
    survives the artwork being re-exported.
    """
    x0, y0, x1, y1 = box
    gx0 = gx1 = None
    for x in range(x0, x1):
        inked = any(cov[y * w + x] > thresh for y in range(y0, y1))
        if inked and gx0 is None:
            gx0 = x
        elif not inked and gx0 is not None:
            gx1 = x
            break
    if gx1 is None:
        gx1 = x1
    # Tighten vertically to the glyph itself, not the whole wordmark band.
    gy0, gy1 = None, None
    for y in range(y0, y1):
        if any(cov[y * w + x] > thresh for x in range(gx0, gx1)):
            if gy0 is None:
                gy0 = y
            gy1 = y + 1
    return gx0, gy0, gx1, gy1


def square_icon(cov, w, gbox, size, bg, fg, pad_frac=0.125):
    """The glyph, centred on an opaque square.

    Opaque rather than transparent on purpose: a tab strip may be light or
    dark, and a transparent beige mark disappears into a light one. The deep
    green field carries the mark at 11.32:1 either way, and is the same fill as
    the masthead, so a tab reads as the same brand as the page.
    """
    gx0, gy0, gx1, gy1 = gbox
    gw, gh = gx1 - gx0, gy1 - gy0
    inner = max(1, int(round(size * (1 - 2 * pad_frac))))
    out_w = inner if gw >= gh else max(1, int(round(inner * gw / gh)))
    mw, mh, mask = downscale(cov, w, gbox, out_w)

    ox, oy = (size - mw) // 2, (size - mh) // 2
    canvas = bytearray()
    for _ in range(size * size):
        canvas += bytes((bg[0], bg[1], bg[2], 255))
    for y in range(mh):
        for x in range(mw):
            a = mask[y * mw + x]
            if not a:
                continue
            o = ((oy + y) * size + (ox + x)) * 4
            for c in range(3):
                canvas[o + c] = (fg[c] * a + bg[c] * (255 - a)) // 255
    return bytes(canvas)


def write_ico(path, images):
    """A .ico holding PNG-encoded frames.

    PNG inside ICO is the modern form and is what every current browser reads;
    the legacy BMP-with-AND-mask form buys nothing here. `images` is a list of
    (size, png_bytes).
    """
    count = len(images)
    header = struct.pack('<HHH', 0, 1, count)
    offset = 6 + 16 * count
    entries, blobs = b'', b''
    for size, blob in images:
        entries += struct.pack(
            '<BBBBHHII',
            0 if size >= 256 else size,   # 0 means 256 in the ICO header
            0 if size >= 256 else size,
            0, 0, 1, 32, len(blob), offset)
        offset += len(blob)
        blobs += blob
    open(path, 'wb').write(header + entries + blobs)
    return 6 + 16 * count + len(blobs)


def main():
    w, h, px = decode(SRC)
    print(f'source {w}x{h}', file=sys.stderr)
    cov = coverage(w, h, px)
    box = bbox(w, h, cov)
    print(f'trimmed to {box} = {box[2]-box[0]}x{box[3]-box[1]}', file=sys.stderr)

    ow, oh, mask = downscale(cov, w, box, 560)
    print(f'wordmark {ow}x{oh}', file=sys.stderr)

    DEEP = hexrgb('#1C3D34')   # on light surfaces
    BEIGE = hexrgb('#FFFAE0')  # reversed out, on the green masthead/footer

    n = encode('web/public/images/evermore-wordmark-deep.png', ow, oh,
               tint(mask, ow, oh, DEEP))
    print(f'  deep  {n} bytes', file=sys.stderr)
    n = encode('web/public/images/evermore-wordmark-light.png', ow, oh,
               tint(mask, ow, oh, BEIGE))
    print(f'  light {n} bytes', file=sys.stderr)

    # Open Graph card. The template already points at /images/og-default.png
    # and nothing was ever generated there, so a share preview shows a broken
    # image. 1200x630 is the size every preview bot crops to.
    OW, OH = 1200, 630
    GREEN = hexrgb('#1C3D34')
    card = bytearray()
    for _ in range(OW * OH):
        card += bytes((GREEN[0], GREEN[1], GREEN[2], 255))
    lw, lh, lmask = downscale(cov, w, box, 760)
    lx, ly = (OW - lw) // 2, (OH - lh) // 2
    for y in range(lh):
        for x in range(lw):
            a = lmask[y * lw + x]
            if not a:
                continue
            o = ((ly + y) * OW + (lx + x)) * 4
            # Composite beige ink over the green field.
            for c in range(3):
                card[o + c] = (BEIGE[c] * a + GREEN[c] * (255 - a)) // 255
    n = encode('web/public/images/og-default.png', OW, OH, bytes(card))
    print(f'  og    {OW}x{OH} {n} bytes', file=sys.stderr)

    # ── Favicons ────────────────────────────────────────────────────────────
    # Square mark, derived from the leading 'e' — the wordmark itself is 104:11
    # and turns into an unreadable smear at 16px.
    gbox = first_glyph_box(cov, w, box)
    print(f'glyph "e" at {gbox} = {gbox[2]-gbox[0]}x{gbox[3]-gbox[1]}', file=sys.stderr)

    ico_frames = []
    for size in (16, 32, 48):
        blob = encode_bytes(size, size,
                            square_icon(cov, w, gbox, size, DEEP, BEIGE))
        ico_frames.append((size, blob))
    n = write_ico('web/public/images/favicon.ico', ico_frames)
    print(f'  favicon.ico 16/32/48 {n} bytes', file=sys.stderr)

    for size, name in ((32, 'favicon-32.png'), (180, 'apple-touch-icon.png')):
        n = encode(f'web/public/images/{name}', size, size,
                   square_icon(cov, w, gbox, size, DEEP, BEIGE))
        print(f'  {name} {size}x{size} {n} bytes', file=sys.stderr)

    # ── Masthead mark ───────────────────────────────────────────────────────
    # The same 'e', INVERTED: a beige field carrying deep-green ink, because
    # this one sits on the mid-green bar rather than on a browser tab strip.
    # Measured, that is the only way round it works: beige on #468973 is 3.93,
    # which clears 1.4.11's 3:1 for a graphic's edge, and deep ink on the beige
    # field is 11.32. A deep-green field on the bar would be 2.88 and the badge
    # would have no findable edge at all.
    #
    # 128px for a mark drawn at roughly 28 CSS px, so it still resolves on a 3x
    # phone. Nothing here redraws the letterform: it is the wordmark's own
    # leading glyph, masked and re-inked, which is the same operation the
    # favicon has always used (design.md §5 — never redraw the mark).
    n = encode('web/public/images/evermore-mark-128.png', 128, 128,
               square_icon(cov, w, gbox, 128, BEIGE, DEEP))
    print(f'  evermore-mark-128.png 128x128 {n} bytes', file=sys.stderr)


if __name__ == "__main__":
    main()
