package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

// ReportRepo produces the operational reports (PROMPT §12).
//
// Every one is kitchen-scopable: a kitchen-scoped staff user sees only their
// own kitchen (docs/02 D-21), and the filter lives here rather than in the
// handler so a new report cannot forget it.
type ReportRepo struct{ db *gorm.DB }

func NewReportRepo(db *gorm.DB) *ReportRepo { return &ReportRepo{db: db} }

// ReportScope narrows every report the same way.
type ReportScope struct {
	From      time.Time
	To        time.Time
	KitchenID *uuid.UUID
	SlotID    *uuid.UUID
	Q         string
}

// ProductionRow is one line of the kitchen production sheet: what to cook.
type ProductionRow struct {
	ServiceDate string `json:"service_date"`
	Slot        string `json:"slot"`
	Kitchen     string `json:"kitchen"`
	DietType    string `json:"diet_type"`
	FoodName    string `json:"food_name"`
	ItemRole    string `json:"item_role"`
	Portions    int    `json:"portions"`
}

// ProductionSheet is what each kitchen cooks from (PROMPT §12.2).
//
// It counts PORTIONS OF EACH FOOD, not meals: the kitchen needs to know it is
// making 40 pieces of chicken and 40 portions of vegetables, and a meal count
// alone does not tell them that.
func (r *ReportRepo) ProductionSheet(ctx context.Context, s ReportScope) ([]ProductionRow, error) {
	out := []ProductionRow{}
	db := r.db.WithContext(ctx).Table("delivery d").
		Joins("JOIN delivery_line dl ON dl.delivery_id = d.id").
		Joins("JOIN scheduled_meal m ON m.id = dl.scheduled_meal_id").
		Joins("JOIN scheduled_meal_item mi ON mi.scheduled_meal_id = m.id").
		Joins("JOIN food f ON f.id = mi.food_id").
		Joins("JOIN diet_type dt ON dt.id = m.diet_type_id").
		Joins("JOIN delivery_time_slot sl ON sl.id = d.slot_id").
		Joins("LEFT JOIN kitchen k ON k.id = d.kitchen_id").
		// Cancelled and skipped deliveries are not cooked.
		Where("d.status IN ('SCHEDULED','PREPARING','OUT_FOR_DELIVERY')").
		Where("d.service_date BETWEEN ?::date AND ?::date",
			s.From.Format("2006-01-02"), s.To.Format("2006-01-02"))

	if s.KitchenID != nil {
		db = db.Where("d.kitchen_id = ?", *s.KitchenID)
	}
	if s.SlotID != nil {
		db = db.Where("d.slot_id = ?", *s.SlotID)
	}
	if s.Q != "" {
		pattern := SearchPattern(s.Q)
		db = db.Where("lower(f.name) LIKE ? OR lower(dt.name) LIKE ?", pattern, pattern)
	}

	err := db.Select(`d.service_date::text AS service_date, sl.alias AS slot,
		COALESCE(k.name,'unassigned') AS kitchen, dt.name AS diet_type,
		f.name AS food_name, mi.item_role, SUM(dl.qty)::int AS portions`).
		Group("d.service_date, sl.alias, sl.sort_order, k.name, dt.name, dt.sort_order, f.name, mi.item_role").
		Order("d.service_date, sl.sort_order, k.name, dt.sort_order, mi.item_role, f.name").
		Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: production sheet: %w", err)
	}
	return out, nil
}

// PackingLabel is one label per delivery LINE (PROMPT §12.3).
type PackingLabel struct {
	DeliveryID   uuid.UUID `json:"delivery_id"`
	DeliveryCode string    `json:"delivery_code"`
	ServiceDate  string    `json:"service_date"`
	Slot         string    `json:"slot"`
	Kitchen      string    `json:"kitchen_code"`
	CustomerName string    `json:"customer_name"`
	Phone        string    `json:"phone"`
	AddressLine  string    `json:"address_line"`
	District     string    `json:"district"`
	DietType     string    `json:"diet_type"`
	Qty          int       `json:"qty"`
	Foods        string    `json:"foods"`
	Allergens    string    `json:"allergens"`
	DriverNote   string    `json:"driver_note"`
}

// PackingLabels prints from the SNAPSHOT, not the live menu: a substitution
// made after the order must not change the label that was already printed.
func (r *ReportRepo) PackingLabels(ctx context.Context, s ReportScope) ([]PackingLabel, error) {
	out := []PackingLabel{}
	db := r.db.WithContext(ctx).Table("delivery d").
		Joins("JOIN delivery_line dl ON dl.delivery_id = d.id").
		Joins("LEFT JOIN diet_type dt ON dt.id = dl.diet_type_id").
		Joins("JOIN delivery_time_slot sl ON sl.id = d.slot_id").
		Joins("LEFT JOIN kitchen k ON k.id = d.kitchen_id").
		Where("d.status IN ('SCHEDULED','PREPARING','OUT_FOR_DELIVERY')").
		Where("d.service_date BETWEEN ?::date AND ?::date",
			s.From.Format("2006-01-02"), s.To.Format("2006-01-02"))
	if s.KitchenID != nil {
		db = db.Where("d.kitchen_id = ?", *s.KitchenID)
	}
	if s.SlotID != nil {
		db = db.Where("d.slot_id = ?", *s.SlotID)
	}

	err := db.Select(`d.id AS delivery_id, d.delivery_code,
		d.service_date::text AS service_date, sl.alias AS slot,
		COALESCE(k.code,'-') AS kitchen,
		d.address_snapshot->>'recipient_name' AS customer_name,
		d.address_snapshot->>'recipient_phone' AS phone,
		d.address_snapshot->>'address_line' AS address_line,
		COALESCE(d.address_snapshot->>'district','') AS district,
		COALESCE(dt.name,'') AS diet_type, dl.qty,
		COALESCE((SELECT string_agg(i->>'name', ', ' ORDER BY i->>'role')
		            FROM jsonb_array_elements(dl.meal_snapshot->'items') i), '') AS foods,
		COALESCE((SELECT string_agg(DISTINCT a.name_id, ', ')
		            FROM jsonb_array_elements(dl.meal_snapshot->'items') i
		            JOIN food_allergen fa ON fa.food_id = (i->>'food_id')::uuid
		            JOIN allergen a ON a.id = fa.allergen_id), '') AS allergens,
		COALESCE(d.driver_note,'') AS driver_note`).
		Order("d.service_date, sl.sort_order, k.code, d.delivery_code").
		Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: packing labels: %w", err)
	}
	return out, nil
}

// ManifestStop is one stop on a courier's run (PROMPT §12.4).
type ManifestStop struct {
	Seq          int       `json:"seq"`
	DeliveryID   uuid.UUID `json:"delivery_id"`
	DeliveryCode string    `json:"delivery_code"`
	CustomerName string    `json:"customer_name"`
	Phone        string    `json:"phone"`
	AddressLine  string    `json:"address_line"`
	District     string    `json:"district"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	MapsURL      string    `json:"maps_url"`
	DistanceM    int       `json:"distance_m"`
	Meals        int       `json:"meals"`
	DriverNote   string    `json:"driver_note"`
	Status       string    `json:"status"`
}

// CourierManifest orders stops by distance from the kitchen.
//
// Distance order is a rough proxy for a route, not an optimal one — but a
// manifest in random order sends a courier back and forth across Jakarta, and
// the difference is an hour of fuel per run.
func (r *ReportRepo) CourierManifest(ctx context.Context, s ReportScope) ([]ManifestStop, error) {
	out := []ManifestStop{}
	if s.KitchenID == nil {
		return nil, fmt.Errorf("postgres: a manifest needs a kitchen")
	}
	slotFilter := ""
	args := []any{*s.KitchenID, s.From.Format("2006-01-02"), s.To.Format("2006-01-02")}
	if s.SlotID != nil {
		slotFilter = " AND d.slot_id = ?"
		args = append(args, *s.SlotID)
	}

	err := r.db.WithContext(ctx).Raw(`
		SELECT d.id AS delivery_id, d.delivery_code,
		       d.address_snapshot->>'recipient_name' AS customer_name,
		       d.address_snapshot->>'recipient_phone' AS phone,
		       d.address_snapshot->>'address_line' AS address_line,
		       COALESCE(d.address_snapshot->>'district','') AS district,
		       d.latitude::float8, d.longitude::float8,
		       COALESCE(d.assigned_distance_m,0) AS distance_m,
		       (SELECT COALESCE(SUM(dl.qty),0) FROM delivery_line dl WHERE dl.delivery_id = d.id) AS meals,
		       COALESCE(d.driver_note,'') AS driver_note, d.status
		  FROM delivery d
		 WHERE d.kitchen_id = ?
		   AND d.service_date BETWEEN ?::date AND ?::date
		   AND d.status IN ('SCHEDULED','PREPARING','OUT_FOR_DELIVERY')`+slotFilter+`
		 ORDER BY COALESCE(d.assigned_distance_m, 0), d.delivery_code`,
		args...).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: manifest: %w", err)
	}
	for i := range out {
		out[i].Seq = i + 1
		// The Maps deep link is built HERE, not in SQL: a literal "?" inside a
		// SQL string is consumed by gorm as a bind placeholder, which shifts
		// every following argument and produces a syntax error nowhere near
		// the cause. "?api=1" cost a debugging session.
		out[i].MapsURL = fmt.Sprintf(
			"https://www.google.com/maps/dir/?api=1&destination=%.6f,%.6f",
			out[i].Latitude, out[i].Longitude)
	}
	return out, nil
}

// CoverageRow is one district we could not serve (PROMPT §12.4b).
type CoverageRow struct {
	District       string  `json:"district"`
	City           string  `json:"city"`
	Attempts       int     `json:"attempts"`
	NotifyRequests int     `json:"notify_requests"`
	AvgDistanceKM  float64 `json:"avg_distance_to_nearest_km"`
	NearestKitchen string  `json:"nearest_kitchen"`
}

// CoverageReport is the map of where to open the next kitchen.
func (r *ReportRepo) CoverageReport(ctx context.Context, s ReportScope) ([]CoverageRow, error) {
	out := []CoverageRow{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(NULLIF(o.district,''),'(not given)') AS district,
		       COALESCE(NULLIF(o.city,''),'(not given)') AS city,
		       count(*)::int AS attempts,
		       count(*) FILTER (WHERE o.notify_requested)::int AS notify_requests,
		       ROUND(AVG(o.nearest_distance_m)/1000.0, 1)::float8 AS avg_distance_km,
		       COALESCE(MODE() WITHIN GROUP (ORDER BY k.code), '-') AS nearest_kitchen
		  FROM out_of_range_attempt o
		  LEFT JOIN kitchen k ON k.id = o.nearest_kitchen_id
		 WHERE o.occurred_at BETWEEN ?::date AND (?::date + 1)
		 GROUP BY 1, 2
		 ORDER BY attempts DESC, district`,
		s.From.Format("2006-01-02"), s.To.Format("2006-01-02")).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: coverage report: %w", err)
	}
	return out, nil
}

// SalesRow is one line of the sales report (PROMPT §12.5).
type SalesRow struct {
	Period           string `json:"period"`
	CustomerType     string `json:"customer_type"`
	DietType         string `json:"diet_type"`
	OrderType        string `json:"order_type"`
	Orders           int    `json:"orders"`
	Meals            int    `json:"meals"`
	GrossIDR         int64  `json:"gross_idr" gorm:"column:gross_idr"`
	Gross            string `json:"gross"`
	TaxBaseIDR       int64  `json:"tax_base_idr" gorm:"column:tax_base_idr"`
	TaxIDR           int64  `json:"tax_idr" gorm:"column:tax_idr"`
	DiscountGivenIDR int64  `json:"promo_discount_idr" gorm:"column:discount_given_idr"`
	DiscountGiven    string `json:"promo_discount"`
}

// SalesReport aggregates paid revenue.
//
// Promo impact is (normal - promo) x qty, which is the discount actually given
// (PROMPT §12.5) — computed from the SNAPSHOT on the line, so a later price
// change cannot rewrite last month's numbers.
func (r *ReportRepo) SalesReport(ctx context.Context, s ReportScope, groupBy string) ([]SalesRow, error) {
	period := "to_char(o.paid_at, 'YYYY-MM-DD')"
	switch groupBy {
	case "month":
		period = "to_char(o.paid_at, 'YYYY-MM')"
	case "week":
		period = "to_char(o.paid_at, 'IYYY-\"W\"IW')"
	}

	out := []SalesRow{}
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT %s AS period,
		       ct.name AS customer_type,
		       COALESCE(dt.name, '(package)') AS diet_type,
		       o.order_type,
		       count(DISTINCT o.id)::int AS orders,
		       COALESCE(SUM(ol.qty),0)::int AS meals,
		       COALESCE(SUM(ol.line_total_idr),0) AS gross_idr,
		       COALESCE(SUM(ol.line_tax_base_idr),0) AS tax_base_idr,
		       COALESCE(SUM(ol.line_tax_idr),0) AS tax_idr,
		       COALESCE(SUM((ol.normal_price_idr - ol.unit_price_idr) * ol.qty)
		                FILTER (WHERE ol.is_promo), 0) AS discount_given_idr
		  FROM customer_order o
		  JOIN order_line ol ON ol.order_id = o.id
		  JOIN customer_type ct ON ct.id = o.customer_type_id
		  LEFT JOIN diet_type dt ON dt.id = ol.diet_type_id
		 WHERE o.status IN ('PAID','COMPLETED')
		   AND o.paid_at BETWEEN ?::date AND (?::date + 1)
		 GROUP BY 1, 2, 3, 4
		 ORDER BY 1 DESC, 2, 3`, period),
		s.From.Format("2006-01-02"), s.To.Format("2006-01-02")).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: sales report: %w", err)
	}
	for i := range out {
		out[i].Gross = money.Format(money.IDR(out[i].GrossIDR))
		out[i].DiscountGiven = money.Format(money.IDR(out[i].DiscountGivenIDR))
	}
	return out, nil
}

// CreditReportRow is the report PROMPT §7 names explicitly.
type CreditReportRow struct {
	CustomerName      string    `json:"customer_name"`
	CustomerEmail     string    `json:"customer_email"`
	PackageName       string    `json:"package_name"`
	DatetimeToday     string    `json:"datetime_today"`
	DatePurchase      string    `json:"date_purchase"`
	DateExpired       string    `json:"date_expired"`
	PurchasedCredit   int       `json:"purchased_credit"`
	RemainingCredit   int       `json:"remaining_credit"`
	Status            string    `json:"status"`
	CustomerPackageID uuid.UUID `json:"customer_package_id"`
}

// CreditReport lists every package with its live balance.
func (r *ReportRepo) CreditReport(ctx context.Context, p ListParams) (Page[CreditReportRow], error) {
	p = p.Normalise("purchased_at", "purchased_at", "expires_at")

	base := r.db.WithContext(ctx).Table("customer_package cp").
		Joins("JOIN customer c ON c.id = cp.customer_id").
		Joins("JOIN app_user u ON u.id = c.user_id")
	if p.Q != "" {
		pattern := SearchPattern(p.Q)
		base = base.Where(`lower(u.full_name) LIKE ? OR lower(u.email::text) LIKE ?
		                   OR lower(cp.package_name) LIKE ? OR lower(cp.status) LIKE ?`,
			pattern, pattern, pattern, pattern)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[CreditReportRow]{}, fmt.Errorf("postgres: count credit report: %w", err)
	}

	var items []CreditReportRow
	err := base.Session(&gorm.Session{}).
		Select(`cp.id AS customer_package_id, u.full_name AS customer_name,
		        u.email::text AS customer_email, cp.package_name,
		        to_char(now() AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD HH24:MI') AS datetime_today,
		        to_char(cp.purchased_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD') AS date_purchase,
		        COALESCE(cp.expires_at::text,'-') AS date_expired,
		        cp.meal_credits AS purchased_credit,
		        COALESCE((SELECT SUM(qty) FROM credit_ledger cl
		                   WHERE cl.customer_package_id = cp.id),0)::int AS remaining_credit,
		        cp.status`).
		Order("cp." + p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[CreditReportRow]{}, fmt.Errorf("postgres: credit report: %w", err)
	}
	return NewPage(items, total, p), nil
}

// UnpaidRow is an order awaiting money or a package about to expire.
type UnpaidRow struct {
	Kind         string `json:"kind"`
	Reference    string `json:"reference"`
	CustomerName string `json:"customer_name"`
	AmountIDR    int64  `json:"amount_idr" gorm:"column:amount_idr"`
	Amount       string `json:"amount"`
	Deadline     string `json:"deadline"`
	MinutesLeft  int    `json:"minutes_left"`
}

// UnpaidAndExpiring is the chase list (PROMPT §12.6).
func (r *ReportRepo) UnpaidAndExpiring(ctx context.Context, expiringDays int) ([]UnpaidRow, error) {
	out := []UnpaidRow{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT 'UNPAID_ORDER' AS kind, o.order_code AS reference,
		       u.full_name AS customer_name, o.payment_amount_idr AS amount_idr,
		       o.payment_deadline_at::text AS deadline,
		       GREATEST(0, EXTRACT(EPOCH FROM (o.payment_deadline_at - now()))/60)::int AS minutes_left
		  FROM customer_order o
		  JOIN customer c ON c.id = o.customer_id
		  JOIN app_user u ON u.id = c.user_id
		 WHERE o.status IN ('AWAITING_PAYMENT','PAYMENT_SUBMITTED')
		UNION ALL
		SELECT 'EXPIRING_PACKAGE', cp.package_name, u.full_name, 0,
		       cp.expires_at::text,
		       (cp.expires_at - CURRENT_DATE) * 1440
		  FROM customer_package cp
		  JOIN customer c ON c.id = cp.customer_id
		  JOIN app_user u ON u.id = c.user_id
		 WHERE cp.status = 'ACTIVE'
		   AND cp.expires_at <= CURRENT_DATE + ?::int
		   AND COALESCE((SELECT SUM(qty) FROM credit_ledger cl
		                  WHERE cl.customer_package_id = cp.id),0) > 0
		 ORDER BY minutes_left`, expiringDays).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: unpaid report: %w", err)
	}
	for i := range out {
		out[i].Amount = money.Format(money.IDR(out[i].AmountIDR))
	}
	return out, nil
}

// RetentionRow is one cohort (PROMPT §12.7).
type RetentionRow struct {
	Cohort        string  `json:"cohort"`
	Customers     int     `json:"customers"`
	Repeat        int     `json:"repeat_customers"`
	RepeatRatePct float64 `json:"repeat_rate_pct"`
	AvgOrders     float64 `json:"avg_orders"`
}

// Retention reports repeat rate by first-order month.
func (r *ReportRepo) Retention(ctx context.Context) ([]RetentionRow, error) {
	out := []RetentionRow{}
	err := r.db.WithContext(ctx).Raw(`
		WITH first_order AS (
		  SELECT customer_id, MIN(paid_at) AS first_paid, COUNT(*) AS orders
		    FROM customer_order WHERE status IN ('PAID','COMPLETED')
		   GROUP BY customer_id
		)
		SELECT to_char(first_paid, 'YYYY-MM') AS cohort,
		       count(*)::int AS customers,
		       count(*) FILTER (WHERE orders > 1)::int AS repeat,
		       ROUND(100.0 * count(*) FILTER (WHERE orders > 1) / NULLIF(count(*),0), 1)::float8 AS repeat_rate_pct,
		       ROUND(AVG(orders), 2)::float8 AS avg_orders
		  FROM first_order
		 GROUP BY 1 ORDER BY 1 DESC`).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: retention: %w", err)
	}
	return out, nil
}
