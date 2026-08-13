package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/domain/order"
	"github.com/stevenwilliam/healthy_catering/internal/domain/schedule"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Fulfilment is the kitchen and courier side of a delivery (PROMPT §6.3).
type Fulfilment struct {
	deliveries *postgres.DeliveryRepo
	credits    *postgres.CreditRepo
	audit      *postgres.AuditRepo
	params     *sysparam.Store
	tz         *time.Location
	now        func() time.Time
}

func NewFulfilment(d *postgres.DeliveryRepo, c *postgres.CreditRepo,
	a *postgres.AuditRepo, p *sysparam.Store, tz *time.Location, now func() time.Time) *Fulfilment {
	if now == nil {
		now = time.Now
	}
	return &Fulfilment{deliveries: d, credits: c, audit: a, params: p, tz: tz, now: now}
}

// List returns the deliveries a staff user may see, kitchen-scoped.
func (f *Fulfilment) List(ctx context.Context, ident Identity,
	p postgres.ListParams, from, to, status string) (postgres.Page[postgres.DeliveryRow], error) {

	q := postgres.DeliveryQuery{Params: p, Status: status}
	if from != "" {
		t, err := time.ParseInLocation("2006-01-02", from, f.tz)
		if err != nil {
			return postgres.Page[postgres.DeliveryRow]{},
				apierror.Validation("from must be YYYY-MM-DD.", nil)
		}
		q.From = &t
	}
	if to != "" {
		t, err := time.ParseInLocation("2006-01-02", to, f.tz)
		if err != nil {
			return postgres.Page[postgres.DeliveryRow]{},
				apierror.Validation("to must be YYYY-MM-DD.", nil)
		}
		q.To = &t
	}
	// A kitchen-scoped user sees only their kitchen, enforced here rather than
	// trusted from the request (docs/02 D-21).
	q.KitchenID = ident.KitchenID

	page, err := f.deliveries.List(ctx, q)
	if err != nil {
		return postgres.Page[postgres.DeliveryRow]{}, apierror.Internal(err)
	}
	return page, nil
}

// Advance moves a delivery along its fulfilment lifecycle.
//
// The transition is validated by the DOMAIN machine, so an illegal move — a
// delivery jumping from SCHEDULED straight to DELIVERED without ever being
// cooked — is refused in one place rather than per handler.
func (f *Fulfilment) Advance(ctx context.Context, ident Identity, id uuid.UUID,
	to string, reason string, by Actor) error {

	next, err := sanitize.Enum("status", to,
		string(order.Preparing), string(order.OutForDelivery),
		string(order.Delivered), string(order.Failed))
	if err != nil {
		return validationFrom(err)
	}

	current, kitchenID, err := f.deliveries.StatusOf(ctx, id)
	if err != nil {
		return notFoundOr(err, "No such delivery.")
	}

	// A kitchen-scoped user cannot touch another kitchen's delivery even by id.
	if ident.KitchenID != nil && (kitchenID == nil || *kitchenID != *ident.KitchenID) {
		return apierror.Forbidden(apierror.CodeForbidden,
			"That delivery belongs to another kitchen.")
	}

	if err := order.TransitionDelivery(order.DeliveryStatus(current),
		order.DeliveryStatus(next)); err != nil {
		return apierror.Conflict(apierror.CodeIllegalTransition,
			"A delivery cannot go from "+current+" to "+next+".")
	}

	if next == string(order.Failed) {
		clean, err := sanitize.Required("reason", reason, 500)
		if err != nil {
			// A failed delivery with no reason cannot be actioned by anyone.
			return apierror.Validation("Say what went wrong — nobody home, wrong address, damaged.",
				map[string]any{"reason": "required"})
		}
		reason = clean
	}

	if err := f.deliveries.Advance(ctx, id, next, reason, by.UserID); err != nil {
		return apierror.Internal(err)
	}

	f.log(ctx, by, "delivery."+strings.ToLower(next), id, current, next, reason)
	return nil
}

// Skip cancels one delivery, returning the credit only before the cut-off.
//
// PROMPT §8.3: after the cut-off the food is committed, so the credit is not
// returned. Nothing here is automatic — a human asked for it (99 §8).
func (f *Fulfilment) Skip(ctx context.Context, ident Identity, id uuid.UUID,
	reason string, by Actor) (map[string]any, error) {

	d, err := f.deliveries.Get(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "No such delivery.")
	}

	// A customer may skip only their own; staff may skip on their behalf.
	if ident.CustomerID != nil && d.CustomerID != *ident.CustomerID {
		return nil, apierror.NotFound("No such delivery.")
	}
	if d.Status != string(order.Scheduled) {
		return nil, apierror.Conflict(apierror.CodeIllegalTransition,
			"Only a scheduled delivery can be skipped.")
	}

	serviceDate, err := time.ParseInLocation("2006-01-02", d.ServiceDate, f.tz)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	rule := schedule.Rule{
		CutoffTime: f.params.TimeOfDay(ctx, sysparam.KeyCutoffTime, 18*time.Hour),
		LeadDays:   f.params.Int(ctx, sysparam.KeyCutoffLeadDays, 1),
		Location:   f.tz,
	}
	beforeCutoff := rule.IsOpen(serviceDate, f.now())

	creditReturned, err := f.deliveries.Skip(ctx, id, reason, beforeCutoff, by.UserID)
	if err != nil {
		return nil, apierror.Internal(err)
	}

	f.log(ctx, by, "delivery.skip", id, d.Status, string(order.Skipped), reason)

	out := map[string]any{"status": "SKIPPED", "credit_returned": creditReturned}
	if creditReturned {
		out["message"] = "Skipped. Your credit has been returned."
	} else {
		out["message"] = "Skipped. The cut-off has passed, so the credit is not returned — " +
			"this meal is already being prepared."
	}
	return out, nil
}

// Reassign moves a delivery to another kitchen by hand.
//
// A manual assignment is NEVER overwritten by the auto-router (PROMPT §9.3),
// and the reason is mandatory because staff will be asked why in a week's time.
func (f *Fulfilment) Reassign(ctx context.Context, id, kitchenID uuid.UUID,
	reason string, by Actor) error {

	clean, err := sanitize.Required("reason", reason, 500)
	if err != nil {
		return apierror.Validation("Say why this is moving kitchens.",
			map[string]any{"reason": "required"})
	}

	current, _, err := f.deliveries.StatusOf(ctx, id)
	if err != nil {
		return notFoundOr(err, "No such delivery.")
	}
	if current != string(order.Scheduled) {
		return apierror.Conflict(apierror.CodeIllegalTransition,
			"Only a scheduled delivery can be reassigned — this one is already "+current+".")
	}

	if err := f.deliveries.Reassign(ctx, id, kitchenID, clean, by.UserID); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return apierror.NotFound("No such kitchen.")
		}
		return apierror.Internal(err)
	}
	f.log(ctx, by, "delivery.reassign", id, current, "MANUAL:"+kitchenID.String(), clean)
	return nil
}

func (f *Fulfilment) log(ctx context.Context, by Actor, action string,
	id uuid.UUID, before, after, reason string) {
	_ = f.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &by.UserID, ActorEmail: by.Email, Action: action,
		EntityType: "delivery", EntityID: &id,
		Before: map[string]any{"status": before},
		After:  map[string]any{"status": after},
		Reason: reason, IP: by.IP, UserAgent: by.UA,
	})
}
