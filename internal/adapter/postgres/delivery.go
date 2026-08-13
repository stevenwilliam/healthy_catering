package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeliveryRepo owns the fulfilment side of a delivery.
type DeliveryRepo struct{ db *gorm.DB }

func NewDeliveryRepo(db *gorm.DB) *DeliveryRepo { return &DeliveryRepo{db: db} }

// DeliveryRow is one delivery in a staff list.
type DeliveryRow struct {
	ID           uuid.UUID `json:"id"`
	DeliveryCode string    `json:"delivery_code"`
	ServiceDate  string    `json:"service_date"`
	Slot         string    `json:"slot"`
	Status       string    `json:"status"`
	Kitchen      string    `json:"kitchen"`
	CustomerName string    `json:"customer_name"`
	Phone        string    `json:"phone"`
	AddressLine  string    `json:"address_line"`
	District     string    `json:"district"`
	Meals        int       `json:"meals"`
	DistanceM    int       `json:"distance_m"`
	Mode         string    `json:"assignment_mode"`
	Reason       string    `json:"assignment_reason"`
}

// DeliveryQuery narrows a list.
type DeliveryQuery struct {
	Params    ListParams
	From      *time.Time
	To        *time.Time
	KitchenID *uuid.UUID
	Status    string
}

// List returns a searchable page of deliveries.
func (r *DeliveryRepo) List(ctx context.Context, q DeliveryQuery) (Page[DeliveryRow], error) {
	p := q.Params.Normalise("service_date", "service_date", "status")

	db := r.db.WithContext(ctx).Table("delivery d").
		Joins("JOIN delivery_time_slot s ON s.id = d.slot_id").
		Joins("LEFT JOIN kitchen k ON k.id = d.kitchen_id")

	if q.From != nil {
		db = db.Where("d.service_date >= ?::date", q.From.Format("2006-01-02"))
	}
	if q.To != nil {
		db = db.Where("d.service_date <= ?::date", q.To.Format("2006-01-02"))
	}
	if q.KitchenID != nil {
		db = db.Where("d.kitchen_id = ?", *q.KitchenID)
	}
	if q.Status != "" {
		db = db.Where("d.status = ?", q.Status)
	}
	if p.Q != "" {
		// Staff search a delivery by the customer's name or the code on the
		// packing label, not by an id they have never seen.
		pattern := SearchPattern(p.Q)
		db = db.Where(`lower(d.delivery_code) LIKE ?
			OR lower(d.address_snapshot->>'recipient_name') LIKE ?
			OR lower(d.address_snapshot->>'address_line') LIKE ?
			OR d.address_snapshot->>'recipient_phone' LIKE ?`,
			pattern, pattern, pattern, pattern)
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[DeliveryRow]{}, fmt.Errorf("postgres: count deliveries: %w", err)
	}

	var items []DeliveryRow
	err := db.Session(&gorm.Session{}).
		Select(`d.id, d.delivery_code, d.service_date::text AS service_date,
			s.alias AS slot, d.status, COALESCE(k.name,'unassigned') AS kitchen,
			d.address_snapshot->>'recipient_name' AS customer_name,
			d.address_snapshot->>'recipient_phone' AS phone,
			d.address_snapshot->>'address_line' AS address_line,
			COALESCE(d.address_snapshot->>'district','') AS district,
			(SELECT COALESCE(SUM(dl.qty),0) FROM delivery_line dl
			  WHERE dl.delivery_id = d.id) AS meals,
			COALESCE(d.assigned_distance_m,0) AS distance_m,
			d.assignment_mode AS mode, d.assignment_reason AS reason`).
		Order("d." + p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[DeliveryRow]{}, fmt.Errorf("postgres: list deliveries: %w", err)
	}
	return NewPage(items, total, p), nil
}

// ListForCustomer returns one customer's deliveries, scoped by owner in the
// query so a handler cannot forget it.
func (r *DeliveryRepo) ListForCustomer(ctx context.Context, customerID uuid.UUID,
	p ListParams) (Page[DeliveryRow], error) {

	p = p.Normalise("service_date", "service_date", "status")
	db := r.db.WithContext(ctx).Table("delivery d").
		Joins("JOIN delivery_time_slot s ON s.id = d.slot_id").
		Joins("LEFT JOIN kitchen k ON k.id = d.kitchen_id").
		Where("d.customer_id = ?", customerID)
	if p.Q != "" {
		pattern := SearchPattern(p.Q)
		db = db.Where("lower(d.delivery_code) LIKE ? OR lower(d.status) LIKE ?", pattern, pattern)
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[DeliveryRow]{}, fmt.Errorf("postgres: count my deliveries: %w", err)
	}
	var items []DeliveryRow
	err := db.Session(&gorm.Session{}).
		Select(`d.id, d.delivery_code, d.service_date::text AS service_date,
			s.alias AS slot, d.status, COALESCE(k.name,'') AS kitchen,
			d.address_snapshot->>'recipient_name' AS customer_name,
			d.address_snapshot->>'recipient_phone' AS phone,
			d.address_snapshot->>'address_line' AS address_line,
			COALESCE(d.address_snapshot->>'district','') AS district,
			(SELECT COALESCE(SUM(dl.qty),0) FROM delivery_line dl
			  WHERE dl.delivery_id = d.id) AS meals,
			COALESCE(d.assigned_distance_m,0) AS distance_m,
			d.assignment_mode AS mode, d.assignment_reason AS reason`).
		Order("d." + p.OrderBy() + ", s.sort_order").
		Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[DeliveryRow]{}, fmt.Errorf("postgres: list my deliveries: %w", err)
	}
	return NewPage(items, total, p), nil
}

// Delivery is the minimum a skip or a transition needs.
type Delivery struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	PackageID   *uuid.UUID
	ServiceDate string
	Status      string
	KitchenID   *uuid.UUID
}

// Get loads one delivery.
func (r *DeliveryRepo) Get(ctx context.Context, id uuid.UUID) (Delivery, error) {
	var d Delivery
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, customer_id, customer_package_id AS package_id,
		       service_date::text AS service_date, status, kitchen_id
		  FROM delivery WHERE id = ?`, id).Scan(&d).Error
	if err != nil {
		return Delivery{}, fmt.Errorf("postgres: get delivery: %w", err)
	}
	if d.ID == uuid.Nil {
		return Delivery{}, ErrNotFound
	}
	return d, nil
}

// StatusOf returns the current status and kitchen, for the transition check.
func (r *DeliveryRepo) StatusOf(ctx context.Context, id uuid.UUID) (string, *uuid.UUID, error) {
	d, err := r.Get(ctx, id)
	if err != nil {
		return "", nil, err
	}
	return d.Status, d.KitchenID, nil
}

// Advance writes a validated status change with its timestamp.
func (r *DeliveryRepo) Advance(ctx context.Context, id uuid.UUID, to, reason string, by uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE delivery
		   SET status = ?,
		       prepared_at   = CASE WHEN ? = 'PREPARING'        THEN now() ELSE prepared_at   END,
		       dispatched_at = CASE WHEN ? = 'OUT_FOR_DELIVERY' THEN now() ELSE dispatched_at END,
		       delivered_at  = CASE WHEN ? = 'DELIVERED'        THEN now() ELSE delivered_at  END,
		       delivered_by  = CASE WHEN ? = 'DELIVERED'        THEN ?    ELSE delivered_by   END,
		       failure_reason = CASE WHEN ? = 'FAILED' THEN NULLIF(?,'') ELSE failure_reason END
		 WHERE id = ?`, to, to, to, to, to, by, to, reason, id)
	if res.Error != nil {
		return fmt.Errorf("postgres: advance delivery: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Skip cancels a delivery and returns the credit only before the cut-off.
//
// The credit return and the status change are ONE transaction: a skip that
// returned a credit but left the delivery scheduled would have the customer
// paid twice for one meal.
func (r *DeliveryRepo) Skip(ctx context.Context, id uuid.UUID, reason string,
	beforeCutoff bool, by uuid.UUID) (bool, error) {

	returned := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var d Delivery
		if err := tx.Raw(`
			SELECT id, customer_id, customer_package_id AS package_id,
			       service_date::text AS service_date, status, kitchen_id
			  FROM delivery WHERE id = ? FOR UPDATE`, id).Scan(&d).Error; err != nil {
			return err
		}
		if d.ID == uuid.Nil {
			return ErrNotFound
		}

		if err := tx.Exec(`
			UPDATE delivery SET status='SKIPPED', skip_reason=NULLIF(?,'')
			 WHERE id=? AND status='SCHEDULED'`, reason, id).Error; err != nil {
			return err
		}

		// Give the meal's capacity back so somebody else can buy it.
		if err := tx.Exec(`
			UPDATE scheduled_meal m
			   SET qty_reserved = GREATEST(0, m.qty_reserved - agg.qty)
			  FROM (SELECT scheduled_meal_id, SUM(qty) AS qty FROM delivery_line
			         WHERE delivery_id = ? AND scheduled_meal_id IS NOT NULL
			         GROUP BY scheduled_meal_id) agg
			 WHERE m.id = agg.scheduled_meal_id`, id).Error; err != nil {
			return err
		}

		if d.PackageID != nil && beforeCutoff {
			if err := tx.Exec(`
				INSERT INTO credit_ledger
				  (id, customer_id, customer_package_id, entry_type, qty,
				   reference_type, reference_id, note, created_by)
				VALUES (?,?,?, 'REFUND', 1, 'delivery', ?, 'skipped before cut-off', ?)`,
				uuid.Must(uuid.NewV7()), d.CustomerID, *d.PackageID, id, by).Error; err != nil {
				return err
			}
			// The package may have been EXHAUSTED; a returned credit revives it.
			if err := tx.Exec(`
				UPDATE customer_package SET status='ACTIVE'
				 WHERE id=? AND status='EXHAUSTED'`, *d.PackageID).Error; err != nil {
				return err
			}
			returned = true
		}
		return nil
	})
	return returned, err
}

// Reassign moves a delivery to another kitchen by hand.
func (r *DeliveryRepo) Reassign(ctx context.Context, id, kitchenID uuid.UUID,
	reason string, by uuid.UUID) error {

	var exists int64
	if err := r.db.WithContext(ctx).Raw(
		`SELECT count(*) FROM kitchen WHERE id=? AND is_active`, kitchenID).Scan(&exists).Error; err != nil {
		return err
	}
	if exists == 0 {
		return ErrNotFound
	}

	// assignment_mode becomes MANUAL, and the auto-router never touches a
	// MANUAL assignment again (PROMPT §9.3).
	res := r.db.WithContext(ctx).Exec(`
		UPDATE delivery
		   SET kitchen_id = ?, assignment_mode = 'MANUAL',
		       assignment_reason = ?, assigned_at = now(),
		       assigned_distance_m = ROUND(ST_Distance(
		         (SELECT k.geom FROM kitchen k WHERE k.id = ?), delivery.geom))::int
		 WHERE id = ? AND status = 'SCHEDULED'`, kitchenID, reason, kitchenID, id)
	if res.Error != nil {
		return fmt.Errorf("postgres: reassign: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
