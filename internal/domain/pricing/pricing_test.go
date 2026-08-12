package pricing

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

// The test matrix from docs/01 §5.1, which is the matrix PROMPT §14 demands
// near-100% coverage of.

var (
	companyA = uuid.MustParse("00000000-0000-7000-8000-0000000000a1")
	siloam   = uuid.MustParse("00000000-0000-7000-8000-0000000000a2")
	tier1ID  = uuid.MustParse("00000000-0000-7000-8000-0000000000b1")
	tier2ID  = uuid.MustParse("00000000-0000-7000-8000-0000000000b2")
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptr[T any](v T) *T { return &v }

func tiers() []Tier {
	return []Tier{
		{ID: tier1ID, Label: "1-4", MinQty: 1, MaxQty: ptr(4), Active: true},
		{ID: tier2ID, Label: "5+", MinQty: 5, MaxQty: nil, Active: true},
	}
}

func row(id string, scope Scope, tier uuid.UUID, price money.IDR, from string, to *string, promo bool) Row {
	r := Row{
		ID: uuid.MustParse(id), Scope: scope, TierID: tier, PriceIDR: price,
		ValidFrom: date(from), IsPromo: promo, Active: true,
	}
	if to != nil {
		r.ValidTo = ptr(date(*to))
	}
	if promo {
		r.Table = "meal_promo"
		r.PromoLabel = "Promo Agustus"
	} else {
		r.Table = "meal_normal"
	}
	return r
}

func baseCatalogue() Catalogue {
	return Catalogue{
		Tiers: tiers(),
		Normals: []Row{
			row("00000000-0000-7000-8000-0000000000c1", ScopeDefault, tier1ID, 50_000, "2026-01-01", nil, false),
			row("00000000-0000-7000-8000-0000000000c2", ScopeDefault, tier2ID, 45_000, "2026-01-01", nil, false),
		},
	}
}

func req(ct uuid.UUID, qty int, on string) Request {
	return Request{CustomerType: ct, Qty: qty, OnDate: date(on), MaxQty: 999, TaxRateBps: 1100}
}

func TestScopeHit(t *testing.T) {
	cat := baseCatalogue()
	cat.Normals = append(cat.Normals,
		row("00000000-0000-7000-8000-0000000000c3", ScopeFor(companyA), tier1ID, 55_000, "2026-01-01", nil, false))

	got, err := ResolveMeal(req(companyA, 2, "2026-09-01"), cat)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitPrice != 55_000 {
		t.Errorf("unit price = %d, want 55000", got.UnitPrice)
	}
	if got.Scope != ScopeFor(companyA) || got.Trace.FellBack {
		t.Errorf("scope = %s fellBack = %v, want the customer type without fallback", got.Scope, got.Trace.FellBack)
	}
}

func TestScopeMissFallsBackToDefault(t *testing.T) {
	got, err := ResolveMeal(req(companyA, 2, "2026-09-01"), baseCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitPrice != 50_000 {
		t.Errorf("unit price = %d, want the DEFAULT 50000", got.UnitPrice)
	}
	if got.Scope != ScopeDefault || !got.Trace.FellBack {
		t.Errorf("expected a recorded fallback to DEFAULT, got scope %s fellBack %v", got.Scope, got.Trace.FellBack)
	}
}

func TestBothScopesMissingBlocks(t *testing.T) {
	cat := Catalogue{Tiers: tiers()}
	_, err := ResolveMeal(req(companyA, 2, "2026-09-01"), cat)
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured — a missing price must never be guessed", err)
	}
}

func TestPromoOverridesNormalInSameScope(t *testing.T) {
	cat := baseCatalogue()
	cat.Promos = []Row{
		row("00000000-0000-7000-8000-0000000000d1", ScopeDefault, tier1ID, 40_000, "2026-08-01", ptr("2026-09-01"), true),
	}
	got, err := ResolveMeal(req(uuid.Nil, 2, "2026-08-15"), cat)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitPrice != 40_000 || !got.IsPromo {
		t.Errorf("unit = %d promo = %v, want the 40000 promo", got.UnitPrice, got.IsPromo)
	}
	// The struck-through price the cart must show.
	if got.NormalPrice != 50_000 {
		t.Errorf("normal price = %d, want 50000 for the strike-through", got.NormalPrice)
	}
	if got.PromoLabel == "" {
		t.Error("promo label must survive to the cart")
	}
}

// D-9, the decision that materially changes what customers pay: a DEFAULT promo
// must NOT undercut a negotiated corporate normal price.
func TestCustomerTypeNormalBeatsDefaultPromo(t *testing.T) {
	cat := baseCatalogue()
	cat.Normals = append(cat.Normals,
		row("00000000-0000-7000-8000-0000000000c3", ScopeFor(companyA), tier1ID, 55_000, "2026-01-01", nil, false))
	cat.Promos = []Row{
		row("00000000-0000-7000-8000-0000000000d1", ScopeDefault, tier1ID, 40_000, "2026-08-01", ptr("2026-09-01"), true),
	}

	got, err := ResolveMeal(req(companyA, 2, "2026-08-15"), cat)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitPrice != 55_000 {
		t.Fatalf("unit price = %d, want Company A's negotiated 55000 (D-9)", got.UnitPrice)
	}
	if got.IsPromo {
		t.Error("a DEFAULT promo must not apply to a customer-type scope")
	}

	// And the walk-in on the same day does get the promo — the asymmetry
	// D-9 warns about, asserted so it is deliberate rather than discovered.
	walkIn, err := ResolveMeal(req(uuid.Nil, 2, "2026-08-15"), cat)
	if err != nil {
		t.Fatal(err)
	}
	if walkIn.UnitPrice >= got.UnitPrice {
		t.Errorf("walk-in %d should pay less than Company A %d during a DEFAULT promo",
			walkIn.UnitPrice, got.UnitPrice)
	}
}

func TestPromoInCustomerScopeApplies(t *testing.T) {
	cat := baseCatalogue()
	cat.Normals = append(cat.Normals,
		row("00000000-0000-7000-8000-0000000000c3", ScopeFor(companyA), tier1ID, 55_000, "2026-01-01", nil, false))
	cat.Promos = []Row{
		row("00000000-0000-7000-8000-0000000000d2", ScopeFor(companyA), tier1ID, 48_000, "2026-08-01", ptr("2026-09-01"), true),
	}
	got, err := ResolveMeal(req(companyA, 2, "2026-08-15"), cat)
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitPrice != 48_000 || !got.IsPromo {
		t.Errorf("unit = %d promo = %v, want Company A's own promo at 48000", got.UnitPrice, got.IsPromo)
	}
}

// Flat tiering (D-10): ten meals are all priced at the tier-2 rate.
func TestTierBoundaries(t *testing.T) {
	tests := []struct {
		qty       int
		wantUnit  money.IDR
		wantTier  string
		wantTotal money.IDR
	}{
		{1, 50_000, "1-4", 50_000},
		{4, 50_000, "1-4", 200_000},
		{5, 45_000, "5+", 225_000},
		{10, 45_000, "5+", 450_000},
		{999, 45_000, "5+", 44_955_000},
	}
	for _, tc := range tests {
		got, err := ResolveMeal(req(uuid.Nil, tc.qty, "2026-09-01"), baseCatalogue())
		if err != nil {
			t.Fatalf("qty %d: %v", tc.qty, err)
		}
		if got.UnitPrice != tc.wantUnit || got.TierLabel != tc.wantTier {
			t.Errorf("qty %d: unit %d tier %q, want %d %q", tc.qty, got.UnitPrice, got.TierLabel, tc.wantUnit, tc.wantTier)
		}
		if got.LineTotal != tc.wantTotal {
			t.Errorf("qty %d: total %d, want %d (flat tiering, D-10)", tc.qty, got.LineTotal, tc.wantTotal)
		}
	}
}

func TestQtyOutOfRange(t *testing.T) {
	for _, qty := range []int{0, -1, 1000} {
		if _, err := ResolveMeal(req(uuid.Nil, qty, "2026-09-01"), baseCatalogue()); !errors.Is(err, ErrQtyOutOfRange) {
			t.Errorf("qty %d: err = %v, want ErrQtyOutOfRange", qty, err)
		}
	}
}

func TestNoTierCoversQty(t *testing.T) {
	cat := baseCatalogue()
	cat.Tiers = []Tier{{ID: tier2ID, Label: "5+", MinQty: 5, Active: true}}
	if _, err := ResolveMeal(req(uuid.Nil, 2, "2026-09-01"), cat); !errors.Is(err, ErrNoTier) {
		t.Errorf("err = %v, want ErrNoTier", err)
	}
}

// The half-open [from, to) boundary, which must match the daterange in the
// migration exactly or the application and the database disagree by a day.
func TestValidityBoundaries(t *testing.T) {
	cat := Catalogue{
		Tiers: tiers(),
		Normals: []Row{
			row("00000000-0000-7000-8000-0000000000e1", ScopeDefault, tier1ID, 50_000, "2026-09-01", ptr("2026-10-01"), false),
		},
	}
	tests := []struct {
		on   string
		want bool
	}{
		{"2026-08-31", false}, // before valid_from
		{"2026-09-01", true},  // valid_from is inclusive
		{"2026-09-30", true},
		{"2026-10-01", false}, // valid_to is exclusive
		{"2026-10-02", false},
	}
	for _, tc := range tests {
		_, err := ResolveMeal(req(uuid.Nil, 1, tc.on), cat)
		got := err == nil
		if got != tc.want {
			t.Errorf("date %s: resolved = %v, want %v", tc.on, got, tc.want)
		}
	}
}

func TestOpenEndedValidity(t *testing.T) {
	cat := Catalogue{
		Tiers: tiers(),
		Normals: []Row{
			row("00000000-0000-7000-8000-0000000000e2", ScopeDefault, tier1ID, 50_000, "2026-09-01", nil, false),
		},
	}
	if _, err := ResolveMeal(req(uuid.Nil, 1, "2099-01-01"), cat); err != nil {
		t.Errorf("open-ended price must still resolve far in the future: %v", err)
	}
}

func TestInactiveRowIgnored(t *testing.T) {
	cat := baseCatalogue()
	cat.Normals[0].Active = false
	cat.Normals[1].Active = false
	if _, err := ResolveMeal(req(uuid.Nil, 1, "2026-09-01"), cat); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured when every row is inactive", err)
	}
}

// The tax split rides with the price so the caller snapshots one consistent
// answer (D-30).
func TestResolvedCarriesTaxSplit(t *testing.T) {
	got, err := ResolveMeal(req(uuid.Nil, 10, "2026-09-01"), baseCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	if got.Split.Gross != got.LineTotal {
		t.Errorf("split gross %d != line total %d", got.Split.Gross, got.LineTotal)
	}
	if got.Split.Base+got.Split.Tax != got.LineTotal {
		t.Error("split must reconstitute the line total")
	}
	if got.Split.RateBps != 1100 {
		t.Errorf("rate = %d, want the requested 1100 snapshotted", got.Split.RateBps)
	}
}

func TestPackagePricing(t *testing.T) {
	normals := []Row{{
		ID: uuid.MustParse("00000000-0000-7000-8000-0000000000f1"), Scope: ScopeDefault,
		PriceIDR: 900_000, ValidFrom: date("2026-01-01"), Active: true, Table: "package_normal",
	}}
	promos := []Row{{
		ID: uuid.MustParse("00000000-0000-7000-8000-0000000000f2"), Scope: ScopeDefault,
		PriceIDR: 800_000, ValidFrom: date("2026-08-01"), ValidTo: ptr(date("2026-09-01")),
		Active: true, IsPromo: true, Table: "package_promo", PromoLabel: "Promo Agustus",
	}}

	onPromo, err := ResolvePackage(req(uuid.Nil, 1, "2026-08-15"), normals, promos)
	if err != nil {
		t.Fatal(err)
	}
	if onPromo.UnitPrice != 800_000 || onPromo.NormalPrice != 900_000 {
		t.Errorf("got %d / %d, want 800000 promo over 900000 normal", onPromo.UnitPrice, onPromo.NormalPrice)
	}

	after, err := ResolvePackage(req(uuid.Nil, 1, "2026-09-15"), normals, promos)
	if err != nil {
		t.Fatal(err)
	}
	if after.UnitPrice != 900_000 || after.IsPromo {
		t.Errorf("after the promo window: got %d promo=%v, want 900000 and no promo", after.UnitPrice, after.IsPromo)
	}
}

func TestValidateTiers(t *testing.T) {
	tests := []struct {
		name    string
		tiers   []Tier
		wantErr bool
	}{
		{"contiguous cover", tiers(), false},
		{"gap at 5", []Tier{
			{ID: tier1ID, MinQty: 1, MaxQty: ptr(4), Active: true},
			{ID: tier2ID, MinQty: 6, Active: true},
		}, true},
		{"overlap at 4", []Tier{
			{ID: tier1ID, MinQty: 1, MaxQty: ptr(4), Active: true},
			{ID: tier2ID, MinQty: 4, Active: true},
		}, true},
		{"does not start at 1", []Tier{{ID: tier1ID, MinQty: 2, Active: true}}, true},
		{"no active tiers", []Tier{{ID: tier1ID, MinQty: 1, Active: false}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTiers(tc.tiers, 999)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestScopeForMatchesMigration(t *testing.T) {
	if got := ScopeFor(uuid.Nil); got != ScopeDefault {
		t.Errorf("nil type = %q, want DEFAULT", got)
	}
	want := Scope("CT:" + siloam.String())
	if got := ScopeFor(siloam); got != want {
		t.Errorf("ScopeFor = %q, want %q — must match the generated column in 0007", got, want)
	}
}
