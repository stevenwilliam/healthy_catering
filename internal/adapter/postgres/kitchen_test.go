package postgres

import (
	"testing"

	"github.com/google/uuid"
)

// The Overview join returns a kitchen once per active slot, with the capacity
// columns NULL where that slot is not open on the date. Folding it back is
// where the one distinction that matters can quietly be lost:
//
//	no capacity row  ->  the slot is NOT OPEN on this date  ->  "closed"
//	quota 0          ->  the slot IS open and has no room   ->  "0 / 0", full
//
// Collapse them and the dashboard shows a kitchen as shut when it is merely
// unconfigured, or as open with nowhere to put an order. Neither is visible by
// looking at the screen, which is why it is asserted here.

func ptr[T any](v T) *T { return &v }

func TestFoldKitchenRowsSeparatesUnopenedFromFull(t *testing.T) {
	k := uuid.MustParse("00000000-0000-7000-8000-000000000001")
	open := uuid.MustParse("00000000-0000-7000-8000-000000000601")
	shut := uuid.MustParse("00000000-0000-7000-8000-000000000602")
	full := uuid.MustParse("00000000-0000-7000-8000-000000000603")

	base := kitchenRow{ID: k, Code: "KTC-01", Name: "Tebet", IsActive: true}

	rows := []kitchenRow{
		// Open, with room.
		func() kitchenRow {
			r := base
			r.SlotID, r.SlotTime = &open, ptr("11:30")
			r.Quota, r.Used = ptr(40), ptr(18)
			return r
		}(),
		// Not opened on this date: no capacity row, so both are NULL.
		func() kitchenRow {
			r := base
			r.SlotID, r.SlotTime = &shut, ptr("18:00")
			return r
		}(),
		// Open and full. Quota zero is still OPEN.
		func() kitchenRow {
			r := base
			r.SlotID, r.SlotTime = &full, ptr("12:00")
			r.Quota, r.Used = ptr(0), ptr(0)
			return r
		}(),
	}

	got := foldKitchenRows(rows)
	if len(got) != 1 {
		t.Fatalf("want one kitchen, got %d", len(got))
	}
	if len(got[0].Slots) != 3 {
		t.Fatalf("want three slots, got %d", len(got[0].Slots))
	}

	for _, c := range []struct {
		name          string
		i             int
		wantAvailable bool
		wantQuota     int
		wantUsed      int
	}{
		{"open with room", 0, true, 40, 18},
		{"no capacity row is NOT open", 1, false, 0, 0},
		{"quota zero is open and full", 2, true, 0, 0},
	} {
		s := got[0].Slots[c.i]
		if s.Available != c.wantAvailable {
			t.Errorf("%s: Available = %v, want %v", c.name, s.Available, c.wantAvailable)
		}
		if s.Quota != c.wantQuota || s.Used != c.wantUsed {
			t.Errorf("%s: quota/used = %d/%d, want %d/%d",
				c.name, s.Quota, s.Used, c.wantQuota, c.wantUsed)
		}
	}
}

// A half-NULL pair cannot come out of the join as written, but treating it as
// open would invent a quota out of nothing — so it is refused rather than
// half-read.
func TestFoldKitchenRowsRefusesAHalfNullCapacityPair(t *testing.T) {
	k := uuid.MustParse("00000000-0000-7000-8000-000000000001")
	slot := uuid.MustParse("00000000-0000-7000-8000-000000000601")

	for _, c := range []struct {
		name        string
		quota, used *int
	}{
		{"quota without used", ptr(40), nil},
		{"used without quota", nil, ptr(18)},
	} {
		got := foldKitchenRows([]kitchenRow{{
			ID: k, Code: "KTC-01", SlotID: &slot, Quota: c.quota, Used: c.used,
		}})
		if got[0].Slots[0].Available {
			t.Errorf("%s: reported available", c.name)
		}
	}
}

// A kitchen serving no active slot is still a kitchen: it must appear on the
// coverage screen so somebody can see that it has none, rather than vanishing
// from the list entirely.
func TestFoldKitchenRowsKeepsAKitchenWithNoSlots(t *testing.T) {
	k := uuid.MustParse("00000000-0000-7000-8000-000000000009")
	got := foldKitchenRows([]kitchenRow{{ID: k, Code: "NEW-01", Name: "Unopened"}})

	if len(got) != 1 {
		t.Fatalf("want one kitchen, got %d", len(got))
	}
	if got[0].Slots == nil {
		t.Error("Slots is nil; a JSON null where the client expects an array is a frontend crash")
	}
	if len(got[0].Slots) != 0 {
		t.Errorf("want no slots, got %d", len(got[0].Slots))
	}
}

// Row order carries the ORDER BY (priority, code, slot). Folding must preserve
// it, or the dashboard's kitchen rows and slot columns reshuffle per request.
func TestFoldKitchenRowsPreservesOrder(t *testing.T) {
	a := uuid.MustParse("00000000-0000-7000-8000-00000000000a")
	b := uuid.MustParse("00000000-0000-7000-8000-00000000000b")
	s1 := uuid.MustParse("00000000-0000-7000-8000-000000000601")
	s2 := uuid.MustParse("00000000-0000-7000-8000-000000000602")

	got := foldKitchenRows([]kitchenRow{
		{ID: a, Code: "AAA", SlotID: &s1, SlotTime: ptr("07:00")},
		{ID: a, Code: "AAA", SlotID: &s2, SlotTime: ptr("11:30")},
		{ID: b, Code: "BBB", SlotID: &s1, SlotTime: ptr("07:00")},
		// Interleaved: the query cannot produce this, but the fold must not
		// depend on that to keep A and B apart.
		{ID: a, Code: "AAA", SlotID: &s1, SlotTime: ptr("18:00")},
	})

	if len(got) != 2 || got[0].Code != "AAA" || got[1].Code != "BBB" {
		t.Fatalf("kitchen order lost: %+v", got)
	}
	want := []string{"07:00", "11:30", "18:00"}
	for i, w := range want {
		if got[0].Slots[i].SlotTime != w {
			t.Errorf("slot %d = %q, want %q", i, got[0].Slots[i].SlotTime, w)
		}
	}
}
