// Command mkpattern turns a supplied pattern image into a transparent overlay.
//
//	go run ./scripts/mkpattern /home/dev/pattern.jpg web/public/images/pattern.png
//
// The problem it solves: the supplied file is a JPEG, so it has no alpha and
// carries its own background colour. Used directly as a CSS background it would
// COVER the page's ground colour rather than overlay it — the opposite of what
// was asked for. So the artwork is separated from its background and re-emitted
// as beige-on-transparent, letting the deep green show through underneath.
//
// Two things it works out rather than assumes:
//
//   - The background colour, taken as the most common colour in the image. The
//     linework is a small minority of pixels, so the mode is the background by
//     definition; hard-coding a guess would break on the next file.
//   - The tile period, by autocorrelation. A crop of a repeating pattern is
//     almost never a whole number of repeats — this one is 959px wide with a
//     period near 150 — so tiling the file as-is shows a seam every screen
//     width. Cropping to exactly one period makes it seamless.
package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"sort"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mkpattern <in.jpg> <out.png>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "mkpattern:", err)
		os.Exit(1)
	}
}

// The ink the overlay is drawn in, and how strong the strongest line may be.
// 0.16 rather than the 0.05 the hand-drawn motif used: these lines are one
// pixel of a thin outline, so they need more alpha to read at all, and they
// cover far less of the surface.
const (
	inkR, inkG, inkB = 0xFF, 0xFA, 0xE0 // Restore Beige
	maxAlpha         = 0.16
)

func run(inPath, outPath string) error {
	f, err := os.Open(inPath)
	if err != nil {
		return err
	}
	src, format, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("decoding %s: %w", inPath, err)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	fmt.Fprintf(os.Stderr, "read %s %dx%d (%s)\n", inPath, w, h, format)

	// Luma per pixel, once.
	luma := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bb, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			luma[y*w+x] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bb>>8)
		}
	}

	base := modalLuma(luma)
	fmt.Fprintf(os.Stderr, "background luma ≈ %.1f\n", base)

	if os.Getenv("PATTERN_REPORT") != "" {
		axisReport(luma, w, h, true, "horizontal")
		axisReport(luma, w, h, false, "vertical")
	}
	px, py := period(luma, w, h)
	// Overrides, for measuring candidate tiles against the wrap metric.
	if v := os.Getenv("PATTERN_PX"); v != "" {
		fmt.Sscanf(v, "%d", &px)
	}
	if v := os.Getenv("PATTERN_PY"); v != "" {
		fmt.Sscanf(v, "%d", &py)
	}
	fmt.Fprintf(os.Stderr, "tile period %dx%d\n", px, py)

	// How far the brightest line sits from the background, so the scaling
	// uses the real dynamic range rather than a guess.
	var peak float64
	for _, l := range luma {
		if d := math.Abs(l - base); d > peak {
			peak = d
		}
	}
	if peak < 1 {
		return fmt.Errorf("image is flat — nothing to extract")
	}
	fmt.Fprintf(os.Stderr, "peak deviation %.1f\n", peak)

	// Alpha straight from the source, for any pixel including ones beyond the
	// crop — the feather below needs to read a period further on.
	// Sub-pixel period, then RESAMPLE.
	//
	// The best INTEGER period is not the true one: horizontally the repeat is
	// a clean 151, but vertically the residual bottoms out at 155 well above
	// the noise floor, which means the real period is fractional. Cropping at
	// 155 leaves the tile's last row not quite meeting its first, and that is
	// a hard line every 155px down the page.
	//
	// A cross-fade does NOT fix this — it was tried, and blending the tail
	// toward "one period further on" is a no-op when the period is exact and a
	// distortion when it is not. What fixes it is sampling exactly one period:
	// find the period to sub-pixel accuracy by fitting a parabola through the
	// autocorrelation minimum, then resample that many source pixels into an
	// integer-sized tile.
	fpx := refine(luma, w, h, true, px)
	fpy := refine(luma, w, h, false, py)
	fmt.Fprintf(os.Stderr, "refined period %.2fx%.2f\n", fpx, fpy)

	outW, outH := int(math.Round(fpx)), int(math.Round(fpy))
	out := image.NewNRGBA(image.Rect(0, 0, outW, outH))
	for y := 0; y < outH; y++ {
		sy := float64(y) * fpy / float64(outH)
		for x := 0; x < outW; x++ {
			sx := float64(x) * fpx / float64(outW)
			d := sampleAlpha(luma, w, h, sx, sy, base, peak)
			out.SetNRGBA(x, y, color.NRGBA{
				R: inkR, G: inkG, B: inkB,
				A: uint8(math.Round(d * maxAlpha * 255)),
			})
		}
	}
	px, py = outW, outH

	g, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = g.Close() }()
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(g, out); err != nil {
		return err
	}
	st, _ := os.Stat(outPath)
	fmt.Fprintf(os.Stderr, "wrote %s %dx%d, %d bytes\n", outPath, px, py, st.Size())
	return nil
}

// modalLuma returns the most common luma, bucketed — the background, since the
// linework is a small minority of the pixels.
func modalLuma(luma []float64) float64 {
	var hist [256]int
	for _, l := range luma {
		i := int(l)
		if i < 0 {
			i = 0
		}
		if i > 255 {
			i = 255
		}
		hist[i]++
	}
	best, bestN := 0, -1
	for i, n := range hist {
		if n > bestN {
			best, bestN = i, n
		}
	}
	return float64(best)
}

// period finds the smallest horizontal and vertical repeat by comparing the
// image with itself at increasing offsets and taking the best match.
//
// Without this the file tiles at its own 959x569 crop, which is not a whole
// number of repeats, so a visible seam lands every screen width.
func period(luma []float64, w, h int) (int, int) {
	return axisPeriod(luma, w, h, true), axisPeriod(luma, w, h, false)
}

// axisReport prints the best candidate periods and their residuals, so a bad
// match is visible as a number rather than as a seam noticed later on a page.
func axisReport(luma []float64, w, h int, horizontal bool, label string) {
	type cand struct {
		p     int
		score float64
	}
	size := w
	if !horizontal {
		size = h
	}
	var all []cand
	for p := 24; p <= size/2; p++ {
		if s, ok := axisScore(luma, w, h, horizontal, p); ok {
			all = append(all, cand{p, s})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].score < all[j].score })
	fmt.Fprintf(os.Stderr, "%s candidates (lower is more seamless):\n", label)
	for i := 0; i < len(all) && i < 8; i++ {
		fmt.Fprintf(os.Stderr, "   p=%-4d residual %.2f\n", all[i].p, all[i].score)
	}
}

func axisScore(luma []float64, w, h int, horizontal bool, p int) (float64, bool) {
	var sum float64
	var n int
	if horizontal {
		for y := 0; y < h; y += 3 {
			for x := 0; x+p < w; x += 3 {
				d := luma[y*w+x] - luma[y*w+x+p]
				sum += d * d
				n++
			}
		}
	} else {
		for y := 0; y+p < h; y += 3 {
			for x := 0; x < w; x += 3 {
				d := luma[y*w+x] - luma[(y+p)*w+x]
				sum += d * d
				n++
			}
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

func axisPeriod(luma []float64, w, h int, horizontal bool) int {
	size := w
	if !horizontal {
		size = h
	}
	// A period below 24px is almost certainly noise; above half the image
	// there is not enough overlap left to trust the comparison.
	best, bestScore := size, math.MaxFloat64
	for p := 24; p <= size/2; p++ {
		var sum float64
		var n int
		// Sample rather than compare every pixel: this runs over ~500k
		// candidates otherwise, and every 3rd pixel is ample to find a match.
		if horizontal {
			for y := 0; y < h; y += 3 {
				for x := 0; x+p < w; x += 3 {
					d := luma[y*w+x] - luma[y*w+x+p]
					sum += d * d
					n++
				}
			}
		} else {
			for y := 0; y+p < h; y += 3 {
				for x := 0; x < w; x += 3 {
					d := luma[y*w+x] - luma[(y+p)*w+x]
					sum += d * d
					n++
				}
			}
		}
		if n == 0 {
			continue
		}
		score := sum / float64(n)
		// Strictly better, so the SMALLEST period wins a tie — a pattern that
		// repeats every 150px also "repeats" every 300px, and the small tile
		// is the one worth shipping.
		if score < bestScore*0.995 {
			best, bestScore = p, score
		}
	}
	return best
}

// refine turns an integer period into a sub-pixel one by fitting a parabola
// through the autocorrelation minimum and its two neighbours. The vertex of
// that parabola is a better estimate than the sampled minimum, which is what
// lets the resample land on a whole period.
func refine(luma []float64, w, h int, horizontal bool, p int) float64 {
	prev, ok1 := axisScore(luma, w, h, horizontal, p-1)
	here, ok2 := axisScore(luma, w, h, horizontal, p)
	next, ok3 := axisScore(luma, w, h, horizontal, p+1)
	if !ok1 || !ok2 || !ok3 {
		return float64(p)
	}
	den := prev - 2*here + next
	if math.Abs(den) < 1e-9 {
		return float64(p)
	}
	delta := 0.5 * (prev - next) / den
	// A vertex further than half a sample away means the fit is not describing
	// a minimum; trust the integer instead of an extrapolation.
	if math.Abs(delta) > 0.5 {
		return float64(p)
	}
	return float64(p) + delta
}

// sampleAlpha reads the source bilinearly at a fractional position and returns
// ink coverage. Bilinear because the resample lands between pixels by design —
// nearest-neighbour would reintroduce the very misalignment being removed.
func sampleAlpha(luma []float64, w, h int, x, y, base, peak float64) float64 {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	at := func(xx, yy int) float64 {
		// Wrap rather than clamp: a sample past the edge belongs to the next
		// repeat, and clamping would smear the final row across the seam.
		return luma[((yy%h+h)%h)*w+((xx%w+w)%w)]
	}
	l := at(x0, y0)*(1-fx)*(1-fy) + at(x0+1, y0)*fx*(1-fy) +
		at(x0, y0+1)*(1-fx)*fy + at(x0+1, y0+1)*fx*fy

	d := math.Abs(l-base) / peak
	if d < 0.06 {
		return 0
	}
	return d
}
