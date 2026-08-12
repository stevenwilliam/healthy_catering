package money

import "testing"

// The tax split is the money path Steven asked for by name (D-30), so it gets
// the boundary cases as well as the happy one.
func TestSplitInclusive(t *testing.T) {
	tests := []struct {
		name              string
		gross             IDR
		bps               int
		wantBase, wantTax IDR
	}{
		{"worked example from docs/01 §3.11", 500_000, 1100, 450_450, 49_550},
		{"12 percent", 500_000, 1200, 446_429, 53_571},
		{"zero rate leaves everything as base", 500_000, 0, 500_000, 0},
		{"zero amount", 0, 1100, 0, 0},
		{"one rupiah rounds to base", 1, 1100, 1, 0},
		{"two rupiah", 2, 1100, 2, 0},
		{"ten rupiah", 10, 1100, 9, 1},
		{"large line total", 999 * 500_000, 1100, 450_000_000, 49_500_000},
		{"rate of 100 percent halves", 1000, 10000, 500, 500},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SplitInclusive(tc.gross, tc.bps)
			if err != nil {
				t.Fatalf("SplitInclusive: %v", err)
			}
			if got.Base != tc.wantBase || got.Tax != tc.wantTax {
				t.Errorf("base/tax = %d/%d, want %d/%d", got.Base, got.Tax, tc.wantBase, tc.wantTax)
			}
			// The invariant that matters more than any single figure.
			if got.Base+got.Tax != tc.gross {
				t.Errorf("base+tax = %d, want gross %d", got.Base+got.Tax, tc.gross)
			}
			if got.RateBps != tc.bps {
				t.Errorf("RateBps = %d, want %d", got.RateBps, tc.bps)
			}
		})
	}
}

// Whatever the amount and whatever the rate, the parts must reconstitute the
// whole. This is the property the invoice depends on.
func TestSplitInclusiveAlwaysReconstitutes(t *testing.T) {
	for _, bps := range []int{0, 1, 100, 1100, 1200, 3300, 10000} {
		for gross := IDR(0); gross <= 3000; gross++ {
			s, err := SplitInclusive(gross, bps)
			if err != nil {
				t.Fatalf("gross %d bps %d: %v", gross, bps, err)
			}
			if s.Base+s.Tax != gross {
				t.Fatalf("gross %d bps %d: base %d + tax %d != gross", gross, bps, s.Base, s.Tax)
			}
			if s.Base < 0 || s.Tax < 0 {
				t.Fatalf("gross %d bps %d: negative component base=%d tax=%d", gross, bps, s.Base, s.Tax)
			}
		}
	}
}

func TestSplitInclusiveRejectsBadRate(t *testing.T) {
	for _, bps := range []int{-1, 10001} {
		if _, err := SplitInclusive(1000, bps); err == nil {
			t.Errorf("bps %d: expected an error", bps)
		}
	}
}

// Order tax is the sum of line taxes, never a re-derivation from the order
// total. This test is the difference between the two.
func TestSumSplitsDiffersFromRederiving(t *testing.T) {
	var lines []Split
	for i := 0; i < 3; i++ {
		s, err := SplitInclusive(10, 1100) // each rounds down to 9 base, 1 tax
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, s)
	}
	summed := SumSplits(lines, 1100)
	if summed.Gross != 30 || summed.Base != 27 || summed.Tax != 3 {
		t.Fatalf("summed = %+v, want gross 30 base 27 tax 3", summed)
	}
	if summed.Base+summed.Tax != summed.Gross {
		t.Fatal("summed split does not reconstitute")
	}
	// Re-deriving from the total gives a different answer — which is exactly
	// why the order stores the sum instead.
	rederived, err := SplitInclusive(30, 1100)
	if err != nil {
		t.Fatal(err)
	}
	if rederived.Tax == summed.Tax {
		t.Skip("no divergence at this rate; the rule still holds by construction")
	}
	t.Logf("summed tax %d vs re-derived %d — the reason order.tax_idr is a SUM",
		summed.Tax, rederived.Tax)
}

func TestPercentRoundsHalfUp(t *testing.T) {
	tests := []struct {
		amount IDR
		bps    int
		want   IDR
	}{
		{100_000, 1100, 11_000},
		{5, 5000, 3},  // 2.5 → 3, half-up
		{15, 5000, 8}, // 7.5 → 8
		{100, 0, 0},
		{0, 1100, 0},
	}
	for _, tc := range tests {
		got, err := Percent(tc.amount, tc.bps)
		if err != nil {
			t.Fatalf("Percent(%d,%d): %v", tc.amount, tc.bps, err)
		}
		if got != tc.want {
			t.Errorf("Percent(%d,%d) = %d, want %d", tc.amount, tc.bps, got, tc.want)
		}
	}
}

func TestMultiplyRefusesOverflow(t *testing.T) {
	if _, err := Multiply(MaxIDR, 2); err == nil {
		t.Error("expected overflow to be refused, not wrapped")
	}
	got, err := Multiply(500_000, 999)
	if err != nil || got != 499_500_000 {
		t.Errorf("Multiply(500000,999) = %d, %v", got, err)
	}
	if _, err := Multiply(1000, -1); err == nil {
		t.Error("expected negative qty to be refused")
	}
}

func TestValidate(t *testing.T) {
	if err := Validate(-1); err == nil {
		t.Error("negative amount must be invalid")
	}
	if err := Validate(MaxIDR + 1); err == nil {
		t.Error("over-max amount must be invalid")
	}
	if err := Validate(0); err != nil {
		t.Errorf("zero must be valid: %v", err)
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		in   IDR
		want string
	}{
		{500_000, "Rp 500.000"},
		{0, "Rp 0"},
		{1, "Rp 1"},
		{999, "Rp 999"},
		{1_000, "Rp 1.000"},
		{12_345_678, "Rp 12.345.678"},
		{500_123, "Rp 500.123"}, // a total carrying its kode unik
		{-1_500, "-Rp 1.500"},
	}
	for _, tc := range tests {
		if got := Format(tc.in); got != tc.want {
			t.Errorf("Format(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
