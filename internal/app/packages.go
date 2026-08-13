package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/domain/credit"
	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
	"github.com/stevenwilliam/healthy_catering/internal/domain/schedule"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Packages is Flow B: buying credits and booking meals with them (PROMPT §6.2).
type Packages struct {
	credits *postgres.CreditRepo
	orders  *postgres.OrderRepo
	sched   *postgres.ScheduleRepo
	users   *postgres.UserRepo
	pricing *Pricing
	service *Serviceability
	audit   *postgres.AuditRepo
	params  *sysparam.Store
	tz      *time.Location
	now     func() time.Time
}

// PackagesDeps wires the service.
type PackagesDeps struct {
	Credits  *postgres.CreditRepo
	Orders   *postgres.OrderRepo
	Schedule *postgres.ScheduleRepo
	Users    *postgres.UserRepo
	Pricing  *Pricing
	Service  *Serviceability
	Audit    *postgres.AuditRepo
	Params   *sysparam.Store
	TZ       *time.Location
	Now      func() time.Time
}

func NewPackages(d PackagesDeps) *Packages {
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Packages{
		credits: d.Credits, orders: d.Orders, sched: d.Schedule, users: d.Users,
		pricing: d.Pricing, service: d.Service, audit: d.Audit, params: d.Params,
		tz: d.TZ, now: d.Now,
	}
}

// List returns the sellable packages.
func (p *Packages) List(ctx context.Context, lp postgres.ListParams) (postgres.Page[postgres.Package], error) {
	active := true
	lp.Active = &active
	page, err := p.credits.ListPackages(ctx, lp)
	if err != nil {
		return postgres.Page[postgres.Package]{}, apierror.Internal(err)
	}
	return page, nil
}

// Buy purchases a package. Credits are NOT issued here — verification does that.
func (p *Packages) Buy(ctx context.Context, ident Identity, packageID uuid.UUID,
	idempotencyKey, ip, ua string) (PlacedOrder, error) {

	if ident.CustomerID == nil {
		return PlacedOrder{}, apierror.Forbidden(apierror.CodeForbidden,
			"Only customers can buy packages.")
	}
	cust, err := p.users.CustomerByUser(ctx, ident.UserID)
	if err != nil {
		return PlacedOrder{}, apierror.Internal(err)
	}

	pkg, err := p.credits.GetPackage(ctx, packageID)
	if err != nil {
		return PlacedOrder{}, notFoundOr(err, "That package is not available.")
	}

	now := p.now()
	today := schedule.Today(now, p.tz)
	quote, resolved, err := p.pricing.QuotePackage(ctx, cust.CustomerTypeID, packageID, today)
	if err != nil {
		return PlacedOrder{}, err
	}

	// A package has no cut-off to cap against — it buys credits, not a slot on
	// a particular day — so the full payment window applies.
	window := p.params.Duration(ctx, sysparam.KeyPaymentWindow, 2*time.Hour)
	deadline := now.Add(window)

	res, err := p.credits.BuyPackage(ctx, postgres.BuyPackageParams{
		CustomerID: *ident.CustomerID, CustomerTypeID: cust.CustomerTypeID,
		Package: pkg, PriceIDR: quote.UnitPriceIDR, NormalPriceIDR: quote.NormalPriceIDR,
		IsPromo: quote.IsPromo, PromoLabel: quote.PromoLabel,
		PriceRowID: resolved.PriceRowID, PriceTable: resolved.PriceTable,
		TaxBaseIDR: quote.TaxBaseIDR, TaxIDR: quote.TaxIDR, TaxRateBps: quote.TaxRateBps,
		UseUniqueCode: p.params.Bool(ctx, sysparam.KeyUniqueCodeEnabled, true),
		Deadline:      deadline, IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if errors.Is(err, postgres.ErrIdempotentReplay) {
			return PlacedOrder{}, apierror.Conflict(apierror.CodeIdempotencyMismatch,
				"That purchase was already submitted.")
		}
		return PlacedOrder{}, apierror.Internal(err)
	}

	_ = p.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &ident.UserID, ActorEmail: ident.Email, Action: "package.buy",
		EntityType: "customer_order", EntityID: &res.OrderID,
		After: map[string]any{"package": pkg.Name, "price_idr": quote.UnitPriceIDR},
		IP:    ip, UserAgent: ua,
	})

	return PlacedOrder{
		OrderID: res.OrderID, OrderCode: res.OrderCode, Status: "AWAITING_PAYMENT",
		SubtotalIDR: quote.LineTotalIDR, TotalIDR: quote.LineTotalIDR,
		Total:      money.Format(money.IDR(quote.LineTotalIDR)),
		TaxBaseIDR: quote.TaxBaseIDR, TaxIDR: quote.TaxIDR, TaxRateBps: quote.TaxRateBps,
		PaymentAmountIDR: res.PaymentAmountIDR,
		PaymentAmount:    money.Format(money.IDR(res.PaymentAmountIDR)),
		UniqueCode:       res.UniqueCode, PaymentDeadline: deadline,
	}, nil
}

// MyPackages returns the caller's packages with live balances.
func (p *Packages) MyPackages(ctx context.Context, ident Identity,
	lp postgres.ListParams) (postgres.Page[postgres.CustomerPackage], error) {

	if ident.CustomerID == nil {
		return postgres.Page[postgres.CustomerPackage]{}, apierror.Forbidden(
			apierror.CodeForbidden, "Only customers have packages.")
	}
	page, err := p.credits.ListCustomerPackages(ctx, *ident.CustomerID, lp)
	if err != nil {
		return postgres.Page[postgres.CustomerPackage]{}, apierror.Internal(err)
	}
	return page, nil
}

// Ledger returns every movement for one of the caller's packages — the
// drill-down the credit report needs (PROMPT §7).
func (p *Packages) Ledger(ctx context.Context, ident Identity, packageID uuid.UUID) ([]postgres.LedgerEntry, error) {
	if ident.CustomerID == nil {
		return nil, apierror.Forbidden(apierror.CodeForbidden, "Only customers have packages.")
	}
	// Scoped by customer id in the query, so another customer's package id
	// returns an empty ledger rather than someone else's movements.
	entries, err := p.credits.Ledger(ctx, *ident.CustomerID, packageID)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return entries, nil
}

// BookInput picks one meal against a package.
type BookInput struct {
	PackageID       uuid.UUID
	ScheduledMealID uuid.UUID
	AddressID       uuid.UUID
}

// Booked is the result of spending a credit.
type Booked struct {
	DeliveryID  uuid.UUID `json:"delivery_id"`
	ServiceDate string    `json:"service_date"`
	Slot        string    `json:"slot"`
	Kitchen     string    `json:"kitchen"`
	Remaining   int       `json:"remaining_credits"`
	Message     string    `json:"message"`
}

// Book spends ONE credit on ONE meal, whatever that meal contains (D-32).
func (p *Packages) Book(ctx context.Context, ident Identity, in BookInput) (Booked, error) {
	if ident.CustomerID == nil {
		return Booked{}, apierror.Forbidden(apierror.CodeForbidden,
			"Only customers can book meals.")
	}

	meal, err := p.sched.GetMeal(ctx, in.ScheduledMealID, true)
	if err != nil {
		return Booked{}, notFoundOr(err, "That meal is not on the published menu.")
	}
	serviceDate, err := time.ParseInLocation("2006-01-02", meal.ServiceDate, p.tz)
	if err != nil {
		return Booked{}, apierror.Internal(err)
	}

	now := p.now()
	rule := schedule.Rule{
		CutoffTime: p.params.TimeOfDay(ctx, sysparam.KeyCutoffTime, 18*time.Hour),
		LeadDays:   p.params.Int(ctx, sysparam.KeyCutoffLeadDays, 1),
		Location:   p.tz,
	}
	if err := rule.Check(serviceDate, now); err != nil {
		return Booked{}, apierror.Unprocessable(apierror.CodeSlotCutoff,
			"Ordering for "+meal.ServiceDate+" has closed.")
	}

	addr, err := p.orders.AddressForCustomer(ctx, *ident.CustomerID, in.AddressID)
	if err != nil {
		return Booked{}, apierror.NotFound("That delivery address is not one of yours.")
	}

	check, err := p.service.Check(ctx, CheckInput{
		Lat: addr.Latitude, Lng: addr.Longitude, SlotID: meal.SlotID,
		Date: serviceDate, Qty: 1, District: addr.District, City: addr.City,
		Source: "CHECKOUT", CustomerID: ident.CustomerID,
	})
	if err != nil {
		return Booked{}, err
	}
	if !check.Serviceable {
		return Booked{}, apierror.Unprocessable(apierror.CodeSlotNotBookable, check.Message)
	}

	deliveryID, err := p.credits.Redeem(ctx, postgres.RedeemParams{
		CustomerID: *ident.CustomerID, PackageID: in.PackageID, MealID: meal.ID,
		AddressID: addr.ID, ServiceDate: serviceDate, SlotID: meal.SlotID,
		KitchenID: *check.KitchenID, DistanceM: int(check.DistanceKM * 1000),
		Reason: check.Reason, Address: addr, Now: now,
	})
	if err != nil {
		return Booked{}, creditError(err)
	}

	remaining := 0
	if entries, err := p.credits.Ledger(ctx, *ident.CustomerID, in.PackageID); err == nil {
		for _, e := range entries {
			remaining += e.Qty
		}
	}

	return Booked{
		DeliveryID: deliveryID, ServiceDate: meal.ServiceDate, Slot: meal.SlotAlias,
		Kitchen: check.KitchenName, Remaining: remaining,
		Message: "Booked. One credit used.",
	}, nil
}

// creditError maps ledger failures to customer-facing messages.
func creditError(err error) error {
	switch {
	case errors.Is(err, credit.ErrInsufficient):
		return apierror.Unprocessable(apierror.CodeValidation,
			"You have no credits left on this package.")
	case errors.Is(err, credit.ErrExpired):
		return apierror.Unprocessable(apierror.CodeValidation,
			"That package has expired. Unused credits are not refunded — please buy a new package.")
	case errors.Is(err, credit.ErrAfterExpiry):
		return apierror.Unprocessable(apierror.CodeValidation,
			"That delivery date is after your package expires. Choose an earlier date.")
	case errors.Is(err, credit.ErrNotActive):
		return apierror.Unprocessable(apierror.CodeValidation,
			"That package is not active yet — we confirm your transfer first.")
	case errors.Is(err, postgres.ErrCapacityFull):
		return apierror.Conflict(apierror.CodeSlotFull,
			"That meal has just sold out. Please choose another date.")
	case errors.Is(err, postgres.ErrNotFound):
		return apierror.NotFound("That package is not one of yours.")
	default:
		return apierror.Internal(err)
	}
}
