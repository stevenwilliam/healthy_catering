// Package money is integer rupiah arithmetic.
//
// IDR is stored as BIGINT whole rupiah: sen is obsolete in retail, so the
// rupiah is the minor unit here (CLAUDE.md §10). Floating point is prohibited
// on any path that touches money (CLAUDE.md §4) — there is no float64 in this
// package and none may be introduced.
//
// Two operations live here because both are easy to get subtly wrong and both
// are used from more than one place:
//
//   - Percent: basis points, rounded half-up.
//   - SplitInclusive: the tax back-calculation for D-30.
package money

import (
	"errors"
	"fmt"
	"strings"
)

// IDR is a whole-rupiah amount.
type IDR int64

// MaxIDR bounds any single amount. int64 could hold far more, but a price of a
// quadrillion rupiah is a typo, and catching it here means the multiplication
// in a line total cannot overflow silently.
const MaxIDR IDR = 1_000_000_000_000 // Rp 1 trillion

// ErrNegative is returned when an operation would produce a negative amount.
var ErrNegative = errors.New("money: negative amount")

// ErrOverflow is returned when an amount exceeds MaxIDR.
var ErrOverflow = errors.New("money: amount out of range")

// Validate reports whether an amount is a legal stored value.
func Validate(a IDR) error {
	if a < 0 {
		return fmt.Errorf("%w: %d", ErrNegative, a)
	}
	if a > MaxIDR {
		return fmt.Errorf("%w: %d", ErrOverflow, a)
	}
	return nil
}

// Multiply returns amount × qty, refusing to overflow rather than wrapping.
func Multiply(amount IDR, qty int) (IDR, error) {
	if qty < 0 {
		return 0, fmt.Errorf("%w: qty %d", ErrNegative, qty)
	}
	if err := Validate(amount); err != nil {
		return 0, err
	}
	if qty != 0 && int64(amount) > int64(MaxIDR)/int64(qty) {
		return 0, fmt.Errorf("%w: %d × %d", ErrOverflow, amount, qty)
	}
	return amount * IDR(qty), nil
}

// Percent applies a basis-point rate, rounded half-up:
//
//	floor((amount × bps + 5000) / 10000)
//
// This is the house rounding rule (CLAUDE.md §4) and the only place it is
// implemented.
func Percent(amount IDR, bps int) (IDR, error) {
	if bps < 0 {
		return 0, fmt.Errorf("money: negative bps %d", bps)
	}
	if err := Validate(amount); err != nil {
		return 0, err
	}
	return IDR((int64(amount)*int64(bps) + 5000) / 10000), nil
}

// Split is a tax-inclusive amount broken into its base and its tax.
//
// Base + Tax always equals Gross exactly, because Tax is taken as the residue
// rather than computed separately. Rounding the two independently is how an
// invoice ends up disagreeing with its own total by one rupiah.
type Split struct {
	Gross   IDR // what the customer pays — the number staff typed
	Base    IDR // the tax base (DPP)
	Tax     IDR // the tax portion (PPN)
	RateBps int // the rate this split was computed at, for the snapshot
}

// SplitInclusive back-calculates the base and tax from a tax-INCLUSIVE amount
// (docs/02 D-30). With D = 10000 + bps:
//
//	base = (gross × 10000 + D/2) / D     -- integer division, half-up
//	tax  = gross - base                  -- the residue
//
// Worked: Rp 500.000 at 1100 bps → base 450.450, tax 49.550, and the two sum
// back to 500.000 exactly.
//
// Callers pass the LINE TOTAL, not the unit price: splitting per unit and
// multiplying multiplies the rounding error by the quantity.
func SplitInclusive(gross IDR, bps int) (Split, error) {
	if bps < 0 || bps > 10000 {
		return Split{}, fmt.Errorf("money: tax rate %d bps out of range", bps)
	}
	if err := Validate(gross); err != nil {
		return Split{}, err
	}
	if bps == 0 {
		return Split{Gross: gross, Base: gross, Tax: 0, RateBps: 0}, nil
	}
	d := int64(10000 + bps)
	base := IDR((int64(gross)*10000 + d/2) / d)
	return Split{Gross: gross, Base: base, Tax: gross - base, RateBps: bps}, nil
}

// SumSplits adds line splits into an order-level split.
//
// The order's tax is the SUM of its lines' taxes and is never re-derived from
// the order total — re-deriving reintroduces a rounding difference between an
// invoice and the lines printed on it (docs/02 D-30).
func SumSplits(splits []Split, bps int) Split {
	out := Split{RateBps: bps}
	for _, s := range splits {
		out.Gross += s.Gross
		out.Base += s.Base
		out.Tax += s.Tax
	}
	return out
}

// Format renders an amount the Indonesian way: Rp 500.000, dot-grouped, no
// decimals (PROMPT §2). Negative amounts render with the sign before "Rp".
func Format(a IDR) string {
	neg := a < 0
	if neg {
		a = -a
	}
	digits := fmt.Sprintf("%d", int64(a))

	var sb strings.Builder
	if neg {
		sb.WriteByte('-')
	}
	sb.WriteString("Rp ")
	lead := len(digits) % 3
	if lead == 0 {
		lead = 3
	}
	sb.WriteString(digits[:lead])
	for i := lead; i < len(digits); i += 3 {
		sb.WriteByte('.')
		sb.WriteString(digits[i : i+3])
	}
	return sb.String()
}
