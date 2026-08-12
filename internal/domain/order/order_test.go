package order

import (
	"errors"
	"testing"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

func TestLegalTransitions(t *testing.T) {
	tests := []struct {
		from, to Status
		by       Actor
		ok       bool
	}{
		{Draft, AwaitingPayment, ActorCustomer, true},
		{AwaitingPayment, PaymentSubmitted, ActorCustomer, true},
		{PaymentSubmitted, Paid, ActorFinance, true},
		{PaymentSubmitted, AwaitingPayment, ActorFinance, true}, // rejected
		{Paid, Completed, ActorSystem, true},

		// Illegal shapes.
		{Draft, Paid, ActorAdmin, false},
		{AwaitingPayment, Completed, ActorAdmin, false},
		{Completed, Paid, ActorAdmin, false},
		{Expired, AwaitingPayment, ActorCustomer, false},
		{Paid, Draft, ActorAdmin, false},
	}
	for _, tc := range tests {
		err := Transition(tc.from, tc.to, tc.by)
		if (err == nil) != tc.ok {
			t.Errorf("%s → %s by %s: err = %v, want ok=%v", tc.from, tc.to, tc.by, err, tc.ok)
		}
	}
}

// Only the system expires an order, and only an unpaid one. Nothing automated
// touches a paid order (99 §8).
func TestOnlySystemExpiresAndOnlyUnpaid(t *testing.T) {
	if err := Transition(AwaitingPayment, Expired, ActorSystem); err != nil {
		t.Errorf("the sweeper must be able to expire an unpaid order: %v", err)
	}
	if err := Transition(AwaitingPayment, Expired, ActorCustomer); err == nil {
		t.Error("a customer must not be able to drive the EXPIRED state directly")
	}
	if err := Transition(Paid, Expired, ActorSystem); err == nil {
		t.Error("a PAID order must never be expired by anything automated")
	}
	if !AwaitsMoney(AwaitingPayment) || !AwaitsMoney(PaymentSubmitted) {
		t.Error("both unpaid states hold capacity and must be sweepable")
	}
	if AwaitsMoney(Paid) || AwaitsMoney(Completed) {
		t.Error("a paid order does not await money")
	}
}

// D-31: no refunds. The edge survives for erroneous transfers only, and only an
// admin may walk it.
func TestRefundIsAdminOnly(t *testing.T) {
	if err := Transition(Paid, Refunded, ActorAdmin); err != nil {
		t.Errorf("an admin must be able to return an erroneous transfer: %v", err)
	}
	for _, actor := range []Actor{ActorFinance, ActorStaff, ActorCustomer, ActorSystem} {
		if err := Transition(Paid, Refunded, actor); err == nil {
			t.Errorf("%s must not be able to refund — the policy is no refunds (D-31)", actor)
		}
	}
}

func TestSameStateIsNotATransition(t *testing.T) {
	if err := Transition(Paid, Paid, ActorAdmin); !errors.Is(err, ErrIllegalTransition) {
		t.Errorf("err = %v, want ErrIllegalTransition", err)
	}
}

func TestTerminalStates(t *testing.T) {
	for _, s := range []Status{Completed, Expired, Refunded} {
		if !IsTerminal(s) {
			t.Errorf("%s must be terminal", s)
		}
	}
	for _, s := range []Status{Draft, AwaitingPayment, PaymentSubmitted, Paid, Cancelled} {
		if IsTerminal(s) {
			t.Errorf("%s must not be terminal", s)
		}
	}
}

func TestDeliveryTransitions(t *testing.T) {
	tests := []struct {
		from, to DeliveryStatus
		ok       bool
	}{
		{Scheduled, Preparing, true},
		{Scheduled, Skipped, true},
		{Preparing, OutForDelivery, true},
		{OutForDelivery, Delivered, true},
		{OutForDelivery, Failed, true},
		{Failed, Scheduled, true},
		{Delivered, Failed, false},
		{Delivered, Scheduled, false},
		{Skipped, Preparing, false},
		{Scheduled, Delivered, false}, // no skipping the kitchen
	}
	for _, tc := range tests {
		err := TransitionDelivery(tc.from, tc.to)
		if (err == nil) != tc.ok {
			t.Errorf("delivery %s → %s: err = %v, want ok=%v", tc.from, tc.to, err, tc.ok)
		}
	}
}

// D-15: the derived view the API exposes so §6.3's vocabulary survives.
func TestFulfilmentStatusIsLeastAdvanced(t *testing.T) {
	tests := []struct {
		name string
		in   []DeliveryStatus
		want DeliveryStatus
	}{
		{"no deliveries yet", nil, Scheduled},
		{"all delivered", []DeliveryStatus{Delivered, Delivered}, Delivered},
		{"nineteen delivered, one scheduled", []DeliveryStatus{Delivered, Delivered, Scheduled}, Scheduled},
		{"one preparing", []DeliveryStatus{Delivered, Preparing}, Preparing},
		{"skips are ignored", []DeliveryStatus{Delivered, Skipped}, Delivered},
		{"everything skipped", []DeliveryStatus{Skipped, DeliveryCancelled}, Skipped},
		{"a failure is not progress", []DeliveryStatus{Delivered, Failed}, Failed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FulfilmentStatus(tc.in); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func linesAt(bps int, totals ...money.IDR) []Line {
	out := make([]Line, 0, len(totals))
	for _, tt := range totals {
		s, err := money.SplitInclusive(tt, bps)
		if err != nil {
			panic(err)
		}
		out = append(out, Line{LineTotal: tt, Split: s})
	}
	return out
}

// Most tests run at the seeded 11%.
func lines(totals ...money.IDR) []Line { return linesAt(1100, totals...) }

func TestComputeTotals(t *testing.T) {
	got, err := Compute(lines(200_000, 225_000), 15_000, 0, 1100, 123)
	if err != nil {
		t.Fatal(err)
	}
	if got.Subtotal != 425_000 {
		t.Errorf("subtotal = %d, want 425000", got.Subtotal)
	}
	if got.Total != 440_000 {
		t.Errorf("total = %d, want 440000", got.Total)
	}
	// The kode unik rides on top and is not taxable (D-16, D-30).
	if got.PaymentAmount != 440_123 {
		t.Errorf("payment amount = %d, want 440123", got.PaymentAmount)
	}
	if got.Rounding != 123 {
		t.Errorf("rounding = %d, want 123", got.Rounding)
	}
	if got.TaxBase+got.Tax != got.Total {
		t.Errorf("tax split %d+%d does not reconstitute total %d", got.TaxBase, got.Tax, got.Total)
	}
	// The suffix must not have inflated the tax base.
	if got.TaxBase+got.Tax == got.PaymentAmount {
		t.Error("the payment suffix must be excluded from the tax base")
	}
}

func TestComputeWithoutUniqueCode(t *testing.T) {
	got, err := Compute(lines(100_000), 0, 0, 1100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.PaymentAmount != got.Total || got.Rounding != 0 {
		t.Errorf("no suffix: payment %d total %d rounding %d", got.PaymentAmount, got.Total, got.Rounding)
	}
}

func TestComputeAppliesDiscountToTheTaxSplit(t *testing.T) {
	got, err := Compute(lines(100_000), 0, 20_000, 1100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 80_000 {
		t.Errorf("total = %d, want 80000", got.Total)
	}
	if got.TaxBase+got.Tax != got.Total {
		t.Errorf("a discount must pull the split down with it: %d+%d != %d", got.TaxBase, got.Tax, got.Total)
	}
}

func TestComputeRejectsOverDiscount(t *testing.T) {
	if _, err := Compute(lines(50_000), 0, 60_000, 1100, 0); err == nil {
		t.Error("a discount larger than the order must be refused, not produce a negative total")
	}
}

// A line split at a different rate from the order's must be refused, not
// silently summed into an invoice that disagrees with its own rate.
func TestComputeRejectsMismatchedLineRate(t *testing.T) {
	if _, err := Compute(linesAt(1100, 100_000), 0, 0, 1200, 0); err == nil {
		t.Error("a line split at 1100 bps inside a 1200 bps order must be refused")
	}
	bad := lines(100_000)
	bad[0].LineTotal = 90_000 // split no longer describes the line
	if _, err := Compute(bad, 0, 0, 1100, 0); err == nil {
		t.Error("a split whose gross does not match its line total must be refused")
	}
}

func TestComputeZeroTaxRate(t *testing.T) {
	got, err := Compute(linesAt(0, 100_000), 0, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tax != 0 || got.TaxBase != 100_000 {
		t.Errorf("at 0 bps: base %d tax %d, want 100000 / 0", got.TaxBase, got.Tax)
	}
}
