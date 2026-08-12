// Package postgres holds the repositories. gorm is the ORM, but money, capacity
// and geography paths use raw SQL with placeholders (CLAUDE.md §3).
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/routing"
)

// KitchenRepo loads routing candidates.
type KitchenRepo struct{ db *gorm.DB }

func NewKitchenRepo(db *gorm.DB) *KitchenRepo { return &KitchenRepo{db: db} }

// Candidates returns the kitchens that could serve a point on a date and slot,
// with the PostGIS coverage tests already evaluated.
//
// The coverage predicates run in the database because they are index-backed
// (GIST on geom and on service_area); the ranking runs in the domain package
// because it is a business rule that has to be unit-testable without a
// database. ST_Covers is evaluated per row and handed back as a boolean rather
// than used to filter, so the domain can see that a polygon EXCLUDED a kitchen
// and say so, instead of the row silently vanishing.
// A nil slotID means "any active slot", which is what the homepage widget asks:
// a visitor typing their address has not chosen lunch or dinner yet, and
// answering "we do not deliver to you" because of that would be wrong.
func (r *KitchenRepo) Candidates(ctx context.Context, at routing.Point, slotID *uuid.UUID, on time.Time) ([]routing.Kitchen, error) {
	// ISO weekday: Postgres isodow gives 1=Monday..7=Sunday, matching
	// kitchen_operating_day.weekday.
	// Positional placeholders, in text order:
	//   lng, lat, slot, slot, date, date, slot.
	// Named parameters are not used here — gorm does not substitute them inside
	// a CTE, which fails at runtime rather than at compile time.
	const q = `
WITH p AS (
  SELECT ST_SetSRID(ST_MakePoint(?::float8, ?::float8), 4326)::geography AS geom
)
SELECT k.id, k.code, k.name, k.latitude::float8, k.longitude::float8,
       k.service_radius_km::float8,
       (k.service_area IS NOT NULL)                        AS has_polygon,
       COALESCE(ST_Covers(k.service_area, p.geom), FALSE)   AS polygon_covers,
       k.priority, k.is_active,
       EXISTS (SELECT 1 FROM kitchen_slot ks
                 JOIN delivery_time_slot s ON s.id = ks.slot_id AND s.is_active
                WHERE ks.kitchen_id = k.id
                  AND (?::uuid IS NULL OR ks.slot_id = ?::uuid))            AS serves_slot,
       EXISTS (SELECT 1 FROM kitchen_operating_day o
                WHERE o.kitchen_id = k.id
                  AND o.weekday = EXTRACT(ISODOW FROM ?::date)::int)        AS open_on_date,
       COALESCE(c.max_portions, k.default_slot_capacity)    AS max_portions,
       COALESCE(c.reserved_portions, 0)                     AS reserved_portions,
       (COALESCE(c.max_portions, k.default_slot_capacity) IS NULL) AS unlimited
  FROM kitchen k
  CROSS JOIN p
  LEFT JOIN kitchen_capacity c
         ON c.kitchen_id = k.id AND c.service_date = ?::date AND c.slot_id = ?
 WHERE k.is_active`

	type row struct {
		ID               uuid.UUID
		Code             string
		Name             string
		Latitude         float64
		Longitude        float64
		ServiceRadiusKm  float64
		HasPolygon       bool
		PolygonCovers    bool
		Priority         int
		IsActive         bool
		ServesSlot       bool
		OpenOnDate       bool
		MaxPortions      *int
		ReservedPortions int
		Unlimited        bool
	}

	date := on.Format("2006-01-02")
	var rows []row
	if err := r.db.WithContext(ctx).Raw(q,
		at.Lng, at.Lat, slotID, slotID, date, date, slotID,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: kitchen candidates: %w", err)
	}

	out := make([]routing.Kitchen, 0, len(rows))
	for _, k := range rows {
		kk := routing.Kitchen{
			ID: k.ID, Code: k.Code, Name: k.Name,
			At:               routing.Point{Lat: k.Latitude, Lng: k.Longitude},
			RadiusKM:         k.ServiceRadiusKm,
			HasPolygon:       k.HasPolygon,
			PolygonCovers:    k.PolygonCovers,
			Priority:         k.Priority,
			Active:           k.IsActive,
			ServesSlot:       k.ServesSlot,
			OpenOnDate:       k.OpenOnDate,
			Unlimited:        k.Unlimited,
			ReservedPortions: k.ReservedPortions,
		}
		if k.MaxPortions != nil {
			kk.MaxPortions = *k.MaxPortions
		}
		out = append(out, kk)
	}
	return out, nil
}

// LogOutOfRange records demand we could not serve. This log is the map of where
// to open the next kitchen (PROMPT §9.3.5), so it is written even for anonymous
// widget hits.
func (r *KitchenRepo) LogOutOfRange(ctx context.Context, at routing.Point, district, city string,
	slotID *uuid.UUID, on *time.Time, source string, customerID *uuid.UUID,
	nearestID *uuid.UUID, nearestM *int) error {

	return r.db.WithContext(ctx).Exec(`
		INSERT INTO out_of_range_attempt
		  (id, customer_id, latitude, longitude, district, city, slot_id, service_date,
		   source, nearest_kitchen_id, nearest_distance_m)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		uuid.New(), customerID, at.Lat, at.Lng, district, city, slotID, on,
		source, nearestID, nearestM).Error
}

// ActiveSlots returns the customer-facing delivery slots.
func (r *KitchenRepo) ActiveSlots(ctx context.Context) ([]Slot, error) {
	var out []Slot
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, slot_time::text AS slot_time, alias, sort_order,
		       cutoff_time::text AS cutoff_time, cutoff_lead_days
		  FROM delivery_time_slot
		 WHERE is_active
		 ORDER BY sort_order, slot_time`).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: active slots: %w", err)
	}
	return out, nil
}

// Slot is a delivery time slot. Customers see the Alias only; SlotTime is
// internal (PROMPT §8.1).
type Slot struct {
	ID             uuid.UUID `json:"id"`
	SlotTime       string    `json:"-"`
	Alias          string    `json:"alias"`
	SortOrder      int       `json:"-"`
	CutoffTime     *string   `json:"-"`
	CutoffLeadDays *int      `json:"-"`
}
