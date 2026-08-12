// Package order holds the order and delivery state machines and the totalling
// rules. Pure: no I/O.
//
// The order owns the COMMERCIAL lifecycle; the delivery owns FULFILMENT
// (docs/02 D-15). The brief put SCHEDULED → PREPARING → OUT_FOR_DELIVERY →
// DELIVERED on the order, but a package order producing twenty deliveries over
// a month cannot be "out for delivery". Those names live on the delivery, and
// the order exposes a DERIVED fulfilment status so nothing in §6.3 is lost.
package order

import (
	"errors"
	"fmt"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

// Status is the commercial lifecycle.
type Status string

const (
	Draft            Status = "DRAFT"
	AwaitingPayment  Status = "AWAITING_PAYMENT"
	PaymentSubmitted Status = "PAYMENT_SUBMITTED"
	Paid             Status = "PAID"
	Completed        Status = "COMPLETED"
	Expired          Status = "EXPIRED"
	Cancelled        Status = "CANCELLED"
	Refunded         Status = "REFUNDED"
)

// DeliveryStatus is the fulfilment lifecycle.
type DeliveryStatus string

const (
	Scheduled         DeliveryStatus = "SCHEDULED"
	Preparing         DeliveryStatus = "PREPARING"
	OutForDelivery    DeliveryStatus = "OUT_FOR_DELIVERY"
	Delivered         DeliveryStatus = "DELIVERED"
	Failed            DeliveryStatus = "FAILED"
	Skipped           DeliveryStatus = "SKIPPED"
	DeliveryCancelled DeliveryStatus = "CANCELLED"
)

// ErrIllegalTransition is returned for any move not on the machine. Rejected in
// the domain, not the handler (PROMPT §6.3).
var ErrIllegalTransition = errors.New("order: illegal state transition")

// Actor is who is attempting a transition. Some edges are admin-only.
type Actor string

const (
	ActorCustomer Actor = "customer"
	ActorStaff    Actor = "staff"
	ActorFinance  Actor = "finance"
	ActorAdmin    Actor = "admin"
	ActorSystem   Actor = "system" // scheduled jobs
)

type edge struct {
	from Status
	to   Status
}

// transitions is the whole machine. Anything absent is illegal.
var transitions = map[edge][]Actor{
	{Draft, AwaitingPayment}:            {ActorCustomer, ActorStaff, ActorAdmin},
	{AwaitingPayment, PaymentSubmitted}: {ActorCustomer, ActorStaff, ActorAdmin},
	{AwaitingPayment, Expired}:          {ActorSystem},
	{AwaitingPayment, Cancelled}:        {ActorCustomer, ActorStaff, ActorAdmin},
	{PaymentSubmitted, Paid}:            {ActorFinance, ActorAdmin},
	{PaymentSubmitted, AwaitingPayment}: {ActorFinance, ActorAdmin}, // rejected, reason required
	{PaymentSubmitted, Expired}:         {ActorSystem},
	{PaymentSubmitted, Cancelled}:       {ActorStaff, ActorAdmin},
	{Paid, Completed}:                   {ActorSystem, ActorStaff, ActorAdmin},
	{Paid, Cancelled}:                   {ActorStaff, ActorAdmin},
	// No refunds is the policy (D-31). This edge exists only for an erroneous
	// or duplicate transfer and is admin-only, with a mandatory reason.
	{Paid, Refunded}:      {ActorAdmin},
	{Cancelled, Refunded}: {ActorAdmin},
}

// CanTransition reports whether an actor may make a move.
func CanTransition(from, to Status, by Actor) bool {
	actors, ok := transitions[edge{from, to}]
	if !ok {
		return false
	}
	for _, a := range actors {
		if a == by {
			return true
		}
	}
	return false
}

// Transition validates a move.
func Transition(from, to Status, by Actor) error {
	if from == to {
		return fmt.Errorf("%w: already %s", ErrIllegalTransition, from)
	}
	if _, ok := transitions[edge{from, to}]; !ok {
		return fmt.Errorf("%w: %s → %s", ErrIllegalTransition, from, to)
	}
	if !CanTransition(from, to, by) {
		return fmt.Errorf("%w: %s may not move %s → %s", ErrIllegalTransition, by, from, to)
	}
	return nil
}

// IsTerminal reports whether a status can never change again.
func IsTerminal(s Status) bool {
	switch s {
	case Completed, Expired, Refunded:
		return true
	}
	return false
}

// AwaitsMoney reports whether capacity is being held against an unpaid order.
// These are the orders auto-expiry reclaims (PROMPT §10) — and the only ones
// anything automated may touch. Nothing automatic ever cancels a PAID order or
// a scheduled delivery (99 §8, docs/03 Q-14).
func AwaitsMoney(s Status) bool {
	return s == AwaitingPayment || s == PaymentSubmitted
}

var deliveryTransitions = map[struct {
	from DeliveryStatus
	to   DeliveryStatus
}]bool{
	{Scheduled, Preparing}:         true,
	{Scheduled, Skipped}:           true, // before cut-off only; caller checks
	{Scheduled, DeliveryCancelled}: true,
	{Scheduled, Scheduled}:         true, // reschedule / re-route before cut-off
	{Preparing, OutForDelivery}:    true,
	{Preparing, DeliveryCancelled}: true,
	{OutForDelivery, Delivered}:    true,
	{OutForDelivery, Failed}:       true,
	{Failed, Scheduled}:            true, // staff reschedules; no automatic credit
}

// TransitionDelivery validates a fulfilment move.
func TransitionDelivery(from, to DeliveryStatus) error {
	if !deliveryTransitions[struct {
		from DeliveryStatus
		to   DeliveryStatus
	}{from, to}] {
		return fmt.Errorf("%w: delivery %s → %s", ErrIllegalTransition, from, to)
	}
	return nil
}

// FulfilmentStatus derives the order-level fulfilment view from its deliveries,
// so the API can still answer §6.3's vocabulary (D-15).
//
// The rule is "least advanced wins": an order with nineteen delivered meals and
// one still scheduled is not delivered.
func FulfilmentStatus(deliveries []DeliveryStatus) DeliveryStatus {
	if len(deliveries) == 0 {
		return Scheduled
	}
	rank := map[DeliveryStatus]int{
		Scheduled: 0, Preparing: 1, OutForDelivery: 2, Delivered: 3,
		Failed: 0, Skipped: 4, DeliveryCancelled: 4,
	}
	least := Delivered
	leastRank := 99
	live := 0
	for _, d := range deliveries {
		if d == Skipped || d == DeliveryCancelled {
			continue
		}
		live++
		if r := rank[d]; r < leastRank {
			leastRank, least = r, d
		}
	}
	if live == 0 {
		return Skipped
	}
	return least
}

// Totals is an order's money, all tax-inclusive except the split.
type Totals struct {
	Subtotal    money.IDR
	DeliveryFee money.IDR
	Discount    money.IDR
	Total       money.IDR
	TaxBase     money.IDR
	Tax         money.IDR
	TaxRateBps  int
	// UniqueCode is the Indonesian kode unik (D-16); Rounding is what it adds.
	UniqueCode    int
	Rounding      money.IDR
	PaymentAmount money.IDR
}

// Line is one priced order line, already resolved.
type Line struct {
	LineTotal money.IDR
	Split     money.Split
}

// Compute builds the order totals.
//
// The tax is the SUM of the line taxes and is never re-derived from the total
// (D-30). The delivery fee is a taxable supply and is split the same way. The
// unique code is NOT taxable — a matching device is not consideration — so it
// is added after the split and reported separately.
func Compute(lines []Line, deliveryFee, discount money.IDR, taxRateBps, uniqueCode int) (Totals, error) {
	t := Totals{TaxRateBps: taxRateBps, DeliveryFee: deliveryFee, Discount: discount}

	splits := make([]money.Split, 0, len(lines)+1)
	for i, l := range lines {
		// A line split computed at a different rate from the order's would
		// produce an invoice whose tax does not match its own rate, silently.
		// Refuse rather than sum inconsistent parts.
		if l.Split.RateBps != taxRateBps {
			return Totals{}, fmt.Errorf(
				"order: line %d was split at %d bps but the order is at %d bps",
				i+1, l.Split.RateBps, taxRateBps)
		}
		if l.Split.Gross != l.LineTotal {
			return Totals{}, fmt.Errorf(
				"order: line %d split gross %d does not match its total %d",
				i+1, l.Split.Gross, l.LineTotal)
		}
		t.Subtotal += l.LineTotal
		splits = append(splits, l.Split)
	}

	if deliveryFee > 0 {
		fs, err := money.SplitInclusive(deliveryFee, taxRateBps)
		if err != nil {
			return Totals{}, err
		}
		splits = append(splits, fs)
	}

	t.Total = t.Subtotal + t.DeliveryFee - t.Discount
	if t.Total < 0 {
		return Totals{}, fmt.Errorf("order: discount %d exceeds subtotal plus fee", discount)
	}

	summed := money.SumSplits(splits, taxRateBps)
	t.TaxBase, t.Tax = summed.Base, summed.Tax

	// A discount changes the gross, so the split has to follow it down rather
	// than being carried over from the undiscounted lines.
	if discount > 0 {
		ds, err := money.SplitInclusive(t.Total, taxRateBps)
		if err != nil {
			return Totals{}, err
		}
		t.TaxBase, t.Tax = ds.Base, ds.Tax
	}

	if t.TaxBase+t.Tax != t.Total {
		return Totals{}, fmt.Errorf("order: tax split %d+%d does not reconstitute total %d",
			t.TaxBase, t.Tax, t.Total)
	}

	if uniqueCode > 0 {
		t.UniqueCode = uniqueCode
		t.Rounding = money.IDR(uniqueCode)
	}
	t.PaymentAmount = t.Total + t.Rounding
	return t, nil
}
