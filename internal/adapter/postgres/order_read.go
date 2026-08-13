package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

// SaveAddressParams is a new delivery address.
type SaveAddressParams struct {
	CustomerID     uuid.UUID
	Label          string
	RecipientName  string
	RecipientPhone string
	AddressLine    string
	District       string
	City           string
	Province       string
	PostalCode     string
	Latitude       float64
	Longitude      float64
	GooglePlaceID  string
	DriverNote     string
	IsDefault      bool
}

// SaveAddress inserts an address, maintaining the one-default rule.
//
// The partial unique index enforces one default per customer, so clearing the
// old default and setting the new one happen in the SAME transaction — two
// statements outside one would briefly have two defaults and the index would
// refuse the second.
func (r *OrderRepo) SaveAddress(ctx context.Context, p SaveAddressParams) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		isDefault := p.IsDefault
		if !isDefault {
			// The first address a customer saves becomes their default
			// whatever they ticked — otherwise checkout has nothing to
			// preselect and every order starts with a decision.
			var n int64
			if err := tx.Raw(`SELECT count(*) FROM customer_address
			                   WHERE customer_id = ? AND is_active`, p.CustomerID).Scan(&n).Error; err != nil {
				return err
			}
			isDefault = n == 0
		}
		if isDefault {
			if err := tx.Exec(`UPDATE customer_address SET is_default = FALSE
			                    WHERE customer_id = ? AND is_default`, p.CustomerID).Error; err != nil {
				return err
			}
		}
		return tx.Exec(`
			INSERT INTO customer_address
			  (id, customer_id, label, recipient_name, recipient_phone, address_line,
			   district, city, province, postal_code, latitude, longitude,
			   google_place_id, driver_note, is_default, is_active)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,NULLIF(?,''),?,?,TRUE)`,
			id, p.CustomerID, p.Label, p.RecipientName, p.RecipientPhone, p.AddressLine,
			p.District, p.City, p.Province, p.PostalCode, p.Latitude, p.Longitude,
			p.GooglePlaceID, p.DriverNote, isDefault).Error
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres: save address: %w", err)
	}
	return id, nil
}

// ListAddresses returns a customer's active addresses.
func (r *OrderRepo) ListAddresses(ctx context.Context, customerID uuid.UUID) ([]Address, error) {
	out := []Address{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, label, recipient_name, recipient_phone, address_line, district, city,
		       province, postal_code, latitude::float8, longitude::float8, driver_note
		  FROM customer_address
		 WHERE customer_id = ? AND is_active
		 ORDER BY is_default DESC, label`, customerID).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: list addresses: %w", err)
	}
	return out, nil
}

// OrderSummary is one row of the customer's order list.
type OrderSummary struct {
	ID               uuid.UUID `json:"id"`
	OrderCode        string    `json:"order_code"`
	OrderType        string    `json:"order_type"`
	Status           string    `json:"status"`
	TotalIDR         int64     `json:"total_idr" gorm:"column:total_idr"`
	Total            string    `json:"total"`
	PaymentAmountIDR int64     `json:"payment_amount_idr" gorm:"column:payment_amount_idr"`
	PaymentAmount    string    `json:"payment_amount"`
	PlacedAt         *string   `json:"placed_at,omitempty"`
	PaymentDeadline  *string   `json:"payment_deadline,omitempty"`
	DeliveryCount    int       `json:"delivery_count"`
	// FulfilmentStatus is DERIVED from the deliveries (docs/02 D-15), so the
	// API still answers §6.3's vocabulary without the order pretending to be
	// in a fulfilment state.
	FulfilmentStatus string `json:"fulfilment_status,omitempty"`
}

// ListForCustomer returns a searchable page of one customer's orders.
func (r *OrderRepo) ListForCustomer(ctx context.Context, customerID uuid.UUID,
	p ListParams) (Page[OrderSummary], error) {

	p = p.Normalise("placed_at", "placed_at", "total_idr", "status")

	base := r.db.WithContext(ctx).Table("customer_order o").
		Where("o.customer_id = ?", customerID)
	if p.Q != "" {
		pattern := SearchPattern(p.Q)
		base = base.Where(`lower(o.order_code) LIKE ? OR lower(o.status) LIKE ?
		                   OR o.total_idr::text LIKE ?`, pattern, pattern, pattern)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[OrderSummary]{}, fmt.Errorf("postgres: count orders: %w", err)
	}

	var items []OrderSummary
	err := base.Session(&gorm.Session{}).
		Select(`o.id, o.order_code, o.order_type, o.status, o.total_idr,
		        o.payment_amount_idr, o.placed_at::text AS placed_at,
		        o.payment_deadline_at::text AS payment_deadline,
		        (SELECT count(*) FROM delivery d WHERE d.order_id = o.id) AS delivery_count`).
		Order("o." + p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[OrderSummary]{}, fmt.Errorf("postgres: list orders: %w", err)
	}
	for i := range items {
		items[i].Total = money.Format(money.IDR(items[i].TotalIDR))
		items[i].PaymentAmount = money.Format(money.IDR(items[i].PaymentAmountIDR))
	}
	return NewPage(items, total, p), nil
}

// OrderCodeOf returns the human-facing code for an order.
func (r *OrderRepo) OrderCodeOf(ctx context.Context, orderID uuid.UUID) (string, error) {
	var code string
	err := r.db.WithContext(ctx).Raw(
		`SELECT order_code FROM customer_order WHERE id = ?`, orderID).Scan(&code).Error
	return code, err
}

// OrderDetail is one order with its lines and deliveries.
type OrderDetail struct {
	OrderSummary
	SubtotalIDR    int64  `json:"subtotal_idr" gorm:"column:subtotal_idr"`
	DeliveryFeeIDR int64  `json:"delivery_fee_idr" gorm:"column:delivery_fee_idr"`
	TaxBaseIDR     int64  `json:"tax_base_idr" gorm:"column:tax_base_idr"`
	TaxIDR         int64  `json:"tax_idr" gorm:"column:tax_idr"`
	TaxRateBps     int    `json:"tax_rate_bps"`
	UniqueCode     *int   `json:"unique_code,omitempty"`
	BankName       string `json:"bank_name,omitempty"`
	BankAccount    string `json:"bank_account_number,omitempty"`
	BankHolder     string `json:"bank_account_holder,omitempty"`

	Lines      []OrderLineDetail `json:"lines"`
	Deliveries []DeliveryDetail  `json:"deliveries"`
}

// OrderLineDetail is one line, showing the snapshot rather than live data.
type OrderLineDetail struct {
	LineNo         int             `json:"line_no"`
	Qty            int             `json:"qty"`
	UnitPriceIDR   int64           `json:"unit_price_idr" gorm:"column:unit_price_idr"`
	UnitPrice      string          `json:"unit_price"`
	NormalPriceIDR int64           `json:"normal_price_idr" gorm:"column:normal_price_idr"`
	NormalPrice    string          `json:"normal_price"`
	IsPromo        bool            `json:"is_promo"`
	PromoLabel     string          `json:"promo_label,omitempty"`
	LineTotalIDR   int64           `json:"line_total_idr" gorm:"column:line_total_idr"`
	LineTotal      string          `json:"line_total"`
	ServiceDate    *string         `json:"service_date,omitempty"`
	MealSnapshot   json.RawMessage `json:"meal"`
}

// DeliveryDetail is one drop.
type DeliveryDetail struct {
	ID           uuid.UUID       `json:"id"`
	DeliveryCode string          `json:"delivery_code"`
	ServiceDate  string          `json:"service_date"`
	Slot         string          `json:"slot"`
	Status       string          `json:"status"`
	Kitchen      string          `json:"kitchen,omitempty"`
	Reason       string          `json:"assignment_reason,omitempty"`
	Address      json.RawMessage `json:"address"`
}

// GetForCustomer returns one order, scoped to its owner.
func (r *OrderRepo) GetForCustomer(ctx context.Context, customerID, orderID uuid.UUID) (OrderDetail, error) {
	var d OrderDetail
	err := r.db.WithContext(ctx).Raw(`
		SELECT o.id, o.order_code, o.order_type, o.status, o.total_idr, o.payment_amount_idr,
		       o.subtotal_idr, o.delivery_fee_idr, o.tax_base_idr, o.tax_idr, o.tax_rate_bps,
		       o.unique_code, o.placed_at::text AS placed_at,
		       o.payment_deadline_at::text AS payment_deadline,
		       COALESCE(b.bank_name,'') AS bank_name,
		       COALESCE(b.account_number,'') AS bank_account,
		       COALESCE(b.account_holder,'') AS bank_holder
		  FROM customer_order o
		  LEFT JOIN bank_account b ON b.id = o.bank_account_id
		 WHERE o.id = ? AND o.customer_id = ?`, orderID, customerID).Scan(&d).Error
	if err != nil {
		return OrderDetail{}, fmt.Errorf("postgres: get order: %w", err)
	}
	if d.ID == uuid.Nil {
		return OrderDetail{}, ErrNotFound
	}
	d.Total = money.Format(money.IDR(d.TotalIDR))
	d.PaymentAmount = money.Format(money.IDR(d.PaymentAmountIDR))

	d.Lines = []OrderLineDetail{}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT line_no, qty, unit_price_idr, normal_price_idr, is_promo, promo_label,
		       line_total_idr, service_date::text AS service_date, meal_snapshot
		  FROM order_line WHERE order_id = ? ORDER BY line_no`, orderID).Scan(&d.Lines).Error; err != nil {
		return OrderDetail{}, fmt.Errorf("postgres: order lines: %w", err)
	}
	for i := range d.Lines {
		d.Lines[i].UnitPrice = money.Format(money.IDR(d.Lines[i].UnitPriceIDR))
		d.Lines[i].NormalPrice = money.Format(money.IDR(d.Lines[i].NormalPriceIDR))
		d.Lines[i].LineTotal = money.Format(money.IDR(d.Lines[i].LineTotalIDR))
	}

	d.Deliveries = []DeliveryDetail{}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT dl.id, dl.delivery_code, dl.service_date::text AS service_date,
		       s.alias AS slot, dl.status, COALESCE(k.name,'') AS kitchen,
		       dl.assignment_reason AS reason, dl.address_snapshot AS address
		  FROM delivery dl
		  JOIN delivery_time_slot s ON s.id = dl.slot_id
		  LEFT JOIN kitchen k ON k.id = dl.kitchen_id
		 WHERE dl.order_id = ?
		 ORDER BY dl.service_date, s.sort_order`, orderID).Scan(&d.Deliveries).Error; err != nil {
		return OrderDetail{}, fmt.Errorf("postgres: order deliveries: %w", err)
	}
	d.DeliveryCount = len(d.Deliveries)
	return d, nil
}
