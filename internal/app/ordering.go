package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
	"github.com/stevenwilliam/healthy_catering/internal/domain/order"
	"github.com/stevenwilliam/healthy_catering/internal/domain/schedule"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Ordering is the cart and checkout use cases (PROMPT §6).
type Ordering struct {
	orders     *postgres.OrderRepo
	payments   *postgres.PaymentRepo
	deliveries *postgres.DeliveryRepo
	notifier   *Notifier
	sched      *postgres.ScheduleRepo
	kitchens   *postgres.KitchenRepo
	users      *postgres.UserRepo
	pricing    *Pricing
	service    *Serviceability
	audit      *postgres.AuditRepo
	params     *sysparam.Store
	tz         *time.Location
	now        func() time.Time
}

// OrderingDeps wires the service.
type OrderingDeps struct {
	Orders     *postgres.OrderRepo
	Payments   *postgres.PaymentRepo
	Deliveries *postgres.DeliveryRepo
	Notifier   *Notifier
	Schedule   *postgres.ScheduleRepo
	Kitchens   *postgres.KitchenRepo
	Users      *postgres.UserRepo
	Pricing    *Pricing
	Service    *Serviceability
	Audit      *postgres.AuditRepo
	Params     *sysparam.Store
	TZ         *time.Location
	Now        func() time.Time
}

func NewOrdering(d OrderingDeps) *Ordering {
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Ordering{
		orders: d.Orders, payments: d.Payments, deliveries: d.Deliveries,
		notifier: d.Notifier, sched: d.Schedule, kitchens: d.Kitchens, users: d.Users,
		pricing: d.Pricing, service: d.Service, audit: d.Audit, params: d.Params,
		tz: d.TZ, now: d.Now,
	}
}

// QuoteQuery is a price question from the cart.
type QuoteQuery struct {
	DietTypeID *uuid.UUID
	PackageID  *uuid.UUID
	Qty        int
	Date       string
}

// QuoteFor prices an item for the caller's customer type.
func (o *Ordering) QuoteFor(ctx context.Context, ident Identity, q QuoteQuery) (Quote, error) {
	if ident.CustomerID == nil {
		return Quote{}, apierror.Forbidden(apierror.CodeForbidden,
			"Only customers can be quoted a price.")
	}
	cust, err := o.users.CustomerByUser(ctx, ident.UserID)
	if err != nil {
		return Quote{}, apierror.Internal(err)
	}

	date, err := o.businessDate(q.Date)
	if err != nil {
		return Quote{}, err
	}

	switch {
	case q.PackageID != nil:
		quote, _, err := o.pricing.QuotePackage(ctx, cust.CustomerTypeID, *q.PackageID, date)
		return quote, err
	case q.DietTypeID != nil:
		quote, _, err := o.pricing.QuoteMeals(ctx, cust.CustomerTypeID, *q.DietTypeID, q.Qty, date)
		return quote, err
	default:
		return Quote{}, apierror.Validation(
			"Ask for a price with either diet_type_id or package_id.", nil)
	}
}

// CartLine is one requested meal.
type CartLine struct {
	ScheduledMealID uuid.UUID
	Qty             int
	AddressID       uuid.UUID
}

// PlaceOrderInput is a checkout.
type PlaceOrderInput struct {
	Lines          []CartLine
	IdempotencyKey string
	IP             string
	UA             string
	// DriverNote is the note typed at CHECKOUT (artboard 04), which is a
	// different thing from the note saved on the address: "leave with the
	// receptionist" is a property of the address, "I'm in meeting room 3
	// today" is a property of this delivery. Empty falls back to the
	// address's, so a customer who types nothing keeps their standing note.
	DriverNote string
}

// PlacedOrder is what checkout returns.
type PlacedOrder struct {
	OrderID          uuid.UUID        `json:"order_id"`
	OrderCode        string           `json:"order_code"`
	Status           string           `json:"status"`
	SubtotalIDR      int64            `json:"subtotal_idr"`
	DeliveryFeeIDR   int64            `json:"delivery_fee_idr"`
	TotalIDR         int64            `json:"total_idr"`
	Total            string           `json:"total"`
	TaxBaseIDR       int64            `json:"tax_base_idr"`
	TaxIDR           int64            `json:"tax_idr"`
	TaxRateBps       int              `json:"tax_rate_bps"`
	PaymentAmountIDR int64            `json:"payment_amount_idr"`
	PaymentAmount    string           `json:"payment_amount"`
	UniqueCode       int              `json:"unique_code,omitempty"`
	PaymentDeadline  time.Time        `json:"payment_deadline"`
	Deliveries       []PlacedDelivery `json:"deliveries"`
}

// PlacedDelivery is one delivery created by the order.
type PlacedDelivery struct {
	ID          uuid.UUID `json:"id"`
	ServiceDate string    `json:"service_date"`
	Slot        string    `json:"slot"`
	Kitchen     string    `json:"kitchen"`
	Reason      string    `json:"assignment_reason"`
}

// PlaceOrder is Flow A: a cart of meals becomes an unpaid order with its
// deliveries routed (PROMPT §6.1).
//
// Everything happens in ONE transaction: pricing snapshots, capacity on both
// counters, the order, its lines and its deliveries. A checkout that reserved
// capacity but failed to create the order would hold a slot nobody can buy.
func (o *Ordering) PlaceOrder(ctx context.Context, ident Identity, in PlaceOrderInput) (PlacedOrder, error) {
	if ident.CustomerID == nil {
		return PlacedOrder{}, apierror.Forbidden(apierror.CodeForbidden,
			"Only customers can place orders.")
	}
	if len(in.Lines) == 0 {
		return PlacedOrder{}, apierror.Validation("Your cart is empty.", nil)
	}
	if len(in.Lines) > 100 {
		return PlacedOrder{}, apierror.Validation("That is too many lines for one order.", nil)
	}

	cust, err := o.users.CustomerByUser(ctx, ident.UserID)
	if err != nil {
		return PlacedOrder{}, apierror.Internal(err)
	}

	maxQty := o.params.Int(ctx, sysparam.KeyMaxQtyPerLine, 999)
	taxBps := o.params.Int(ctx, sysparam.KeyTaxRateBps, 0)
	now := o.now()

	// ── Resolve every line before touching the database for writes ──────────
	prepared := make([]postgres.PreparedLine, 0, len(in.Lines))
	totalQty := 0
	var earliestCutoff time.Time

	for i, line := range in.Lines {
		if line.Qty < 1 || line.Qty > maxQty {
			return PlacedOrder{}, apierror.Validation(
				fmt.Sprintf("Line %d: quantity must be between 1 and %d.", i+1, maxQty),
				map[string]any{"qty": fmt.Sprintf("1–%d", maxQty)})
		}
		totalQty += line.Qty

		meal, err := o.sched.GetMeal(ctx, line.ScheduledMealID, true)
		if err != nil {
			if errors.Is(err, postgres.ErrNotFound) {
				return PlacedOrder{}, apierror.Unprocessable(apierror.CodeItemUnavailable,
					fmt.Sprintf("Line %d: that meal is not on the published menu.", i+1))
			}
			return PlacedOrder{}, apierror.Internal(err)
		}

		serviceDate, err := time.ParseInLocation("2006-01-02", meal.ServiceDate, o.tz)
		if err != nil {
			return PlacedOrder{}, apierror.Internal(err)
		}

		// Cut-off, server-side, against the server clock (PROMPT §6).
		rule := o.cutoffRule(ctx)
		if err := rule.Check(serviceDate, now); err != nil {
			switch {
			case errors.Is(err, schedule.ErrPastCutoff):
				return PlacedOrder{}, apierror.Unprocessable(apierror.CodeSlotCutoff,
					fmt.Sprintf("Line %d: ordering for %s has closed.", i+1, meal.ServiceDate))
			default:
				return PlacedOrder{}, apierror.Unprocessable(apierror.CodeSlotPast,
					fmt.Sprintf("Line %d: %s is in the past.", i+1, meal.ServiceDate))
			}
		}
		if c := rule.CutoffFor(serviceDate); earliestCutoff.IsZero() || c.Before(earliestCutoff) {
			earliestCutoff = c
		}

		addr, err := o.orders.AddressForCustomer(ctx, *ident.CustomerID, line.AddressID)
		if err != nil {
			// Ownership is checked in the repository, by customer id — holding
			// address.manage does not say WHICH address is yours (IDOR).
			return PlacedOrder{}, apierror.NotFound(
				fmt.Sprintf("Line %d: that delivery address is not one of yours.", i+1))
		}

		prepared = append(prepared, postgres.PreparedLine{
			Meal: meal, Qty: line.Qty, Address: addr, ServiceDate: serviceDate,
		})
	}

	// ── Price on the ORDER's total quantity (docs/02 B-11) ──────────────────
	// Mon–Fri × 2 meals is ten meals and reaches the 10–19 tier, which is more
	// generous than pricing each date separately and is what customers assume.
	for i := range prepared {
		quote, resolved, err := o.pricing.QuoteMeals(ctx, cust.CustomerTypeID,
			prepared[i].Meal.DietTypeID, totalQty, prepared[i].ServiceDate)
		if err != nil {
			return PlacedOrder{}, err
		}
		unit := money.IDR(quote.UnitPriceIDR)
		lineTotal, err := money.Multiply(unit, prepared[i].Qty)
		if err != nil {
			return PlacedOrder{}, apierror.Internal(err)
		}
		split, err := money.SplitInclusive(lineTotal, taxBps)
		if err != nil {
			return PlacedOrder{}, apierror.Internal(err)
		}
		prepared[i].UnitPrice = unit
		prepared[i].NormalPrice = money.IDR(quote.NormalPriceIDR)
		prepared[i].LineTotal = lineTotal
		prepared[i].Split = split
		prepared[i].IsPromo = quote.IsPromo
		prepared[i].PromoLabel = quote.PromoLabel
		prepared[i].PriceRowID = resolved.PriceRowID
		prepared[i].PriceTable = resolved.PriceTable
		prepared[i].TierID = resolved.TierID
		prepared[i].Trace = resolved.Trace
	}

	// ── Route every delivery, and price the fee from the assigned kitchen ────
	deliveries := map[string]*postgres.PreparedDelivery{}
	for i := range prepared {
		p := &prepared[i]
		key := fmt.Sprintf("%s|%s|%s", p.Meal.ServiceDate, p.Meal.SlotID, p.Address.ID)
		if _, ok := deliveries[key]; !ok {
			check, err := o.service.Check(ctx, CheckInput{
				Lat: p.Address.Latitude, Lng: p.Address.Longitude,
				SlotID: p.Meal.SlotID, Date: p.ServiceDate, Qty: p.Qty,
				District: p.Address.District, City: p.Address.City,
				Source: "CHECKOUT", CustomerID: ident.CustomerID,
			})
			if err != nil {
				return PlacedOrder{}, err
			}
			if !check.Serviceable {
				return PlacedOrder{}, apierror.Unprocessable(apierror.CodeSlotNotBookable,
					fmt.Sprintf("We cannot deliver to %s on %s — %s",
						p.Address.Label, p.Meal.ServiceDate, check.Message))
			}
			deliveries[key] = &postgres.PreparedDelivery{
				ServiceDate: p.ServiceDate, SlotID: p.Meal.SlotID, SlotAlias: p.Meal.SlotAlias,
				Address: p.Address, KitchenID: *check.KitchenID, KitchenName: check.KitchenName,
				DistanceM: int(check.DistanceKM * 1000), Reason: check.Reason,
				FeeIDR: check.DeliveryFee, DriverNote: in.DriverNote,
			}
		}
		deliveries[key].Lines = append(deliveries[key].Lines, i)
	}

	// The fee is charged ONCE per delivery, not per line: two meals in one drop
	// is one journey.
	var deliveryFee money.IDR
	for _, d := range deliveries {
		deliveryFee += d.FeeIDR
	}

	// ── Payment deadline, capped at the cut-off (docs/02 D-13) ──────────────
	window := o.params.Duration(ctx, sysparam.KeyPaymentWindow, 2*time.Hour)
	deadline := schedule.PaymentDeadline(now, window, earliestCutoff)
	if deadline.Sub(now) < 15*time.Minute {
		return PlacedOrder{}, apierror.Unprocessable(apierror.CodeSlotCutoff,
			fmt.Sprintf("There is not enough time left before the cut-off to complete a transfer "+
				"(%d minutes). Please order for a later date.", int(deadline.Sub(now).Minutes())))
	}

	lines := make([]order.Line, 0, len(prepared))
	for _, p := range prepared {
		lines = append(lines, order.Line{LineTotal: p.LineTotal, Split: p.Split})
	}

	useUniqueCode := o.params.Bool(ctx, sysparam.KeyUniqueCodeEnabled, true)
	totals, err := order.Compute(lines, deliveryFee, 0, taxBps, 0)
	if err != nil {
		return PlacedOrder{}, apierror.Internal(err)
	}

	res, err := o.orders.PlaceOrder(ctx, postgres.PlaceOrderParams{
		CustomerID:     *ident.CustomerID,
		CustomerTypeID: cust.CustomerTypeID,
		Lines:          prepared,
		Deliveries:     deliveries,
		Totals:         totals,
		UseUniqueCode:  useUniqueCode,
		Deadline:       deadline,
		IdempotencyKey: in.IdempotencyKey,
		TaxRateBps:     taxBps,
	})
	if err != nil {
		return PlacedOrder{}, orderError(err)
	}

	_ = o.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &ident.UserID, ActorEmail: ident.Email, Action: "order.place",
		EntityType: "customer_order", EntityID: &res.OrderID,
		After: map[string]any{"total_idr": res.TotalIDR, "lines": len(prepared)},
		IP:    in.IP, UserAgent: in.UA,
	})

	// The customer is told how to pay. Queued, never sent inline: an SMTP
	// timeout must not fail an order that already reserved capacity.
	if o.notifier != nil {
		email, name, locale := o.users.ContactForCustomer(ctx, *ident.CustomerID)
		bank, _ := o.payments.BankAccounts(ctx)
		bankName, bankNo, bankHolder := "", "", ""
		if len(bank) > 0 {
			bankName, bankNo, bankHolder = bank[0].BankName, bank[0].AccountNumber, bank[0].AccountHolder
		}
		o.notifier.OrderPlaced(ctx, email, name, locale, res.OrderID, map[string]any{
			"Name": name, "OrderCode": res.OrderCode,
			"PaymentAmount": money.Format(money.IDR(res.PaymentAmountIDR)),
			"BankName":      bankName, "BankAccount": bankNo, "BankHolder": bankHolder,
			"Deadline": deadline.In(o.tz).Format("2 January 2006, 15:04 WIB"),
		})
	}

	out := PlacedOrder{
		OrderID: res.OrderID, OrderCode: res.OrderCode, Status: string(order.AwaitingPayment),
		SubtotalIDR: int64(totals.Subtotal), DeliveryFeeIDR: int64(totals.DeliveryFee),
		TotalIDR: int64(totals.Total), Total: money.Format(totals.Total),
		TaxBaseIDR: int64(totals.TaxBase), TaxIDR: int64(totals.Tax), TaxRateBps: taxBps,
		PaymentAmountIDR: res.PaymentAmountIDR,
		PaymentAmount:    money.Format(money.IDR(res.PaymentAmountIDR)),
		UniqueCode:       res.UniqueCode,
		PaymentDeadline:  deadline,
	}
	for _, d := range res.Deliveries {
		out.Deliveries = append(out.Deliveries, PlacedDelivery{
			ID: d.ID, ServiceDate: d.ServiceDate, Slot: d.Slot,
			Kitchen: d.Kitchen, Reason: d.Reason,
		})
	}
	return out, nil
}

// cutoffRule reads the configured cut-off.
func (o *Ordering) cutoffRule(ctx context.Context) schedule.Rule {
	return schedule.Rule{
		CutoffTime: o.params.TimeOfDay(ctx, sysparam.KeyCutoffTime, 18*time.Hour),
		LeadDays:   o.params.Int(ctx, sysparam.KeyCutoffLeadDays, 1),
		Location:   o.tz,
	}
}

func (o *Ordering) businessDate(v string) (time.Time, error) {
	if v == "" {
		now := o.now().In(o.tz)
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, o.tz), nil
	}
	t, err := time.ParseInLocation("2006-01-02", v, o.tz)
	if err != nil {
		return time.Time{}, apierror.Validation("That date is not valid.",
			map[string]any{"date": "YYYY-MM-DD"})
	}
	return t, nil
}

// orderError maps repository failures to customer-facing messages.
func orderError(err error) error {
	switch {
	case errors.Is(err, postgres.ErrCapacityFull):
		return apierror.Conflict(apierror.CodeSlotFull,
			"That meal has just sold out. Please choose another date or diet type.")
	case errors.Is(err, postgres.ErrKitchenFull):
		return apierror.Conflict(apierror.CodeSlotFull,
			"This slot is full at your nearest kitchen. Please choose another date or time.")
	case errors.Is(err, postgres.ErrIdempotentReplay):
		return apierror.Conflict(apierror.CodeIdempotencyMismatch,
			"That order was already submitted.")
	default:
		return apierror.Internal(err)
	}
}
