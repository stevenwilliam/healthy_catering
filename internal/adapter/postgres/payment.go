package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
)

// Errors the finance service maps to messages.
var (
	ErrAlreadyVerified = errors.New("postgres: payment already verified")
	ErrWrongState      = errors.New("postgres: not in the required state")
)

// PaymentRepo owns payments, proofs and the verification queue.
type PaymentRepo struct{ db *gorm.DB }

func NewPaymentRepo(db *gorm.DB) *PaymentRepo { return &PaymentRepo{db: db} }

// QueueItem is one row of the finance verification queue.
type QueueItem struct {
	PaymentID      uuid.UUID `json:"payment_id"`
	OrderID        uuid.UUID `json:"order_id"`
	OrderCode      string    `json:"order_code"`
	CustomerName   string    `json:"customer_name"`
	CustomerEmail  string    `json:"customer_email"`
	ExpectedIDR    int64     `json:"expected_amount_idr" gorm:"column:expected_amount_idr"`
	Expected       string    `json:"expected_amount"`
	UniqueCode     *int      `json:"unique_code,omitempty"`
	Status         string    `json:"status"`
	SubmittedAt    *string   `json:"submitted_at,omitempty"`
	ProofCount     int       `json:"proof_count"`
	BankName       string    `json:"bank_name"`
	WaitingMinutes int       `json:"waiting_minutes"`
}

// Queue returns the payments awaiting a decision, oldest first.
//
// Oldest first is deliberate: a queue sorted newest-first quietly starves the
// customer who has been waiting longest, which is exactly the one about to
// telephone.
func (r *PaymentRepo) Queue(ctx context.Context, p ListParams, status string) (Page[QueueItem], error) {
	p = p.Normalise("submitted_at", "submitted_at", "expected_amount_idr")

	base := r.db.WithContext(ctx).Table("payment pay").
		Joins("JOIN customer_order o ON o.id = pay.order_id").
		Joins("JOIN customer c ON c.id = o.customer_id").
		Joins("JOIN app_user u ON u.id = c.user_id").
		Joins("LEFT JOIN bank_account b ON b.id = pay.bank_account_id")

	if status != "" {
		base = base.Where("pay.status = ?", status)
	} else {
		base = base.Where("pay.status = 'SUBMITTED'")
	}
	if p.Q != "" {
		pattern := SearchPattern(p.Q)
		base = base.Where(`lower(o.order_code) LIKE ? OR lower(u.full_name) LIKE ?
		                   OR lower(u.email::text) LIKE ? OR pay.expected_amount_idr::text LIKE ?`,
			pattern, pattern, pattern, pattern)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[QueueItem]{}, fmt.Errorf("postgres: count queue: %w", err)
	}

	var items []QueueItem
	err := base.Session(&gorm.Session{}).
		Select(`pay.id AS payment_id, o.id AS order_id, o.order_code,
		        u.full_name AS customer_name, u.email::text AS customer_email,
		        pay.expected_amount_idr, o.unique_code, pay.status,
		        pay.submitted_at::text AS submitted_at,
		        COALESCE(b.bank_name,'') AS bank_name,
		        (SELECT count(*) FROM payment_proof pp WHERE pp.payment_id = pay.id) AS proof_count,
		        COALESCE(EXTRACT(EPOCH FROM (now() - pay.submitted_at))/60, 0)::int AS waiting_minutes`).
		Order("pay.submitted_at ASC NULLS LAST").
		Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[QueueItem]{}, fmt.Errorf("postgres: queue: %w", err)
	}
	for i := range items {
		items[i].Expected = money.Format(money.IDR(items[i].ExpectedIDR))
	}
	return NewPage(items, total, p), nil
}

// SubmitProof records a transfer proof and moves the order to
// PAYMENT_SUBMITTED, scoped to the owner.
func (r *PaymentRepo) SubmitProof(ctx context.Context, customerID, orderID uuid.UUID,
	objectKey, contentType string, bytes int64, checksum string) error {

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var paymentIDs []uuid.UUID
		if err := tx.Raw(`
			SELECT pay.id FROM payment pay
			  JOIN customer_order o ON o.id = pay.order_id
			 WHERE pay.order_id = ? AND o.customer_id = ?
			   AND o.status IN ('AWAITING_PAYMENT','PAYMENT_SUBMITTED')
			   AND pay.status IN ('PENDING','SUBMITTED','REJECTED')
			 FOR UPDATE OF pay`, orderID, customerID).Scan(&paymentIDs).Error; err != nil {
			return err
		}
		if len(paymentIDs) == 0 {
			return ErrNotFound
		}
		paymentID := paymentIDs[0]

		if err := tx.Exec(`
			INSERT INTO payment_proof (id, payment_id, object_key, content_type, bytes, checksum, uploaded_by)
			VALUES (?,?,?,?,?,NULLIF(?,''), NULL)`,
			uuid.Must(uuid.NewV7()), paymentID, objectKey, contentType, bytes, checksum).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE payment SET status='SUBMITTED', submitted_at=now(),
			       rejection_reason=NULL, rejected_at=NULL, rejected_by=NULL
			 WHERE id=?`, paymentID).Error; err != nil {
			return err
		}
		return tx.Exec(`
			UPDATE customer_order SET status='PAYMENT_SUBMITTED'
			 WHERE id=? AND status='AWAITING_PAYMENT'`, orderID).Error
	})
}

// VerifyResult reports what a verification did.
type VerifyResult struct {
	OrderID    uuid.UUID
	OrderType  string
	CustomerID uuid.UUID
	PackageID  *uuid.UUID
}

// Verify marks a payment good and the order PAID, in one transaction.
//
// The row is locked and re-checked inside the lock: two finance users clicking
// verify at the same moment must not both succeed, or a package order would
// issue its credits twice.
func (r *PaymentRepo) Verify(ctx context.Context, paymentID, by uuid.UUID, paidAmount *int64) (VerifyResult, error) {
	var out VerifyResult

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []struct {
			OrderID    uuid.UUID
			Status     string
			OrderType  string
			CustomerID uuid.UUID
			Expected   int64 `gorm:"column:expected_amount_idr"`
		}
		if err := tx.Raw(`
			SELECT pay.order_id, pay.status, o.order_type, o.customer_id,
			       pay.expected_amount_idr
			  FROM payment pay JOIN customer_order o ON o.id = pay.order_id
			 WHERE pay.id = ? FOR UPDATE OF pay`, paymentID).Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return ErrNotFound
		}
		p := rows[0]
		if p.Status == "VERIFIED" {
			return ErrAlreadyVerified
		}
		if p.Status != "SUBMITTED" {
			return ErrWrongState
		}

		amount := p.Expected
		if paidAmount != nil {
			amount = *paidAmount
		}
		if err := tx.Exec(`
			UPDATE payment SET status='VERIFIED', verified_at=now(), verified_by=?,
			       paid_amount_idr=? WHERE id=?`, by, amount, paymentID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE customer_order SET status='PAID', paid_at=now()
			 WHERE id=? AND status='PAYMENT_SUBMITTED'`, p.OrderID).Error; err != nil {
			return err
		}

		out = VerifyResult{OrderID: p.OrderID, OrderType: p.OrderType, CustomerID: p.CustomerID}
		return nil
	})
	return out, err
}

// Reject sends a payment back with a reason.
func (r *PaymentRepo) Reject(ctx context.Context, paymentID, by uuid.UUID, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var orderIDs []uuid.UUID
		if err := tx.Raw(`SELECT order_id FROM payment
		                   WHERE id=? AND status='SUBMITTED' FOR UPDATE`, paymentID).Scan(&orderIDs).Error; err != nil {
			return err
		}
		if len(orderIDs) == 0 {
			return ErrWrongState
		}
		if err := tx.Exec(`
			UPDATE payment SET status='REJECTED', rejected_at=now(), rejected_by=?,
			       rejection_reason=? WHERE id=?`, by, reason, paymentID).Error; err != nil {
			return err
		}
		// Back to AWAITING_PAYMENT, not cancelled: the customer can upload a
		// corrected proof, and the capacity they hold is still theirs until
		// the deadline.
		return tx.Exec(`UPDATE customer_order SET status='AWAITING_PAYMENT'
		                 WHERE id=? AND status='PAYMENT_SUBMITTED'`, orderIDs[0]).Error
	})
}

// ExpireUnpaid releases orders whose deadline has passed, returning the
// capacity they held.
//
// This is the ONLY automated cancellation in the system, and it touches unpaid
// orders only (99 §8, docs/03 Q-14). Nothing automatic ever cancels a paid
// order or a scheduled delivery.
func (r *PaymentRepo) ExpireUnpaid(ctx context.Context, now time.Time) (int, error) {
	var expired int

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var orders []struct {
			ID uuid.UUID
		}
		if err := tx.Raw(`
			SELECT id FROM customer_order
			 WHERE status IN ('AWAITING_PAYMENT','PAYMENT_SUBMITTED')
			   AND payment_deadline_at < ?
			 FOR UPDATE SKIP LOCKED`, now).Scan(&orders).Error; err != nil {
			return err
		}
		for _, o := range orders {
			// Give back the meal capacity this order was holding.
			if err := tx.Exec(`
				UPDATE scheduled_meal m
				   SET qty_reserved = GREATEST(0, m.qty_reserved - agg.qty)
				  FROM (SELECT scheduled_meal_id, SUM(qty) AS qty
				          FROM order_line WHERE order_id = ? AND scheduled_meal_id IS NOT NULL
				         GROUP BY scheduled_meal_id) agg
				 WHERE m.id = agg.scheduled_meal_id`, o.ID).Error; err != nil {
				return err
			}
			// And the kitchen capacity.
			if err := tx.Exec(`
				UPDATE kitchen_capacity kc
				   SET reserved_portions = GREATEST(0, kc.reserved_portions - agg.qty)
				  FROM (SELECT d.kitchen_id, d.service_date, d.slot_id, SUM(dl.qty) AS qty
				          FROM delivery d JOIN delivery_line dl ON dl.delivery_id = d.id
				         WHERE d.order_id = ?
				         GROUP BY d.kitchen_id, d.service_date, d.slot_id) agg
				 WHERE kc.kitchen_id = agg.kitchen_id
				   AND kc.service_date = agg.service_date
				   AND kc.slot_id = agg.slot_id`, o.ID).Error; err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE delivery SET status='CANCELLED'
			                    WHERE order_id=? AND status='SCHEDULED'`, o.ID).Error; err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE payment SET status='EXPIRED'
			                    WHERE order_id=? AND status IN ('PENDING','SUBMITTED')`, o.ID).Error; err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE customer_order SET status='EXPIRED'
			                    WHERE id=?`, o.ID).Error; err != nil {
				return err
			}
			expired++
		}
		return nil
	})
	return expired, err
}

// BankAccounts returns the active accounts, for payment instructions.
func (r *PaymentRepo) BankAccounts(ctx context.Context) ([]BankAccount, error) {
	out := []BankAccount{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, bank_name, account_number, account_holder, branch
		  FROM bank_account WHERE is_active ORDER BY sort_order`).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: bank accounts: %w", err)
	}
	return out, nil
}

// BankAccount is a destination for a transfer.
type BankAccount struct {
	ID            uuid.UUID `json:"id"`
	BankName      string    `json:"bank_name"`
	AccountNumber string    `json:"account_number"`
	AccountHolder string    `json:"account_holder"`
	Branch        string    `json:"branch,omitempty"`
}
