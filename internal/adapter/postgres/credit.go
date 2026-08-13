package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/credit"
	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

// CreditRepo owns packages sold and the append-only ledger.
type CreditRepo struct{ db *gorm.DB }

func NewCreditRepo(db *gorm.DB) *CreditRepo { return &CreditRepo{db: db} }

// Package is a sellable package.
type Package struct {
	ID           uuid.UUID   `json:"id"`
	Name         string      `json:"name"`
	Slug         string      `json:"slug"`
	Description  string      `json:"description"`
	MealCredits  int         `json:"meal_credits"`
	ValidityDays int         `json:"validity_days"`
	SortOrder    int         `json:"sort_order"`
	IsActive     bool        `json:"is_active"`
	DietTypeIDs  []uuid.UUID `json:"diet_type_ids" gorm:"-"`
}

// ListPackages returns the sellable packages.
func (r *CreditRepo) ListPackages(ctx context.Context, p ListParams) (Page[Package], error) {
	p = p.Normalise("sort_order", "sort_order", "name", "meal_credits")
	base := r.db.WithContext(ctx).Table("package")
	if p.Q != "" {
		pattern := SearchPattern(p.Q)
		base = base.Where("lower(name) LIKE ? OR lower(slug) LIKE ? OR lower(description) LIKE ?",
			pattern, pattern, pattern)
	}
	base = applyActive(base, p.Active)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[Package]{}, fmt.Errorf("postgres: count packages: %w", err)
	}
	var items []Package
	if err := base.Session(&gorm.Session{}).
		Select("id, name, slug, description, meal_credits, validity_days, sort_order, is_active").
		Order(p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error; err != nil {
		return Page[Package]{}, fmt.Errorf("postgres: list packages: %w", err)
	}
	// No package_diet_type rows means ANY diet type (docs/02 D-12), so an empty
	// slice here is meaningful rather than missing data.
	for i := range items {
		var ids []uuid.UUID
		if err := r.db.WithContext(ctx).Raw(
			`SELECT diet_type_id FROM package_diet_type WHERE package_id = ?`,
			items[i].ID).Scan(&ids).Error; err != nil {
			return Page[Package]{}, err
		}
		items[i].DietTypeIDs = ids
	}
	return NewPage(items, total, p), nil
}

// GetPackage loads one.
func (r *CreditRepo) GetPackage(ctx context.Context, id uuid.UUID) (Package, error) {
	var p Package
	if err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, slug, description, meal_credits, validity_days, sort_order, is_active
		  FROM package WHERE id = ? AND is_active`, id).Scan(&p).Error; err != nil {
		return Package{}, fmt.Errorf("postgres: get package: %w", err)
	}
	if p.ID == uuid.Nil {
		return Package{}, ErrNotFound
	}
	if err := r.db.WithContext(ctx).Raw(
		`SELECT diet_type_id FROM package_diet_type WHERE package_id = ?`, id).
		Scan(&p.DietTypeIDs).Error; err != nil {
		return Package{}, err
	}
	return p, nil
}

// BuyPackageParams is a package purchase.
type BuyPackageParams struct {
	CustomerID     uuid.UUID
	CustomerTypeID uuid.UUID
	Package        Package
	PriceIDR       int64
	NormalPriceIDR int64
	IsPromo        bool
	PromoLabel     string
	PriceRowID     uuid.UUID
	PriceTable     string
	TaxBaseIDR     int64
	TaxIDR         int64
	TaxRateBps     int
	UseUniqueCode  bool
	Deadline       time.Time
	IdempotencyKey string
}

// BuyPackage creates the order and a PENDING customer_package.
//
// No credits are issued yet: the active period starts on payment verification
// (docs/02 D-14), because manual transfer can lag by a weekend and a customer
// should not lose validity days they could not order in.
func (r *CreditRepo) BuyPackage(ctx context.Context, p BuyPackageParams) (PlaceOrderResult, error) {
	var out PlaceOrderResult

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if p.IdempotencyKey != "" {
			res := tx.Exec(`
				INSERT INTO idempotency_key (id, key, user_id, endpoint, request_hash, state)
				VALUES (?,?,?,?, '', 'IN_PROGRESS')
				ON CONFLICT (key, endpoint) DO NOTHING`,
				uuid.Must(uuid.NewV7()), p.IdempotencyKey, p.CustomerID, "POST /api/v1/packages/buy")
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrIdempotentReplay
			}
		}

		orderID := uuid.Must(uuid.NewV7())
		code, err := orderCode()
		if err != nil {
			return err
		}

		var banks []uuid.UUID
		if err := tx.Raw(`SELECT id FROM bank_account WHERE is_active ORDER BY sort_order LIMIT 1`).
			Scan(&banks).Error; err != nil {
			return err
		}
		var bankID *uuid.UUID
		if len(banks) > 0 {
			bankID = &banks[0]
		}

		uniqueCode := 0
		payment := p.PriceIDR
		if p.UseUniqueCode {
			for attempt := 0; attempt < 12; attempt++ {
				n, err := uniqueSuffix()
				if err != nil {
					return err
				}
				candidate := p.PriceIDR + int64(n)
				var clash int64
				if err := tx.Raw(`
					SELECT count(*) FROM customer_order
					 WHERE bank_account_id IS NOT DISTINCT FROM ? AND payment_amount_idr = ?
					   AND status IN ('AWAITING_PAYMENT','PAYMENT_SUBMITTED')`,
					bankID, candidate).Scan(&clash).Error; err != nil {
					return err
				}
				if clash == 0 {
					uniqueCode, payment = n, candidate
					break
				}
			}
		}

		if err := tx.Exec(`
			INSERT INTO customer_order
			  (id, order_code, customer_id, customer_type_id, order_type, status,
			   subtotal_idr, delivery_fee_idr, discount_idr, total_idr,
			   tax_base_idr, tax_idr, tax_rate_bps,
			   unique_code, payment_rounding_idr, payment_amount_idr, bank_account_id,
			   payment_deadline_at, placed_at)
			VALUES (?,?,?,?, 'PACKAGE', 'AWAITING_PAYMENT',
			        ?,0,0,?, ?,?,?, ?,?,?,?, ?, now())`,
			orderID, code, p.CustomerID, p.CustomerTypeID,
			p.PriceIDR, p.PriceIDR, p.TaxBaseIDR, p.TaxIDR, p.TaxRateBps,
			nullableInt(uniqueCode), payment-p.PriceIDR, payment, bankID, p.Deadline).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO order_line
			  (id, order_id, line_no, line_type, package_id, qty,
			   unit_price_idr, normal_price_idr, line_total_idr,
			   line_tax_base_idr, line_tax_idr, is_promo, promo_label,
			   price_row_id, price_table)
			VALUES (?,?,1,'PACKAGE',?,1, ?,?,?, ?,?, ?,?, ?,?)`,
			uuid.Must(uuid.NewV7()), orderID, p.Package.ID,
			p.PriceIDR, p.NormalPriceIDR, p.PriceIDR, p.TaxBaseIDR, p.TaxIDR,
			p.IsPromo, p.PromoLabel, nullableUUID(p.PriceRowID), p.PriceTable).Error; err != nil {
			return err
		}

		// PENDING: no credits until the money is confirmed.
		if err := tx.Exec(`
			INSERT INTO customer_package
			  (id, customer_id, package_id, order_id, meal_credits, validity_days,
			   package_name, price_paid_idr, status)
			VALUES (?,?,?,?,?,?,?,?, 'PENDING')`,
			uuid.Must(uuid.NewV7()), p.CustomerID, p.Package.ID, orderID,
			p.Package.MealCredits, p.Package.ValidityDays, p.Package.Name, p.PriceIDR).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO payment (id, order_id, provider, bank_account_id, expected_amount_idr, status)
			VALUES (?,?, 'MANUAL_TRANSFER', ?, ?, 'PENDING')`,
			uuid.Must(uuid.NewV7()), orderID, bankID, payment).Error; err != nil {
			return err
		}

		out = PlaceOrderResult{
			OrderID: orderID, OrderCode: code, TotalIDR: p.PriceIDR,
			PaymentAmountIDR: payment, UniqueCode: uniqueCode,
		}
		return nil
	})
	return out, err
}

// ActivateForOrder issues credits when a package order is paid.
//
// Idempotent by construction: the PURCHASE entry carries the package id as its
// reference, and status only moves PENDING -> ACTIVE, so a double verification
// cannot issue two lots of credits.
func (r *CreditRepo) ActivateForOrder(ctx context.Context, orderID uuid.UUID,
	loc *time.Location, now time.Time) (uuid.UUID, error) {

	var packageID uuid.UUID

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []struct {
			ID           uuid.UUID
			CustomerID   uuid.UUID
			MealCredits  int
			ValidityDays int
			Status       string
		}
		if err := tx.Raw(`
			SELECT id, customer_id, meal_credits, validity_days, status
			  FROM customer_package WHERE order_id = ? FOR UPDATE`, orderID).Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return ErrNotFound
		}
		cp := rows[0]
		if cp.Status != "PENDING" {
			// Already activated; nothing to do and nothing to double-issue.
			packageID = cp.ID
			return nil
		}

		local := now.In(loc)
		expires := time.Date(local.Year(), local.Month(), local.Day()+cp.ValidityDays,
			0, 0, 0, 0, loc)

		if err := tx.Exec(`
			UPDATE customer_package
			   SET status='ACTIVE', activated_at=?, expires_at=?::date
			 WHERE id=? AND status='PENDING'`, now, expires.Format("2006-01-02"), cp.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO credit_ledger
			  (id, customer_id, customer_package_id, entry_type, qty, reference_type, reference_id, note)
			VALUES (?,?,?, 'PURCHASE', ?, 'order', ?, 'package purchased')`,
			uuid.Must(uuid.NewV7()), cp.CustomerID, cp.ID, cp.MealCredits, orderID).Error; err != nil {
			return err
		}
		packageID = cp.ID
		return nil
	})
	return packageID, err
}

// CustomerPackage is a purchased package with its live balance.
type CustomerPackage struct {
	ID           uuid.UUID `json:"id"`
	PackageName  string    `json:"package_name"`
	MealCredits  int       `json:"purchased_credits"`
	Remaining    int       `json:"remaining_credits"`
	Status       string    `json:"status"`
	PurchasedAt  string    `json:"purchased_at"`
	ActivatedAt  *string   `json:"activated_at,omitempty"`
	ExpiresAt    *string   `json:"expires_at,omitempty"`
	PricePaidIDR int64     `json:"price_paid_idr" gorm:"column:price_paid_idr"`
	PricePaid    string    `json:"price_paid"`
}

// ListCustomerPackages returns a customer's packages with balances.
//
// The balance is SUMmed from the ledger every time (PROMPT §7). Nothing caches
// it, so it cannot drift from the movements that produced it.
func (r *CreditRepo) ListCustomerPackages(ctx context.Context, customerID uuid.UUID,
	p ListParams) (Page[CustomerPackage], error) {

	// Newest first by default: a customer opening this screen wants the
	// package they are using now, not the one they finished in March.
	if p.Sort == "" {
		p.Desc = true
	}
	p = p.Normalise("purchased_at", "purchased_at", "expires_at")

	base := r.db.WithContext(ctx).Table("customer_package cp").
		Where("cp.customer_id = ?", customerID)
	if p.Q != "" {
		pattern := SearchPattern(p.Q)
		base = base.Where("lower(cp.package_name) LIKE ? OR lower(cp.status) LIKE ?", pattern, pattern)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[CustomerPackage]{}, fmt.Errorf("postgres: count packages: %w", err)
	}

	var items []CustomerPackage
	err := base.Session(&gorm.Session{}).
		Select(`cp.id, cp.package_name, cp.meal_credits, cp.status,
		        cp.purchased_at::text AS purchased_at, cp.activated_at::text AS activated_at,
		        cp.expires_at::text AS expires_at, cp.price_paid_idr,
		        COALESCE((SELECT SUM(qty) FROM credit_ledger cl
		                   WHERE cl.customer_package_id = cp.id), 0) AS remaining`).
		Order("cp." + p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[CustomerPackage]{}, fmt.Errorf("postgres: list customer packages: %w", err)
	}
	for i := range items {
		items[i].PricePaid = money.Format(money.IDR(items[i].PricePaidIDR))
	}
	return NewPage(items, total, p), nil
}

// LedgerEntry is one movement, for the drill-down the credit report needs.
type LedgerEntry struct {
	ID            uuid.UUID `json:"id"`
	EntryType     string    `json:"entry_type"`
	Qty           int       `json:"qty"`
	Balance       int       `json:"running_balance"`
	ReferenceType string    `json:"reference_type,omitempty"`
	Note          string    `json:"note,omitempty"`
	OccurredAt    string    `json:"occurred_at"`
	CreatedBy     string    `json:"created_by,omitempty"`
}

// Ledger returns every movement for a package, with a running balance.
func (r *CreditRepo) Ledger(ctx context.Context, customerID, packageID uuid.UUID) ([]LedgerEntry, error) {
	out := []LedgerEntry{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT cl.id, cl.entry_type, cl.qty,
		       SUM(cl.qty) OVER (ORDER BY cl.occurred_at, cl.id) AS balance,
		       COALESCE(cl.reference_type,'') AS reference_type, cl.note,
		       cl.occurred_at::text AS occurred_at,
		       COALESCE(u.email::text,'') AS created_by
		  FROM credit_ledger cl
		  LEFT JOIN app_user u ON u.id = cl.created_by
		 WHERE cl.customer_package_id = ? AND cl.customer_id = ?
		 ORDER BY cl.occurred_at, cl.id`, packageID, customerID).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: ledger: %w", err)
	}
	return out, nil
}

// RedeemParams books one meal against a package.
type RedeemParams struct {
	CustomerID  uuid.UUID
	PackageID   uuid.UUID
	MealID      uuid.UUID
	AddressID   uuid.UUID
	ServiceDate time.Time
	SlotID      uuid.UUID
	KitchenID   uuid.UUID
	DistanceM   int
	Reason      string
	Address     Address
	Now         time.Time
}

// Redeem spends one credit and creates the delivery, under a row lock.
//
// The lock is the whole point (PROMPT §7): the balance is read INSIDE it, so
// two tabs cannot both see the last credit. The unique index on
// (reference_type, reference_id) WHERE entry_type='REDEEM' is the second line —
// a retry cannot spend a second credit for the same delivery.
func (r *CreditRepo) Redeem(ctx context.Context, p RedeemParams) (uuid.UUID, error) {
	var deliveryID uuid.UUID

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pkgs []struct {
			ID         uuid.UUID
			CustomerID uuid.UUID
			Status     string
			ExpiresAt  *time.Time
		}
		if err := tx.Raw(`
			SELECT id, customer_id, status, expires_at
			  FROM customer_package
			 WHERE id = ? AND customer_id = ?
			 FOR UPDATE`, p.PackageID, p.CustomerID).Scan(&pkgs).Error; err != nil {
			return err
		}
		if len(pkgs) == 0 {
			return ErrNotFound
		}
		pkg := pkgs[0]

		var balance int
		if err := tx.Raw(`SELECT COALESCE(SUM(qty),0) FROM credit_ledger
		                   WHERE customer_package_id = ?`, pkg.ID).Scan(&balance).Error; err != nil {
			return err
		}

		dom := credit.Package{
			ID: pkg.ID, CustomerID: pkg.CustomerID,
			Status: credit.Status(pkg.Status), ExpiresAt: pkg.ExpiresAt,
		}
		entries := make([]credit.Entry, 0, 1)
		if balance != 0 {
			entries = append(entries, credit.Entry{Type: credit.Purchase, Qty: balance})
		}

		deliveryID = uuid.Must(uuid.NewV7())
		entry, err := credit.RedeemOne(dom, entries, deliveryID, p.ServiceDate, p.Now)
		if err != nil {
			return err
		}

		// Capacity, same guard as a meal order.
		res := tx.Exec(`
			UPDATE scheduled_meal SET qty_reserved = qty_reserved + 1
			 WHERE id = ? AND status='PUBLISHED'
			   AND (qty_capacity IS NULL OR qty_reserved + 1 <= qty_capacity)`, p.MealID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrCapacityFull
		}

		dcode, err := deliveryCode()
		if err != nil {
			return err
		}
		snapshot, err := addressSnapshotJSON(p.Address)
		if err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO delivery
			  (id, delivery_code, customer_id, customer_package_id, service_date, slot_id,
			   address_id, address_snapshot, latitude, longitude, kitchen_id,
			   assigned_distance_m, assignment_mode, assignment_reason, assigned_at,
			   delivery_fee_idr, status, driver_note)
			VALUES (?,?,?,?, ?::date, ?, ?, ?::jsonb, ?, ?, ?, ?, 'AUTO', ?, now(), 0, 'SCHEDULED', ?)`,
			deliveryID, dcode, p.CustomerID, p.PackageID,
			p.ServiceDate.Format("2006-01-02"), p.SlotID, p.AddressID, snapshot,
			p.Address.Latitude, p.Address.Longitude, p.KitchenID, p.DistanceM,
			p.Reason, p.Address.DriverNote).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO delivery_line (id, delivery_id, scheduled_meal_id, diet_type_id, qty, meal_snapshot)
			SELECT ?, ?, m.id, m.diet_type_id, 1, '{}'::jsonb FROM scheduled_meal m WHERE m.id = ?`,
			uuid.Must(uuid.NewV7()), deliveryID, p.MealID).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO credit_ledger
			  (id, customer_id, customer_package_id, entry_type, qty, reference_type, reference_id, note)
			VALUES (?,?,?, 'REDEEM', -1, 'delivery', ?, '')`,
			uuid.Must(uuid.NewV7()), p.CustomerID, p.PackageID, deliveryID).Error; err != nil {
			if strings.Contains(err.Error(), "credit_ledger_redeem_once") {
				return fmt.Errorf("postgres: this delivery already spent a credit")
			}
			return err
		}

		// The status cache follows the ledger, never leads it.
		if balance-1 <= 0 {
			if err := tx.Exec(`UPDATE customer_package SET status='EXHAUSTED' WHERE id=?`,
				pkg.ID).Error; err != nil {
				return err
			}
		}
		_ = entry
		return nil
	})
	return deliveryID, err
}
