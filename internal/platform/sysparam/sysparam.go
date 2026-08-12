// Package sysparam reads and caches the sys_parameters table.
//
// Everything the business might change without a deploy lives there
// (CLAUDE.md §7). This package is the only way the rest of the service reads
// one, so a parameter is never re-parsed inconsistently in two places.
//
// The cache is in-process with a short TTL and an explicit invalidation on
// write. A stale cut-off time for a few seconds is harmless; a stale one for an
// hour is a support call, so the TTL is short and the write path clears it.
package sysparam

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

// Well-known keys. Typed constants rather than string literals at call sites,
// so a rename is a compile error rather than a silent default.
const (
	KeyTaxRateBps           = "tax.rate_bps"
	KeyTaxInclusive         = "tax.inclusive"
	KeyCompanyLegalName     = "company.legal_name"
	KeyCompanyNPWP          = "company.npwp"
	KeyCompanyPhone         = "company.phone"
	KeyCompanyEmail         = "company.email"
	KeyCompanyWhatsApp      = "company.whatsapp"
	KeyCutoffTime           = "order.cutoff_time"
	KeyCutoffLeadDays       = "order.cutoff_lead_days"
	KeyMaxQtyPerLine        = "order.max_qty_per_line"
	KeyPaymentWindow        = "order.payment_window"
	KeyUniqueCodeEnabled    = "order.unique_code_enabled"
	KeyDeliveryFeeBands     = "delivery.fee_bands"
	KeyDeliveryFreeAbove    = "delivery.free_above_idr"
	KeyPackageDeliveryFree  = "delivery.package_included"
	KeyGeoEnvelope          = "geo.envelope"
	KeyPlacesFallbackBlock  = "geo.places_fallback_blocked"
	KeyCreditLowThreshold   = "credit.low_threshold"
	KeyExpiryWarningDays    = "credit.expiry_warning_days"
	KeyPackageCapacityPct   = "credit.package_capacity_reserve_pct"
	KeyPublishHorizonDays   = "schedule.publish_horizon_days"
	KeyEmailEnabled         = "notify.email_enabled"
	KeyWhatsAppEnabled      = "notify.whatsapp_enabled"
	KeyReminderHour         = "notify.reminder_hour"
	KeyMaxLoginAttempts     = "security.max_login_attempts"
	KeyLockoutDuration      = "security.lockout_duration"
	KeyRequireEmailVerified = "security.require_email_verification"
)

// Param is one row.
type Param struct {
	Key         string
	Value       string
	ValueType   string
	Group       string
	Label       string
	Description string
	IsSecret    bool
	IsSystem    bool
	SortOrder   int
	UpdatedAt   time.Time
}

// Masked returns the value safe to show or log. A secret-flagged parameter is
// never rendered, in the UI or in a log line (99 §8).
func (p Param) Masked() string {
	if p.IsSecret && p.Value != "" {
		return "••••••••"
	}
	return p.Value
}

// Store reads parameters, with a short-TTL cache.
type Store struct {
	db  *sql.DB
	ttl time.Duration

	mu       sync.RWMutex
	cache    map[string]Param
	loadedAt time.Time
}

// NewStore returns a store. A ttl of zero disables caching, which is what the
// tests use so a write is visible immediately.
func NewStore(db *sql.DB, ttl time.Duration) *Store {
	return &Store{db: db, ttl: ttl, cache: map[string]Param{}}
}

// Invalidate clears the cache. Called by the write path so a settings change
// takes effect on the next read rather than at the end of the TTL.
func (s *Store) Invalidate() {
	s.mu.Lock()
	s.cache = map[string]Param{}
	s.loadedAt = time.Time{}
	s.mu.Unlock()
}

// All returns every parameter, refreshing the cache if it is stale.
func (s *Store) All(ctx context.Context) (map[string]Param, error) {
	s.mu.RLock()
	fresh := s.ttl > 0 && !s.loadedAt.IsZero() && time.Since(s.loadedAt) < s.ttl
	if fresh {
		out := make(map[string]Param, len(s.cache))
		for k, v := range s.cache {
			out[k] = v
		}
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT key, value, value_type, param_group, label, description,
		       is_secret, is_system, sort_order, updated_at
		  FROM sys_parameters`)
	if err != nil {
		return nil, fmt.Errorf("sysparam: load: %w", err)
	}
	defer rows.Close()

	loaded := map[string]Param{}
	for rows.Next() {
		var p Param
		if err := rows.Scan(&p.Key, &p.Value, &p.ValueType, &p.Group, &p.Label,
			&p.Description, &p.IsSecret, &p.IsSystem, &p.SortOrder, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("sysparam: scan: %w", err)
		}
		loaded[p.Key] = p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sysparam: rows: %w", err)
	}

	s.mu.Lock()
	s.cache, s.loadedAt = loaded, time.Now()
	s.mu.Unlock()

	out := make(map[string]Param, len(loaded))
	for k, v := range loaded {
		out[k] = v
	}
	return out, nil
}

// Get returns one parameter.
func (s *Store) Get(ctx context.Context, key string) (Param, error) {
	all, err := s.All(ctx)
	if err != nil {
		return Param{}, err
	}
	p, ok := all[key]
	if !ok {
		return Param{}, fmt.Errorf("sysparam: %q is not configured", key)
	}
	return p, nil
}

// String returns a string parameter, or def when it is missing or empty.
//
// A missing parameter falls back rather than failing: an operator who deletes a
// row should get the documented default, not a 500 on every request. The
// fallback is logged by the caller when it matters.
func (s *Store) String(ctx context.Context, key, def string) string {
	p, err := s.Get(ctx, key)
	if err != nil || p.Value == "" {
		return def
	}
	return p.Value
}

// Int returns an integer parameter.
func (s *Store) Int(ctx context.Context, key string, def int) int {
	p, err := s.Get(ctx, key)
	if err != nil {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(p.Value))
	if err != nil {
		return def
	}
	return n
}

// Money returns a whole-rupiah parameter.
func (s *Store) Money(ctx context.Context, key string, def money.IDR) money.IDR {
	return money.IDR(s.Int(ctx, key, int(def)))
}

// Bool returns a boolean parameter.
func (s *Store) Bool(ctx context.Context, key string, def bool) bool {
	p, err := s.Get(ctx, key)
	if err != nil {
		return def
	}
	b, err := strconv.ParseBool(strings.TrimSpace(p.Value))
	if err != nil {
		return def
	}
	return b
}

// Duration returns a duration parameter, e.g. "2h" or "15m".
func (s *Store) Duration(ctx context.Context, key string, def time.Duration) time.Duration {
	p, err := s.Get(ctx, key)
	if err != nil {
		return def
	}
	d, err := time.ParseDuration(strings.TrimSpace(p.Value))
	if err != nil {
		return def
	}
	return d
}

// TimeOfDay returns a wall-clock parameter such as "18:00" as a duration since
// midnight, which is what the cut-off rule takes.
func (s *Store) TimeOfDay(ctx context.Context, key string, def time.Duration) time.Duration {
	p, err := s.Get(ctx, key)
	if err != nil {
		return def
	}
	parts := strings.Split(strings.TrimSpace(p.Value), ":")
	if len(parts) < 2 {
		return def
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return def
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
}

// JSON unmarshals a JSON parameter into v.
func (s *Store) JSON(ctx context.Context, key string, v any) error {
	p, err := s.Get(ctx, key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(p.Value), v); err != nil {
		return fmt.Errorf("sysparam: %q is not valid JSON: %w", key, err)
	}
	return nil
}

// Set writes a parameter and invalidates the cache. The caller writes the audit
// row — this package does not know who the actor is.
func (s *Store) Set(ctx context.Context, key, value string, updatedBy any) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sys_parameters SET value = $1, updated_by = $2 WHERE key = $3`,
		value, updatedBy, key)
	if err != nil {
		return fmt.Errorf("sysparam: set %q: %w", key, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sysparam: set %q: %w", key, err)
	}
	if n == 0 {
		return fmt.Errorf("sysparam: %q does not exist", key)
	}
	s.Invalidate()
	return nil
}
