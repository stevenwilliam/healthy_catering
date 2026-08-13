// Package credit is the meal-credit ledger.
//
// The balance is NEVER stored (PROMPT §7): remaining = SUM(entries.Qty). This
// package holds the pure rules; the repository supplies the entries under a
// row lock and appends what these functions return.
//
// One credit buys one MEAL, whatever that meal contains — a single dish and a
// four-component set both cost one credit (docs/02 D-32, Steven 2026-08-12).
// Nothing here ever counts foods.
package credit

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EntryType is a ledger movement kind.
type EntryType string

const (
	Purchase   EntryType = "PURCHASE"   // + credits bought
	Redeem     EntryType = "REDEEM"     // − one meal taken
	Refund     EntryType = "REFUND"     // + a skip before cut-off returned it
	Expire     EntryType = "EXPIRE"     // − the remainder at expiry
	Adjustment EntryType = "ADJUSTMENT" // ± staff correction, reason required
)

// Errors callers handle rather than log.
var (
	ErrInsufficient   = errors.New("credit: no credits remaining")
	ErrNotActive      = errors.New("credit: package is not active")
	ErrExpired        = errors.New("credit: package has expired")
	ErrAfterExpiry    = errors.New("credit: delivery date is after the package expires")
	ErrReasonRequired = errors.New("credit: an adjustment requires a reason")
	ErrBadSign        = errors.New("credit: entry sign does not match its type")
)

// Entry is one ledger row.
type Entry struct {
	ID            uuid.UUID
	PackageID     uuid.UUID
	CustomerID    uuid.UUID
	Type          EntryType
	Qty           int // signed
	ReferenceType string
	ReferenceID   uuid.UUID
	Note          string
	OccurredAt    time.Time
	CreatedBy     uuid.UUID
}

// Validate mirrors the CHECK constraints in db/migrations/0009_credits.up.sql.
// Both exist deliberately: the database is the last line, this is the first,
// and the error a customer sees comes from here.
func (e Entry) Validate() error {
	if e.Qty == 0 {
		return fmt.Errorf("%w: zero movement", ErrBadSign)
	}
	switch e.Type {
	case Purchase, Refund:
		if e.Qty < 0 {
			return fmt.Errorf("%w: %s must be positive", ErrBadSign, e.Type)
		}
	case Redeem, Expire:
		if e.Qty > 0 {
			return fmt.Errorf("%w: %s must be negative", ErrBadSign, e.Type)
		}
	case Adjustment:
		if e.Note == "" {
			return ErrReasonRequired
		}
	default:
		return fmt.Errorf("credit: unknown entry type %q", e.Type)
	}
	return nil
}

// Status is a package's lifecycle state.
type Status string

const (
	Pending   Status = "PENDING"
	Active    Status = "ACTIVE"
	Exhausted Status = "EXHAUSTED"
	Expired   Status = "EXPIRED"
	Cancelled Status = "CANCELLED"
)

// Package is the purchased instance, as the rules need to see it.
type Package struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	Status      Status
	Credits     int
	ActivatedAt *time.Time
	ExpiresAt   *time.Time // a business DATE in Asia/Jakarta
}

// Balance sums a ledger. This is the only definition of "remaining" in the
// system; nothing caches it.
func Balance(entries []Entry) int {
	total := 0
	for _, e := range entries {
		total += e.Qty
	}
	return total
}

// RedeemOne returns the entry that spends one credit for one meal.
//
// The caller must already hold a row lock on the package and must have read
// `entries` inside that lock — otherwise two tabs can both see the last credit.
// deliveryID makes the entry idempotent: the unique index on
// (reference_type, reference_id) WHERE entry_type='REDEEM' means a retry cannot
// spend a second credit for the same meal.
func RedeemOne(p Package, entries []Entry, deliveryID uuid.UUID, serviceDate time.Time, now time.Time) (Entry, error) {
	// Each non-Active status has its own truthful answer. Collapsing them into
	// one "not active" message told a customer who had just spent their last
	// credit that we were still waiting for their transfer — which is both
	// wrong and alarming.
	switch p.Status {
	case Active:
		// carry on
	case Exhausted:
		return Entry{}, ErrInsufficient
	case Expired:
		return Entry{}, ErrExpired
	default:
		return Entry{}, fmt.Errorf("%w: status %s", ErrNotActive, p.Status)
	}
	if p.ExpiresAt != nil {
		if dateOnly(now).After(dateOnly(*p.ExpiresAt)) {
			return Entry{}, ErrExpired
		}
		// D-27: a credit redeemed on the last valid day must not schedule a
		// delivery a month later, or the package never really expires.
		if dateOnly(serviceDate).After(dateOnly(*p.ExpiresAt)) {
			return Entry{}, fmt.Errorf("%w: %s is after %s",
				ErrAfterExpiry, serviceDate.Format("2006-01-02"), p.ExpiresAt.Format("2006-01-02"))
		}
	}
	if Balance(entries) < 1 {
		return Entry{}, ErrInsufficient
	}
	return Entry{
		PackageID: p.ID, CustomerID: p.CustomerID, Type: Redeem, Qty: -1,
		ReferenceType: "delivery", ReferenceID: deliveryID, OccurredAt: now,
	}, nil
}

// ReturnForSkip returns the credit for a delivery skipped BEFORE the cut-off.
// After the cut-off the food is committed and the credit is not returned
// (PROMPT §8.3).
func ReturnForSkip(p Package, deliveryID uuid.UUID, beforeCutoff bool, now time.Time) (Entry, bool) {
	if !beforeCutoff {
		return Entry{}, false
	}
	return Entry{
		PackageID: p.ID, CustomerID: p.CustomerID, Type: Refund, Qty: 1,
		ReferenceType: "delivery", ReferenceID: deliveryID,
		Note: "skipped before cut-off", OccurredAt: now,
	}, true
}

// ExpireRemainder returns the negative entry that zeroes a package at expiry,
// and whether one is needed. Unused credits are forfeited — Steven, 2026-08-12,
// "no refund" (D-31) — so this posts no money anywhere.
func ExpireRemainder(p Package, entries []Entry, now time.Time) (Entry, bool) {
	remaining := Balance(entries)
	if remaining <= 0 {
		return Entry{}, false
	}
	return Entry{
		PackageID: p.ID, CustomerID: p.CustomerID, Type: Expire, Qty: -remaining,
		ReferenceType: "package", ReferenceID: p.ID,
		Note: "expired, credits forfeited", OccurredAt: now,
	}, true
}

// ReverseExpiry is what an extension posts. The EXPIRE entry is never deleted —
// the ledger is append-only, so an extension compensates rather than rewrites
// (docs/02 D-28).
func ReverseExpiry(p Package, entries []Entry, reason string, by uuid.UUID, now time.Time) (Entry, error) {
	if reason == "" {
		return Entry{}, ErrReasonRequired
	}
	var expired int
	for _, e := range entries {
		if e.Type == Expire {
			expired += -e.Qty
		}
	}
	if expired == 0 {
		return Entry{}, fmt.Errorf("credit: nothing to reverse, no EXPIRE entry")
	}
	return Entry{
		PackageID: p.ID, CustomerID: p.CustomerID, Type: Adjustment, Qty: expired,
		ReferenceType: "package", ReferenceID: p.ID,
		Note: "expiry reversed: " + reason, CreatedBy: by, OccurredAt: now,
	}, nil
}

// Goodwill is the compensation path that exists because there are no refunds
// (D-31): a cancelled paid order or an unfulfillable delivery gives credits.
func Goodwill(p Package, qty int, reason string, by uuid.UUID, now time.Time) (Entry, error) {
	if reason == "" {
		return Entry{}, ErrReasonRequired
	}
	if qty == 0 {
		return Entry{}, fmt.Errorf("%w: zero movement", ErrBadSign)
	}
	return Entry{
		PackageID: p.ID, CustomerID: p.CustomerID, Type: Adjustment, Qty: qty,
		ReferenceType: "goodwill", Note: reason, CreatedBy: by, OccurredAt: now,
	}, nil
}

// NextStatus derives the package status from its balance and the date. Status
// is a cache of these two facts, and deriving it in one place stops the cache
// drifting from the ledger.
func NextStatus(p Package, entries []Entry, now time.Time) Status {
	switch p.Status {
	case Pending, Cancelled:
		return p.Status
	}
	if p.ExpiresAt != nil && dateOnly(now).After(dateOnly(*p.ExpiresAt)) {
		return Expired
	}
	if Balance(entries) <= 0 {
		return Exhausted
	}
	return Active
}

// Activate starts the active period on payment verification, not purchase
// (docs/02 D-14): manual transfer can lag by a weekend and the customer should
// not lose validity days they could not order in.
func Activate(p Package, validityDays int, verifiedAt time.Time, loc *time.Location) (activatedAt time.Time, expiresAt time.Time) {
	activatedAt = verifiedAt
	local := verifiedAt.In(loc)
	expiresAt = time.Date(local.Year(), local.Month(), local.Day()+validityDays, 0, 0, 0, 0, loc)
	return activatedAt, expiresAt
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
