// Package schedule holds the cut-off and business-date rules.
//
// Orders for delivery date D close at 18:00 WIB on D-1 (PROMPT §6). Both the
// time and the lead-day count are sys_parameters, and both can be overridden
// per slot, so tuning dinner later is a settings change rather than a migration
// (docs/03 Q-5).
//
// Everything here converts through Asia/Jakarta EXPLICITLY. The server's local
// time is never consulted — a rule the house style states twice (CLAUDE.md §4)
// because it is the classic way a scheduling system goes wrong in production
// after working perfectly in development.
package schedule

import (
	"errors"
	"fmt"
	"time"
)

// Errors a caller turns into a customer-facing message.
var (
	ErrPastCutoff   = errors.New("schedule: the cut-off for this date has passed")
	ErrPastDate     = errors.New("schedule: the service date is in the past")
	ErrNotPublished = errors.New("schedule: the menu for this date is not published")
)

// Rule is the cut-off configuration for one slot.
type Rule struct {
	// CutoffTime is the wall-clock time in the operating timezone, e.g. 18:00.
	CutoffTime time.Duration
	// LeadDays is how many days BEFORE the service date the cut-off falls.
	LeadDays int
	Location *time.Location
}

// DefaultRule is the seeded rule: 18:00 on D-1, Asia/Jakarta.
func DefaultRule(loc *time.Location) Rule {
	return Rule{CutoffTime: 18 * time.Hour, LeadDays: 1, Location: loc}
}

// CutoffFor returns the instant at which ordering for serviceDate closes.
//
// serviceDate is a business calendar date. The result is an absolute instant,
// so comparing it against time.Now() is correct wherever the server runs.
func (r Rule) CutoffFor(serviceDate time.Time) time.Time {
	loc := r.Location
	if loc == nil {
		loc = time.UTC
	}
	y, m, d := serviceDate.Date()
	midnight := time.Date(y, m, d-r.LeadDays, 0, 0, 0, 0, loc)
	return midnight.Add(r.CutoffTime)
}

// IsOpen reports whether ordering for serviceDate is still open at `now`.
func (r Rule) IsOpen(serviceDate, now time.Time) bool {
	return now.Before(r.CutoffFor(serviceDate))
}

// TimeUntilCutoff drives the live countdown on the menu page (PROMPT §6). It
// is computed server-side; the browser's clock is never trusted for the
// decision, only for animating between polls.
func (r Rule) TimeUntilCutoff(serviceDate, now time.Time) time.Duration {
	d := r.CutoffFor(serviceDate).Sub(now)
	if d < 0 {
		return 0
	}
	return d
}

// Check validates that a service date can still be ordered for.
func (r Rule) Check(serviceDate, now time.Time) error {
	loc := r.Location
	if loc == nil {
		loc = time.UTC
	}
	today := Today(now, loc)
	if serviceDate.Before(today) {
		return fmt.Errorf("%w: %s", ErrPastDate, serviceDate.Format("2006-01-02"))
	}
	if !r.IsOpen(serviceDate, now) {
		return fmt.Errorf("%w: closed at %s", ErrPastCutoff,
			r.CutoffFor(serviceDate).In(loc).Format("2006-01-02 15:04 MST"))
	}
	return nil
}

// Today is the business date at an instant, in the operating timezone. A UTC
// server at 18:30 UTC is already tomorrow in Jakarta, and every list of
// "today's deliveries" depends on getting that right.
func Today(now time.Time, loc *time.Location) time.Time {
	l := now.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

// PaymentDeadline returns when an unpaid order expires.
//
// docs/02 D-13: the window is capped at the cut-off. A flat two hours applied
// at 17:30 would expire at 19:30 — ninety minutes after the 18:00 cut-off the
// order was placed against, having held capacity through the entire window in
// which somebody else could have bought it.
func PaymentDeadline(placedAt time.Time, window time.Duration, earliestCutoff time.Time) time.Time {
	deadline := placedAt.Add(window)
	if !earliestCutoff.IsZero() && earliestCutoff.Before(deadline) {
		return earliestCutoff
	}
	return deadline
}

// EarliestCutoff returns the soonest cut-off across a cart's service dates,
// which is the one that caps the payment window.
func EarliestCutoff(rules map[string]Rule, dates map[string]time.Time) time.Time {
	var earliest time.Time
	for key, d := range dates {
		r, ok := rules[key]
		if !ok {
			continue
		}
		c := r.CutoffFor(d)
		if earliest.IsZero() || c.Before(earliest) {
			earliest = c
		}
	}
	return earliest
}

// PublishHorizon reports how many days ahead the menu is published, and whether
// that is under the operational target (docs/03 Q-17). Package customers cannot
// book what does not exist, so this is a dashboard warning rather than an error.
func PublishHorizon(lastPublished, now time.Time, loc *time.Location, targetDays int) (days int, healthy bool) {
	if lastPublished.IsZero() {
		return 0, false
	}
	today := Today(now, loc)
	days = int(lastPublished.Sub(today).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return days, days >= targetDays
}
