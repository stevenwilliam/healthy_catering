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


def encode(path, w, h, rgba):
    def chunk(typ, body):
        return (struct.pack('>I', len(body)) + typ + body
                + struct.pack('>I', zlib.crc32(typ + body) & 0xFFFFFFFF))

    raw = bytearray()
    for y in range(h):
        raw.append(0)                       # filter 0; the art compresses fine
        raw += rgba[y * w * 4:(y + 1) * w * 4]
    png = (b'\x89PNG\r\n\x1a\n'
           + chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 6, 0, 0, 0))
           + chunk(b'IDAT', zlib.compress(bytes(raw), 9))
           + chunk(b'IEND', b''))
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


if __name__ == "__main__":
    main()
