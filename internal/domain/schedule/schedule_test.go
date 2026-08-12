package schedule

import (
	"errors"
	"testing"
	"time"
)

func jakarta(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	return loc
}

func TestCutoffIsEighteenHundredTheDayBefore(t *testing.T) {
	loc := jakarta(t)
	r := DefaultRule(loc)
	service := time.Date(2026, 9, 2, 0, 0, 0, 0, loc)

	got := r.CutoffFor(service).In(loc)
	want := time.Date(2026, 9, 1, 18, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("cut-off = %v, want %v", got, want)
	}
}

// The rule that matters: the decision must be identical whatever the server's
// local timezone is (CLAUDE.md §4).
func TestCutoffIndependentOfServerTimezone(t *testing.T) {
	loc := jakarta(t)
	r := DefaultRule(loc)
	service := time.Date(2026, 9, 2, 0, 0, 0, 0, loc)
	cutoff := r.CutoffFor(service)

	// 17:59 WIB is 10:59 UTC — open.
	openAt := time.Date(2026, 9, 1, 10, 59, 0, 0, time.UTC)
	// 18:01 WIB is 11:01 UTC — closed.
	closedAt := time.Date(2026, 9, 1, 11, 1, 0, 0, time.UTC)

	if !r.IsOpen(service, openAt) {
		t.Error("17:59 WIB must still be open")
	}
	if r.IsOpen(service, closedAt) {
		t.Error("18:01 WIB must be closed")
	}
	// And expressed in a third timezone, the same instants decide the same way.
	ny, err := time.LoadLocation("America/New_York")
	if err == nil {
		if !r.IsOpen(service, openAt.In(ny)) || r.IsOpen(service, closedAt.In(ny)) {
			t.Error("the decision changed when the instant was rendered in another zone")
		}
	}
	if cutoff.In(loc).Hour() != 18 {
		t.Errorf("cut-off hour in WIB = %d, want 18", cutoff.In(loc).Hour())
	}
}

func TestCheckRejectsPastDatesAndClosedCutoffs(t *testing.T) {
	loc := jakarta(t)
	r := DefaultRule(loc)
	now := time.Date(2026, 9, 1, 19, 0, 0, 0, loc) // after the 18:00 cut-off

	if err := r.Check(time.Date(2026, 8, 30, 0, 0, 0, 0, loc), now); !errors.Is(err, ErrPastDate) {
		t.Errorf("err = %v, want ErrPastDate", err)
	}
	if err := r.Check(time.Date(2026, 9, 2, 0, 0, 0, 0, loc), now); !errors.Is(err, ErrPastCutoff) {
		t.Errorf("err = %v, want ErrPastCutoff", err)
	}
	// The day after tomorrow is still open.
	if err := r.Check(time.Date(2026, 9, 3, 0, 0, 0, 0, loc), now); err != nil {
		t.Errorf("D+2 must still be open: %v", err)
	}
}

func TestPerSlotOverride(t *testing.T) {
	loc := jakarta(t)
	// docs/03 Q-5: dinner may want a different lead time from lunch.
	dinner := Rule{CutoffTime: 12 * time.Hour, LeadDays: 0, Location: loc}
	service := time.Date(2026, 9, 2, 0, 0, 0, 0, loc)

	got := dinner.CutoffFor(service).In(loc)
	want := time.Date(2026, 9, 2, 12, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("same-day dinner cut-off = %v, want %v", got, want)
	}
}

func TestTimeUntilCutoffNeverNegative(t *testing.T) {
	loc := jakarta(t)
	r := DefaultRule(loc)
	service := time.Date(2026, 9, 2, 0, 0, 0, 0, loc)

	before := time.Date(2026, 9, 1, 14, 48, 0, 0, loc)
	if got := r.TimeUntilCutoff(service, before); got != 3*time.Hour+12*time.Minute {
		t.Errorf("countdown = %v, want 3h12m", got)
	}
	after := time.Date(2026, 9, 1, 20, 0, 0, 0, loc)
	if got := r.TimeUntilCutoff(service, after); got != 0 {
		t.Errorf("countdown after cut-off = %v, want 0", got)
	}
}

// D-13: the payment window is capped at the cut-off, or an order placed at
// 17:30 holds capacity until 19:30 — after the cut-off it was placed against.
func TestPaymentDeadlineCappedAtCutoff(t *testing.T) {
	loc := jakarta(t)
	r := DefaultRule(loc)
	service := time.Date(2026, 9, 2, 0, 0, 0, 0, loc)
	cutoff := r.CutoffFor(service)

	placed := time.Date(2026, 9, 1, 17, 30, 0, 0, loc)
	got := PaymentDeadline(placed, 2*time.Hour, cutoff)
	if !got.Equal(cutoff) {
		t.Errorf("deadline = %v, want it capped at the cut-off %v", got.In(loc), cutoff.In(loc))
	}

	// Placed early, the full window applies.
	early := time.Date(2026, 9, 1, 9, 0, 0, 0, loc)
	if got := PaymentDeadline(early, 2*time.Hour, cutoff); !got.Equal(early.Add(2 * time.Hour)) {
		t.Errorf("deadline = %v, want the full 2h window", got.In(loc))
	}

	// With no cut-off in play the window stands.
	if got := PaymentDeadline(early, 2*time.Hour, time.Time{}); !got.Equal(early.Add(2 * time.Hour)) {
		t.Errorf("deadline = %v, want the full window", got.In(loc))
	}
}

func TestEarliestCutoffAcrossACart(t *testing.T) {
	loc := jakarta(t)
	rules := map[string]Rule{"lunch": DefaultRule(loc), "dinner": DefaultRule(loc)}
	dates := map[string]time.Time{
		"lunch":  time.Date(2026, 9, 5, 0, 0, 0, 0, loc),
		"dinner": time.Date(2026, 9, 2, 0, 0, 0, 0, loc), // sooner
	}
	got := EarliestCutoff(rules, dates).In(loc)
	want := time.Date(2026, 9, 1, 18, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("earliest = %v, want %v — a multi-date cart is capped by its soonest date", got, want)
	}
}

func TestTodayIsTheBusinessDate(t *testing.T) {
	loc := jakarta(t)
	// 18:30 UTC is already the next day in Jakarta (UTC+7).
	instant := time.Date(2026, 9, 1, 18, 30, 0, 0, time.UTC)
	got := Today(instant, loc)
	if got.Format("2006-01-02") != "2026-09-02" {
		t.Errorf("business date = %s, want 2026-09-02 — the server's UTC day is not the business day",
			got.Format("2006-01-02"))
	}
}

func TestPublishHorizon(t *testing.T) {
	loc := jakarta(t)
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, loc)

	days, healthy := PublishHorizon(time.Date(2026, 9, 10, 0, 0, 0, 0, loc), now, loc, 7)
	if days != 9 || !healthy {
		t.Errorf("got %d days healthy=%v, want 9 and healthy", days, healthy)
	}
	days, healthy = PublishHorizon(time.Date(2026, 9, 3, 0, 0, 0, 0, loc), now, loc, 7)
	if days != 2 || healthy {
		t.Errorf("got %d days healthy=%v, want 2 and unhealthy", days, healthy)
	}
	if _, healthy := PublishHorizon(time.Time{}, now, loc, 7); healthy {
		t.Error("an unpublished calendar is not healthy")
	}
}
