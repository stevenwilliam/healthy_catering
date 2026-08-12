package routing

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The matrix PROMPT §14 lists by name, plus the cases docs/01 §5.3 adds.

var (
	kA = uuid.MustParse("00000000-0000-7000-8000-0000000000a1")
	kB = uuid.MustParse("00000000-0000-7000-8000-0000000000a2")
	kC = uuid.MustParse("00000000-0000-7000-8000-0000000000a3")
)

// Real Jakarta coordinates, so the distances mean something.
var (
	monas     = Point{Lat: -6.1754, Lng: 106.8272} // central
	kemang    = Point{Lat: -6.2607, Lng: 106.8145} // ~9.5 km south
	kelapaGad = Point{Lat: -6.1588, Lng: 106.9056} // ~8.7 km north-east
	bogor     = Point{Lat: -6.5950, Lng: 106.8167} // ~47 km south, out of range
)

func kitchen(id uuid.UUID, code string, at Point, radiusKM float64) Kitchen {
	return Kitchen{
		ID: id, Code: code, Name: code, At: at, RadiusKM: radiusKM,
		Priority: 100, Active: true, ServesSlot: true, OpenOnDate: true,
		Unlimited: true,
	}
}

func req(to Point) Request {
	return Request{To: to, ServiceDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), QtyNeeded: 1}
}

func TestInsideOneRadius(t *testing.T) {
	got, err := Route(req(kemang), []Kitchen{kitchen(kA, "JKT-S", kemang, 10)})
	if err != nil {
		t.Fatal(err)
	}
	if got.KitchenID != kA {
		t.Errorf("kitchen = %s, want JKT-S", got.KitchenCode)
	}
	if got.Mode != "AUTO" {
		t.Errorf("mode = %s, want AUTO", got.Mode)
	}
	if got.Reason == "" {
		t.Error("staff will ask why this went here — the reason must not be empty")
	}
}

// Steven's rule: nearest kitchen wins when priorities are equal.
func TestTwoOverlappingRadiiPicksNearest(t *testing.T) {
	near := kitchen(kA, "NEAR", kemang, 30)
	far := kitchen(kB, "FAR", kelapaGad, 30)
	got, err := Route(req(kemang), []Kitchen{far, near}) // far listed first
	if err != nil {
		t.Fatal(err)
	}
	if got.KitchenID != kA {
		t.Errorf("kitchen = %s, want the nearer NEAR regardless of slice order", got.KitchenCode)
	}
	if got.DistanceM > 100 {
		t.Errorf("distance = %d m, want ~0 for a kitchen at the address", got.DistanceM)
	}
}

// Priority is the manual override for "3 km away but across the toll road".
func TestPriorityBeatsDistance(t *testing.T) {
	near := kitchen(kA, "NEAR", kemang, 30)
	preferred := kitchen(kB, "PREF", kelapaGad, 30)
	preferred.Priority = 10 // lower = preferred

	got, err := Route(req(kemang), []Kitchen{near, preferred})
	if err != nil {
		t.Fatal(err)
	}
	if got.KitchenID != kB {
		t.Errorf("kitchen = %s, want the priority kitchen PREF", got.KitchenCode)
	}
}

// A polygon exists precisely because the circle was wrong, so it overrides.
func TestPolygonOverridesRadius(t *testing.T) {
	// Inside the polygon but far outside the radius: still covered.
	k := kitchen(kA, "POLY", monas, 1)
	k.HasPolygon = true
	k.PolygonCovers = true
	if _, err := Route(req(bogor), []Kitchen{k}); err != nil {
		t.Errorf("polygon coverage must win over the radius: %v", err)
	}

	// Inside the radius but outside the polygon: NOT covered.
	k2 := kitchen(kB, "POLY2", kemang, 50)
	k2.HasPolygon = true
	k2.PolygonCovers = false
	if _, err := Route(req(kemang), []Kitchen{k2}); !errors.Is(err, ErrNotServiceable) {
		t.Errorf("err = %v, want ErrNotServiceable when the polygon excludes the point", err)
	}
}

func TestOutsideEverythingIsNotServiceable(t *testing.T) {
	_, err := Route(req(bogor), []Kitchen{kitchen(kA, "JKT-C", monas, 10)})
	if !errors.Is(err, ErrNotServiceable) {
		t.Fatalf("err = %v, want ErrNotServiceable", err)
	}
}

func TestCandidateAtCapacityIsDropped(t *testing.T) {
	full := kitchen(kA, "FULL", kemang, 30)
	full.Unlimited = false
	full.MaxPortions = 50
	full.ReservedPortions = 50
	spare := kitchen(kB, "SPARE", kelapaGad, 30)

	got, err := Route(req(kemang), []Kitchen{full, spare})
	if err != nil {
		t.Fatal(err)
	}
	if got.KitchenID != kB {
		t.Errorf("kitchen = %s, want the one with capacity", got.KitchenCode)
	}

	// And with no alternative, a full kitchen means not serviceable.
	if _, err := Route(req(kemang), []Kitchen{full}); !errors.Is(err, ErrNotServiceable) {
		t.Errorf("err = %v, want ErrNotServiceable when the only kitchen is full", err)
	}
}

// A ten-meal order must not fit into three remaining portions.
func TestCapacityRespectsQuantity(t *testing.T) {
	k := kitchen(kA, "TIGHT", kemang, 30)
	k.Unlimited = false
	k.MaxPortions = 50
	k.ReservedPortions = 47

	r := req(kemang)
	r.QtyNeeded = 3
	if _, err := Route(r, []Kitchen{k}); err != nil {
		t.Errorf("3 portions into 3 remaining must fit: %v", err)
	}
	r.QtyNeeded = 4
	if _, err := Route(r, []Kitchen{k}); !errors.Is(err, ErrNotServiceable) {
		t.Errorf("4 portions into 3 remaining must be refused, got %v", err)
	}
}

func TestInactiveClosedAndWrongSlotAreDropped(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Kitchen)
	}{
		{"inactive", func(k *Kitchen) { k.Active = false }},
		{"does not serve the slot", func(k *Kitchen) { k.ServesSlot = false }},
		{"closed that weekday", func(k *Kitchen) { k.OpenOnDate = false }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k := kitchen(kA, "K", kemang, 30)
			tc.mutate(&k)
			if _, err := Route(req(kemang), []Kitchen{k}); !errors.Is(err, ErrNotServiceable) {
				t.Errorf("err = %v, want ErrNotServiceable", err)
			}
		})
	}
}

// Determinism: identical candidates must resolve the same way every call, or
// staff get a different answer on a refresh.
func TestTieBreakIsDeterministic(t *testing.T) {
	a := kitchen(kA, "AAA", kemang, 30)
	b := kitchen(kB, "BBB", kemang, 30)
	c := kitchen(kC, "CCC", kemang, 30)

	first, err := Route(req(kemang), []Kitchen{a, b, c})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		got, err := Route(req(kemang), []Kitchen{c, a, b})
		if err != nil {
			t.Fatal(err)
		}
		if got.KitchenID != first.KitchenID {
			t.Fatalf("run %d picked %s, first picked %s — routing must be deterministic",
				i, got.KitchenCode, first.KitchenCode)
		}
	}
	if first.KitchenCode != "AAA" {
		t.Errorf("tie-break = %s, want the lowest code AAA", first.KitchenCode)
	}
}

func TestDistanceMeters(t *testing.T) {
	if d := DistanceMeters(monas, monas); d != 0 {
		t.Errorf("self-distance = %f, want 0", d)
	}
	// Monas → Kemang is about 9.5 km; allow a wide band since this asserts the
	// formula is right, not the map.
	d := DistanceMeters(monas, kemang)
	if d < 9_000 || d > 10_500 {
		t.Errorf("Monas→Kemang = %.0f m, want roughly 9500", d)
	}
	// Symmetry.
	if math.Abs(DistanceMeters(monas, kemang)-DistanceMeters(kemang, monas)) > 0.001 {
		t.Error("distance must be symmetric")
	}
}

func TestNearestEnrichesTheOutOfRangeLog(t *testing.T) {
	k, dist, ok := Nearest(bogor, []Kitchen{
		kitchen(kA, "JKT-C", monas, 5),
		kitchen(kB, "JKT-S", kemang, 5),
	})
	if !ok {
		t.Fatal("expected a nearest kitchen even when none covers")
	}
	if k.Code != "JKT-S" {
		t.Errorf("nearest = %s, want JKT-S", k.Code)
	}
	if dist < 30_000 {
		t.Errorf("distance = %d m, want a large distance to Bogor", dist)
	}
	if _, _, ok := Nearest(bogor, nil); ok {
		t.Error("no kitchens means no nearest")
	}
}

// docs/03 Q-11: outside the envelope is an INPUT error, which is a different
// message from "not serviceable yet".
func TestEnvelope(t *testing.T) {
	e := JabodetabekEnvelope
	if !e.Contains(monas) || !e.Contains(kemang) {
		t.Error("Jakarta points must be inside the envelope")
	}
	if e.Contains(Point{0, 0}) {
		t.Error("(0,0) must be rejected — it is the classic missing-pin value")
	}
	// A mis-signed latitude puts the customer in the northern hemisphere.
	if e.Contains(Point{Lat: 6.1754, Lng: 106.8272}) {
		t.Error("a mis-signed latitude must be rejected")
	}
	// Swapped lat/lng.
	if e.Contains(Point{Lat: 106.8272, Lng: -6.1754}) {
		t.Error("swapped coordinates must be rejected")
	}
}

func TestRemainingIsNeverNegative(t *testing.T) {
	k := kitchen(kA, "K", kemang, 10)
	k.Unlimited = false
	k.MaxPortions = 10
	k.ReservedPortions = 15 // should not happen, but must not underflow the sort
	if got := k.Remaining(); got != 0 {
		t.Errorf("remaining = %d, want 0", got)
	}
}
