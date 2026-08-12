// Package pricing resolves what a customer pays. It is pure: no I/O, no SQL,
// no clock. The caller loads candidate rows and hands them in.
//
// The rules, from PROMPT §5 and the decisions in docs/02:
//
//  1. Resolve SCOPE first: the customer's own customer type, else DEFAULT.
//  2. Within the resolved scope, a promo overrides the normal price (D-9).
//     A DEFAULT promo therefore does NOT beat a customer-type normal price —
//     a negotiated corporate rate is not undercut by a public promotion.
//  3. Nothing in either scope is a hard error. Never guess a price (§5.1).
//  4. Tiers are FLAT (D-10): the whole quantity is priced at the rate of the
//     tier its total lands in.
//  5. Every price is tax-INCLUSIVE (D-30); the split is computed here and
//     snapshotted by the caller.
package pricing

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

// Scope is the price scope key: either DEFAULT or one customer type.
type Scope string

// ScopeDefault is the fallback every customer resolves to when their own type
// has no price for the date.
const ScopeDefault Scope = "DEFAULT"

// ScopeFor returns the scope key for a customer type. It must match the
// scope_key generated column in db/migrations/0007_pricing.up.sql exactly.
func ScopeFor(customerType uuid.UUID) Scope {
	if customerType == uuid.Nil {
		return ScopeDefault
	}
	return Scope("CT:" + customerType.String())
}

// Errors a caller is expected to handle rather than log.
var (
	// ErrNotConfigured means no price exists in either scope for that date.
	// The purchase is blocked; the system never invents a number (§5.1.3).
	ErrNotConfigured = errors.New("pricing: no price configured for this scope and date")
	// ErrNoTier means the quantity falls outside every configured tier.
	ErrNoTier = errors.New("pricing: no tier covers this quantity")
	// ErrQtyOutOfRange means the quantity is below 1 or above the configured max.
	ErrQtyOutOfRange = errors.New("pricing: quantity out of range")
)

// Tier is one quantity band. MaxQty nil means unbounded.
type Tier struct {
	ID     uuid.UUID
	Label  string
	MinQty int
	MaxQty *int
	Active bool
}

// Covers reports whether qty falls in this tier.
func (t Tier) Covers(qty int) bool {
	if !t.Active || qty < t.MinQty {
		return false
	}
	return t.MaxQty == nil || qty <= *t.MaxQty
}

// Row is one price row from any of the four tables, normalised.
type Row struct {
	ID         uuid.UUID
	Scope      Scope
	TierID     uuid.UUID // zero for package prices
	PriceIDR   money.IDR
	ValidFrom  time.Time // date, in Asia/Jakarta
	ValidTo    *time.Time
	IsPromo    bool
	PromoLabel string
	Table      string // meal_normal | meal_promo | package_normal | package_promo
	Active     bool
}

// CoversDate reports whether the row is valid on a business date. The range is
// half-open [from, to) — matching the daterange in the migration, so the
// application and the database agree on the boundary.
func (r Row) CoversDate(on time.Time) bool {
	if !r.Active {
		return false
	}
	d := dateOnly(on)
	if d.Before(dateOnly(r.ValidFrom)) {
		return false
	}
	if r.ValidTo != nil && !d.Before(dateOnly(*r.ValidTo)) {
		return false
	}
	return true
}

// Resolved is the answer, carrying everything the order line must snapshot and
// everything the trace must explain.
type Resolved struct {
	UnitPrice   money.IDR
	NormalPrice money.IDR // for the struck-through display; equals UnitPrice when not on promo
	IsPromo     bool
	PromoLabel  string
	Scope       Scope
	TierID      uuid.UUID
	TierLabel   string
	PriceRowID  uuid.UUID
	PriceTable  string
	LineTotal   money.IDR
	Split       money.Split
	// Trace explains the decision without re-running the resolver (docs/01 §3.5).
	Trace Trace
}

// Trace is the human-and-machine-readable "why this price".
type Trace struct {
	RequestedScope Scope  `json:"requested_scope"`
	ResolvedScope  Scope  `json:"resolved_scope"`
	FellBack       bool   `json:"fell_back_to_default"`
	Qty            int    `json:"qty"`
	TierID         string `json:"tier_id"`
	TierLabel      string `json:"tier_label"`
	PriceTable     string `json:"price_table"`
	PriceRowID     string `json:"price_row_id"`
	NormalRowID    string `json:"normal_row_id,omitempty"`
	UnitPriceIDR   int64  `json:"unit_price_idr"`
	NormalPriceIDR int64  `json:"normal_price_idr"`
	TaxRateBps     int    `json:"tax_rate_bps"`
	OnDate         string `json:"on_date"`
}

// Request is one pricing question.
type Request struct {
	CustomerType uuid.UUID // uuid.Nil means the customer has no type — DEFAULT
	Qty          int
	OnDate       time.Time // the business date, already in Asia/Jakarta
	MaxQty       int       // from sys_parameters; 0 means unchecked
	TaxRateBps   int
}

// Catalogue is the candidate data a caller loaded. Both slices may contain rows
// from any scope and any date; the resolver filters.
type Catalogue struct {
	Tiers   []Tier
	Normals []Row
	Promos  []Row
}

// ResolveMeal prices a quantity of meals for one diet type.
//
// The caller has already narrowed the catalogue to a single diet type: diet is
// a lookup key, not a rule, and keeping it out of here keeps the resolver
// honest about what it actually decides.
func ResolveMeal(req Request, cat Catalogue) (Resolved, error) {
	if req.Qty < 1 {
		return Resolved{}, fmt.Errorf("%w: %d", ErrQtyOutOfRange, req.Qty)
	}
	if req.MaxQty > 0 && req.Qty > req.MaxQty {
		return Resolved{}, fmt.Errorf("%w: %d exceeds the maximum of %d", ErrQtyOutOfRange, req.Qty, req.MaxQty)
	}

	tier, ok := TierFor(cat.Tiers, req.Qty)
	if !ok {
		return Resolved{}, fmt.Errorf("%w: qty %d", ErrNoTier, req.Qty)
	}

	requested := ScopeFor(req.CustomerType)

	// Step 1 — scope. The customer's own type wins if it has ANY normal price
	// for the date; only then do we fall back (§5.1).
	scope := requested
	normal, found := pick(cat.Normals, scope, tier.ID, req.OnDate)
	if !found && scope != ScopeDefault {
		scope = ScopeDefault
		normal, found = pick(cat.Normals, scope, tier.ID, req.OnDate)
	}
	if !found {
		return Resolved{}, fmt.Errorf("%w (scope %s, tier %s, date %s)",
			ErrNotConfigured, requested, tier.Label, req.OnDate.Format("2006-01-02"))
	}

	// Step 2 — promo, WITHIN the resolved scope only (D-9).
	chosen := normal
	promo, hasPromo := pick(cat.Promos, scope, tier.ID, req.OnDate)
	if hasPromo {
		chosen = promo
	}

	return build(chosen, normal, hasPromo, scope, requested, tier, req)
}

// ResolvePackage prices one package. Packages have no tiers.
func ResolvePackage(req Request, normals, promos []Row) (Resolved, error) {
	requested := ScopeFor(req.CustomerType)
	scope := requested
	normal, found := pick(normals, scope, uuid.Nil, req.OnDate)
	if !found && scope != ScopeDefault {
		scope = ScopeDefault
		normal, found = pick(normals, scope, uuid.Nil, req.OnDate)
	}
	if !found {
		return Resolved{}, fmt.Errorf("%w (scope %s, date %s)",
			ErrNotConfigured, requested, req.OnDate.Format("2006-01-02"))
	}

	chosen := normal
	promo, hasPromo := pick(promos, scope, uuid.Nil, req.OnDate)
	if hasPromo {
		chosen = promo
	}

	req.Qty = 1
	return build(chosen, normal, hasPromo, scope, requested,
		Tier{Label: "package", Active: true, MinQty: 1}, req)
}

func build(chosen, normal Row, isPromo bool, scope, requested Scope, tier Tier, req Request) (Resolved, error) {
	lineTotal, err := money.Multiply(chosen.PriceIDR, req.Qty)
	if err != nil {
		return Resolved{}, err
	}
	split, err := money.SplitInclusive(lineTotal, req.TaxRateBps)
	if err != nil {
		return Resolved{}, err
	}

	res := Resolved{
		UnitPrice:   chosen.PriceIDR,
		NormalPrice: normal.PriceIDR,
		IsPromo:     isPromo,
		PromoLabel:  chosen.PromoLabel,
		Scope:       scope,
		TierID:      tier.ID,
		TierLabel:   tier.Label,
		PriceRowID:  chosen.ID,
		PriceTable:  chosen.Table,
		LineTotal:   lineTotal,
		Split:       split,
		Trace: Trace{
			RequestedScope: requested,
			ResolvedScope:  scope,
			FellBack:       requested != scope,
			Qty:            req.Qty,
			TierID:         tier.ID.String(),
			TierLabel:      tier.Label,
			PriceTable:     chosen.Table,
			PriceRowID:     chosen.ID.String(),
			UnitPriceIDR:   int64(chosen.PriceIDR),
			NormalPriceIDR: int64(normal.PriceIDR),
			TaxRateBps:     req.TaxRateBps,
			OnDate:         req.OnDate.Format("2006-01-02"),
		},
	}
	if isPromo {
		res.Trace.NormalRowID = normal.ID.String()
	}
	return res, nil
}

// TierFor returns the tier covering qty.
func TierFor(tiers []Tier, qty int) (Tier, bool) {
	for _, t := range tiers {
		if t.Covers(qty) {
			return t, true
		}
	}
	return Tier{}, false
}

// pick returns the single active row for a scope, tier and date. The database's
// exclusion constraint guarantees at most one; if data predating the constraint
// ever produced two, the earliest valid_from wins deterministically rather than
// whatever the query happened to return first.
func pick(rows []Row, scope Scope, tier uuid.UUID, on time.Time) (Row, bool) {
	var best Row
	var found bool
	for _, r := range rows {
		if r.Scope != scope || !r.CoversDate(on) {
			continue
		}
		if tier != uuid.Nil && r.TierID != tier {
			continue
		}
		if !found || dateOnly(r.ValidFrom).Before(dateOnly(best.ValidFrom)) {
			best, found = r, true
		}
	}
	return best, found
}

// ValidateTiers reports the gaps and overlaps an admin form must refuse
// (PROMPT §5.4). Tiers must cover 1..maxQty with no holes.
func ValidateTiers(tiers []Tier, maxQty int) error {
	active := make([]Tier, 0, len(tiers))
	for _, t := range tiers {
		if t.Active {
			active = append(active, t)
		}
	}
	if len(active) == 0 {
		return errors.New("pricing: at least one active tier is required")
	}
	for qty := 1; qty <= maxQty; qty++ {
		n := 0
		for _, t := range active {
			if t.Covers(qty) {
				n++
			}
		}
		if n == 0 {
			return fmt.Errorf("%w: quantity %d is not covered by any tier", ErrNoTier, qty)
		}
		if n > 1 {
			return fmt.Errorf("pricing: quantity %d is covered by %d tiers", qty, n)
		}
	}
	return nil
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
