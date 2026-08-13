package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
	"github.com/stevenwilliam/healthy_catering/internal/domain/order"
	"github.com/stevenwilliam/healthy_catering/internal/domain/pricing"
	pid "github.com/stevenwilliam/healthy_catering/internal/platform/id"
)

// Errors the ordering service turns into customer-facing messages.
var (
	ErrCapacityFull     = errors.New("postgres: meal capacity exhausted")
	ErrKitchenFull      = errors.New("postgres: kitchen capacity exhausted")
	ErrIdempotentReplay = errors.New("postgres: idempotent replay")
)

// OrderRepo owns orders, their lines, deliveries and payments.
type OrderRepo struct{ db *gorm.DB }

func NewOrderRepo(db *gorm.DB) *OrderRepo { return &OrderRepo{db: db} }

// Address is a customer's delivery address, as checkout needs it.
type Address struct {
	ID             uuid.UUID
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
	DriverNote     string
}

// AddressForCustomer loads one address, scoped to its owner.
//
// The customer id is in the WHERE clause, not checked afterwards: that is what
// makes an IDOR attempt return "not found" rather than someone else's address
// (PROMPT §14).
func (r *OrderRepo) AddressForCustomer(ctx context.Context, customerID, addressID uuid.UUID) (Address, error) {
	var a Address
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, label, recipient_name, recipient_phone, address_line, district, city,
		       province, postal_code, latitude::float8, longitude::float8, driver_note
		  FROM customer_address
		 WHERE id = ? AND customer_id = ? AND is_active`, addressID, customerID).Scan(&a).Error
	if err != nil {
		return Address{}, fmt.Errorf("postgres: address: %w", err)
	}
	if a.ID == uuid.Nil {
		return Address{}, ErrNotFound
	}
	return a, nil
}

// PreparedLine is a fully priced cart line, ready to persist.
type PreparedLine struct {
	Meal        Meal
	Qty         int
	Address     Address
	ServiceDate time.Time

	UnitPrice   money.IDR
	NormalPrice money.IDR
	LineTotal   money.IDR
	Split       money.Split
	IsPromo     bool
	PromoLabel  string
	PriceRowID  uuid.UUID
	PriceTable  string
	TierID      uuid.UUID
	Trace       pricing.Trace
}

// PreparedDelivery is one routed drop.
type PreparedDelivery struct {
	ServiceDate time.Time
	SlotID      uuid.UUID
	SlotAlias   string
	Address     Address
	KitchenID   uuid.UUID
	KitchenName string
	DistanceM   int
	Reason      string
	FeeIDR      money.IDR
	Lines       []int
}

// PlaceOrderParams is everything the transaction needs.
type PlaceOrderParams struct {
	CustomerID     uuid.UUID
	CustomerTypeID uuid.UUID
	Lines          []PreparedLine
	Deliveries     map[string]*PreparedDelivery
	Totals         order.Totals
	UseUniqueCode  bool
	Deadline       time.Time
	IdempotencyKey string
	TaxRateBps     int
}

// PlaceOrderResult is what the caller reports back.
type PlaceOrderResult struct {
	OrderID          uuid.UUID
	OrderCode        string
	TotalIDR         int64 `gorm:"column:total_idr"`
	PaymentAmountIDR int64 `gorm:"column:payment_amount_idr"`
	UniqueCode       int
	Deliveries       []CreatedDelivery
}

// CreatedDelivery is one persisted delivery.
type CreatedDelivery struct {
	ID          uuid.UUID
	ServiceDate string
	Slot        string
	Kitchen     string
	Reason      string
}

// PlaceOrder writes the whole checkout in ONE transaction.
//
// Two capacity counters must both hold — the meal's and the kitchen's — and
// they are taken in a FIXED ORDER (meal id, then kitchen+date+slot) across
// every caller. Two concurrent orders touching the same pair in opposite orders
// is the textbook deadlock, and the fixed order is what prevents it.
//
// Neither counter is read-then-written: the UPDATE carries its own guard and
// the database's CHECK constraint is the final authority, so a race loses at
// the row rather than in application logic (CLAUDE.md §4).
func (r *OrderRepo) PlaceOrder(ctx context.Context, p PlaceOrderParams) (PlaceOrderResult, error) {
	var out PlaceOrderResult

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if p.IdempotencyKey != "" {
			// Idempotency lives in Postgres, in the same transaction as the
			// write it guards (docs/02 D-4).
			res := tx.Exec(`
				INSERT INTO idempotency_key (id, key, user_id, endpoint, request_hash, state)
				VALUES (?,?,?,?,?, 'IN_PROGRESS')
				ON CONFLICT (key, endpoint) DO NOTHING`,
				uuid.Must(uuid.NewV7()), p.IdempotencyKey, p.CustomerID,
				"POST /api/v1/orders", "")
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrIdempotentReplay
			}
		}

		// ── Capacity, in a fixed order ──────────────────────────────────────
		mealQty := map[uuid.UUID]int{}
		for _, l := range p.Lines {
			mealQty[l.Meal.ID] += l.Qty
		}
		mealIDs := make([]uuid.UUID, 0, len(mealQty))
		for id := range mealQty {
			mealIDs = append(mealIDs, id)
		}
		sort.Slice(mealIDs, func(i, j int) bool { return mealIDs[i].String() < mealIDs[j].String() })

		for _, id := range mealIDs {
			res := tx.Exec(`
				UPDATE scheduled_meal
				   SET qty_reserved = qty_reserved + ?
				 WHERE id = ?
				   AND status = 'PUBLISHED'
				   AND (qty_capacity IS NULL OR qty_reserved + ? <= qty_capacity)`,
				mealQty[id], id, mealQty[id])
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrCapacityFull
			}
		}

		type kitchenKey struct {
			kitchen uuid.UUID
			date    string
			slot    uuid.UUID
		}
		kitchenQty := map[kitchenKey]int{}
		for _, d := range p.Deliveries {
			q := 0
			for _, li := range d.Lines {
				q += p.Lines[li].Qty
			}
			kitchenQty[kitchenKey{d.KitchenID, d.ServiceDate.Format("2006-01-02"), d.SlotID}] += q
		}
		keys := make([]kitchenKey, 0, len(kitchenQty))
		for k := range kitchenQty {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			a := keys[i].kitchen.String() + keys[i].date + keys[i].slot.String()
			b := keys[j].kitchen.String() + keys[j].date + keys[j].slot.String()
			return a < b
		})

		for _, k := range keys {
			// A kitchen with no capacity row for that date is UNLIMITED, which
			// is how a new kitchen behaves before anyone plans its day. The
			// upsert only constrains once a row exists.
			var exists int64
			if err := tx.Raw(`SELECT count(*) FROM kitchen_capacity
			                   WHERE kitchen_id=? AND service_date=?::date AND slot_id=?`,
				k.kitchen, k.date, k.slot).Scan(&exists).Error; err != nil {
				return err
			}
			if exists == 0 {
				continue
			}
			res := tx.Exec(`
				UPDATE kitchen_capacity
				   SET reserved_portions = reserved_portions + ?
				 WHERE kitchen_id = ? AND service_date = ?::date AND slot_id = ?
				   AND reserved_portions + ? <= max_portions`,
				kitchenQty[k], k.kitchen, k.date, k.slot, kitchenQty[k])
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrKitchenFull
			}
		}

		// ── The order ───────────────────────────────────────────────────────
		orderID := uuid.Must(uuid.NewV7())
		code, err := pid.OrderCode()
		if err != nil {
			return err
		}

		uniqueCode := 0
		paymentAmount := p.Totals.Total
		var bankAccountID *uuid.UUID
		var banks []uuid.UUID
		if err := tx.Raw(`SELECT id FROM bank_account WHERE is_active ORDER BY sort_order LIMIT 1`).
			Scan(&banks).Error; err != nil {
			return err
		}
		if len(banks) > 0 {
			bankAccountID = &banks[0]
		}

		if p.UseUniqueCode {
			// The suffix must be unique among orders currently awaiting money,
			// enforced by a partial unique index. Retry a few times, then fall
			// back to no suffix and flag for manual matching (docs/02 D-16).
			for attempt := 0; attempt < 12; attempt++ {
				n, err := pid.UniqueCode()
				if err != nil {
					return err
				}
				candidate := p.Totals.Total + money.IDR(n)
				var clash int64
				if err := tx.Raw(`
					SELECT count(*) FROM customer_order
					 WHERE bank_account_id IS NOT DISTINCT FROM ?
					   AND payment_amount_idr = ?
					   AND status IN ('AWAITING_PAYMENT','PAYMENT_SUBMITTED')`,
					bankAccountID, int64(candidate)).Scan(&clash).Error; err != nil {
					return err
				}
				if clash == 0 {
					uniqueCode, paymentAmount = n, candidate
					break
				}
			}
		}

		trace, _ := json.Marshal(map[string]any{
			"lines": traceOf(p.Lines),
		})

		if err := tx.Exec(`
			INSERT INTO customer_order
			  (id, order_code, customer_id, customer_type_id, order_type, status,
			   subtotal_idr, delivery_fee_idr, discount_idr, total_idr,
			   tax_base_idr, tax_idr, tax_rate_bps,
			   unique_code, payment_rounding_idr, payment_amount_idr, bank_account_id,
			   payment_deadline_at, placed_at, price_trace)
			VALUES (?,?,?,?, 'MEAL', 'AWAITING_PAYMENT',
			        ?,?,0,?, ?,?,?, ?,?,?,?, ?, now(), ?::jsonb)`,
			orderID, code, p.CustomerID, p.CustomerTypeID,
			int64(p.Totals.Subtotal), int64(p.Totals.DeliveryFee), int64(p.Totals.Total),
			int64(p.Totals.TaxBase), int64(p.Totals.Tax), p.TaxRateBps,
			nullableInt(uniqueCode), int64(paymentAmount-p.Totals.Total), int64(paymentAmount),
			bankAccountID, p.Deadline, string(trace)).Error; err != nil {
			return err
		}

		// ── Lines, with their snapshots ─────────────────────────────────────
		lineIDs := make([]uuid.UUID, len(p.Lines))
		for i, l := range p.Lines {
			lineID := uuid.Must(uuid.NewV7())
			lineIDs[i] = lineID
			snapshot, err := json.Marshal(mealSnapshot(l.Meal))
			if err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO order_line
				  (id, order_id, line_no, line_type, scheduled_meal_id, service_date, slot_id,
				   diet_type_id, qty, unit_price_idr, normal_price_idr, line_total_idr,
				   line_tax_base_idr, line_tax_idr, is_promo, promo_label,
				   price_row_id, price_table, tier_id, meal_snapshot)
				VALUES (?,?,?, 'MEAL', ?, ?::date, ?, ?, ?, ?,?,?, ?,?, ?,?, ?,?,?, ?::jsonb)`,
				lineID, orderID, i+1, l.Meal.ID, l.Meal.ServiceDate, l.Meal.SlotID,
				l.Meal.DietTypeID, l.Qty, int64(l.UnitPrice), int64(l.NormalPrice),
				int64(l.LineTotal), int64(l.Split.Base), int64(l.Split.Tax),
				l.IsPromo, l.PromoLabel, nullableUUID(l.PriceRowID), l.PriceTable,
				nullableUUID(l.TierID), string(snapshot)).Error; err != nil {
				return err
			}
		}

		// ── Deliveries, one per date+slot+address ───────────────────────────
		for _, d := range p.Deliveries {
			deliveryID := uuid.Must(uuid.NewV7())
			dcode, err := pid.Token(10)
			if err != nil {
				return err
			}
			addrSnapshot, err := json.Marshal(addressSnapshot(d.Address))
			if err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO delivery
				  (id, delivery_code, customer_id, order_id, service_date, slot_id,
				   address_id, address_snapshot, latitude, longitude,
				   kitchen_id, assigned_distance_m, assignment_mode, assignment_reason,
				   assigned_at, delivery_fee_idr, status, driver_note)
				VALUES (?,?,?,?, ?::date, ?, ?, ?::jsonb, ?, ?, ?, ?, 'AUTO', ?, now(), ?, 'SCHEDULED', ?)`,
				deliveryID, dcode, p.CustomerID, orderID,
				d.ServiceDate.Format("2006-01-02"), d.SlotID,
				d.Address.ID, string(addrSnapshot), d.Address.Latitude, d.Address.Longitude,
				d.KitchenID, d.DistanceM, d.Reason, int64(d.FeeIDR), d.Address.DriverNote).Error; err != nil {
				return err
			}

			for _, li := range d.Lines {
				l := p.Lines[li]
				snapshot, err := json.Marshal(mealSnapshot(l.Meal))
				if err != nil {
					return err
				}
				if err := tx.Exec(`
					INSERT INTO delivery_line
					  (id, delivery_id, scheduled_meal_id, order_line_id, diet_type_id, qty, meal_snapshot)
					VALUES (?,?,?,?,?,?,?::jsonb)`,
					uuid.Must(uuid.NewV7()), deliveryID, l.Meal.ID, lineIDs[li],
					l.Meal.DietTypeID, l.Qty, string(snapshot)).Error; err != nil {
					return err
				}
			}

			out.Deliveries = append(out.Deliveries, CreatedDelivery{
				ID: deliveryID, ServiceDate: d.ServiceDate.Format("2006-01-02"),
				Slot: d.SlotAlias, Kitchen: d.KitchenName, Reason: d.Reason,
			})
		}

		// ── The payment record ──────────────────────────────────────────────
		if err := tx.Exec(`
			INSERT INTO payment (id, order_id, provider, bank_account_id, expected_amount_idr, status)
			VALUES (?,?, 'MANUAL_TRANSFER', ?, ?, 'PENDING')`,
			uuid.Must(uuid.NewV7()), orderID, bankAccountID, int64(paymentAmount)).Error; err != nil {
			return err
		}

		if p.IdempotencyKey != "" {
			if err := tx.Exec(`
				UPDATE idempotency_key SET state='COMPLETED', completed_at=now(),
				       response_code=201, response_body=?::jsonb
				 WHERE key=? AND endpoint=?`,
				fmt.Sprintf(`{"order_id":%q,"order_code":%q}`, orderID, code),
				p.IdempotencyKey, "POST /api/v1/orders").Error; err != nil {
				return err
			}
		}

		out.OrderID = orderID
		out.OrderCode = code
		out.TotalIDR = int64(p.Totals.Total)
		out.PaymentAmountIDR = int64(paymentAmount)
		out.UniqueCode = uniqueCode
		return nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "no_oversell") {
			return PlaceOrderResult{}, ErrCapacityFull
		}
		return PlaceOrderResult{}, err
	}
	return out, nil
}

func traceOf(lines []PreparedLine) []pricing.Trace {
	out := make([]pricing.Trace, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.Trace)
	}
	return out
}

// mealSnapshot freezes what was bought — the foods, their roles and their
// nutrition — so a later recipe edit cannot rewrite history (PROMPT §5.6).
func mealSnapshot(m Meal) map[string]any {
	items := make([]map[string]any, 0, len(m.Items))
	for _, it := range m.Items {
		items = append(items, map[string]any{
			"food_id": it.FoodID, "name": it.FoodName, "role": it.ItemRole,
			"calories_kcal": it.CaloriesKcal, "protein_mg": it.ProteinMg,
			"fat_mg": it.FatMg, "carbohydrate_mg": it.CarbohydrateMg,
			"sodium_mg": it.SodiumMg,
		})
	}
	return map[string]any{
		"meal_id": m.ID, "name": m.Name, "diet_type": m.DietTypeName,
		"slot": m.SlotAlias, "service_date": m.ServiceDate,
		"items": items, "nutrition": m.Nutrition,
	}
}

func addressSnapshot(a Address) map[string]any {
	return map[string]any{
		"label": a.Label, "recipient_name": a.RecipientName,
		"recipient_phone": a.RecipientPhone, "address_line": a.AddressLine,
		"district": a.District, "city": a.City, "province": a.Province,
		"postal_code": a.PostalCode, "latitude": a.Latitude, "longitude": a.Longitude,
		"driver_note": a.DriverNote,
	}
}

func nullableInt(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
