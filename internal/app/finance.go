package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
)

// Finance is the payment verification queue and the expiry sweep.
type Finance struct {
	payments *postgres.PaymentRepo
	credits  *postgres.CreditRepo
	audit    *postgres.AuditRepo
	tz       *time.Location
	now      func() time.Time
	// OnPaid is called after a payment is verified, so notifications can be
	// sent without the finance path depending on a mailer.
	OnPaid func(ctx context.Context, orderID, customerID uuid.UUID, orderType string)
}

func NewFinance(p *postgres.PaymentRepo, c *postgres.CreditRepo, a *postgres.AuditRepo,
	tz *time.Location, now func() time.Time) *Finance {
	if now == nil {
		now = time.Now
	}
	return &Finance{payments: p, credits: c, audit: a, tz: tz, now: now}
}

func (f *Finance) Queue(ctx context.Context, p postgres.ListParams, status string) (postgres.Page[postgres.QueueItem], error) {
	if status != "" {
		if _, err := sanitize.Enum("status", status,
			"PENDING", "SUBMITTED", "VERIFIED", "REJECTED", "EXPIRED", "REFUNDED"); err != nil {
			return postgres.Page[postgres.QueueItem]{}, validationFrom(err)
		}
	}
	page, err := f.payments.Queue(ctx, p, status)
	if err != nil {
		return postgres.Page[postgres.QueueItem]{}, apierror.Internal(err)
	}
	return page, nil
}

// Verify confirms a transfer and, for a package order, issues the credits.
func (f *Finance) Verify(ctx context.Context, paymentID uuid.UUID, paidAmount *int64, by Actor) (map[string]any, error) {
	res, err := f.payments.Verify(ctx, paymentID, by.UserID, paidAmount)
	switch {
	case errors.Is(err, postgres.ErrNotFound):
		return nil, apierror.NotFound("No such payment.")
	case errors.Is(err, postgres.ErrAlreadyVerified):
		// Not an error the user caused — say so plainly rather than implying
		// they did something wrong.
		return nil, apierror.Conflict(apierror.CodePaymentAlreadyVerified,
			"This payment was already verified, possibly by a colleague.")
	case errors.Is(err, postgres.ErrWrongState):
		return nil, apierror.Conflict(apierror.CodeIllegalTransition,
			"That payment is not awaiting verification.")
	case err != nil:
		return nil, apierror.Internal(err)
	}

	out := map[string]any{"order_id": res.OrderID, "status": "PAID"}

	if res.OrderType == "PACKAGE" {
		// The active period starts NOW, on verification, not at purchase
		// (docs/02 D-14).
		pkgID, err := f.credits.ActivateForOrder(ctx, res.OrderID, f.tz, f.now())
		if err != nil {
			// The money is confirmed; failing the whole request would leave
			// finance unsure whether to retry. Report it loudly instead.
			return nil, apierror.Internal(err)
		}
		out["customer_package_id"] = pkgID
		out["credits_issued"] = true
	}

	_ = f.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &by.UserID, ActorEmail: by.Email, Action: "payment.verify",
		EntityType: "payment", EntityID: &paymentID,
		After: out, IP: by.IP, UserAgent: by.UA,
	})

	if f.OnPaid != nil {
		f.OnPaid(ctx, res.OrderID, res.CustomerID, res.OrderType)
	}
	return out, nil
}

// Reject returns a payment to the customer with a reason.
func (f *Finance) Reject(ctx context.Context, paymentID uuid.UUID, reason string, by Actor) error {
	clean, err := sanitize.Required("reason", reason, 500)
	if err != nil {
		// A rejection with no reason leaves the customer with nothing to fix
		// (PROMPT §10).
		return apierror.Validation("Tell the customer why, so they can correct it.",
			map[string]any{"reason": "required"})
	}
	err = f.payments.Reject(ctx, paymentID, by.UserID, clean)
	switch {
	case errors.Is(err, postgres.ErrWrongState):
		return apierror.Conflict(apierror.CodeIllegalTransition,
			"That payment is not awaiting verification.")
	case err != nil:
		return apierror.Internal(err)
	}
	_ = f.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &by.UserID, ActorEmail: by.Email, Action: "payment.reject",
		EntityType: "payment", EntityID: &paymentID, Reason: clean,
		IP: by.IP, UserAgent: by.UA,
	})
	return nil
}

// ExpireUnpaid releases orders past their payment deadline.
//
// The only automated cancellation in the system, and only for orders where
// nothing has been promised (99 §8).
func (f *Finance) ExpireUnpaid(ctx context.Context) (int, error) {
	n, err := f.payments.ExpireUnpaid(ctx, f.now())
	if err != nil {
		return 0, apierror.Internal(err)
	}
	return n, nil
}

// ProofKeys returns the object keys for a payment's proofs, for presigning.
func (f *Finance) ProofKeys(ctx context.Context, paymentID uuid.UUID) ([]string, error) {
	keys, err := f.payments.ProofKeys(ctx, paymentID)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return keys, nil
}

// BankAccounts returns the accounts shown in payment instructions.
func (f *Finance) BankAccounts(ctx context.Context) ([]postgres.BankAccount, error) {
	list, err := f.payments.BankAccounts(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return list, nil
}
