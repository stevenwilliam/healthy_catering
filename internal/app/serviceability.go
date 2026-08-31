// Package app holds the use cases. It orchestrates the domain and the ports and
// contains no framework and no SQL (CLAUDE.md §2).
package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
	"github.com/stevenwilliam/healthy_catering/internal/domain/routing"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// ErrOutsideEnvelope means the coordinates are not plausible — a missing pin,
// a mis-signed latitude, swapped lat/lng. That is an INPUT error and a
// different message from "we do not deliver there yet" (docs/03 Q-11).
var ErrOutsideEnvelope = errors.New("app: coordinates are outside the operating envelope")

// FeeBand is one delivery-fee distance band, measured from the ASSIGNED
// kitchen rather than from a city centre (docs/02 D-19).
type FeeBand struct {
	MaxKM *float64  `json:"max_km"`
	Fee   money.IDR `json:"fee"`
}

// ServiceabilityResult answers "do you deliver to me?".
type ServiceabilityResult struct {
	Serviceable bool `json:"serviceable"`
	// A pointer, not a bare uuid.UUID: encoding/json cannot omit a zero UUID
	// (it is an array type), and an all-zeroes kitchen id in a "not
	// serviceable" reply looks like a real id to a client.
	KitchenID      *uuid.UUID `json:"kitchen_id,omitempty"`
	KitchenCode    string     `json:"kitchen_code,omitempty"`
	KitchenName    string     `json:"kitchen_name,omitempty"`
	DistanceKM     float64    `json:"distance_km,omitempty"`
	DeliveryFee    money.IDR  `json:"delivery_fee_idr"`
	DeliveryFeeFmt string     `json:"delivery_fee,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	// Message is customer-facing and already in the requested language.
	Message string `json:"message"`
}

// Serviceability answers the delivery-area question used by the address form,
// checkout, and the "do we deliver to you?" widget on the homepage — one
// endpoint, three surfaces (PROMPT §9.3).
type Serviceability struct {
	kitchens *postgres.KitchenRepo
	params   *sysparam.Store
	tz       *time.Location
}

func NewServiceability(k *postgres.KitchenRepo, p *sysparam.Store, tz *time.Location) *Serviceability {
	return &Serviceability{kitchens: k, params: p, tz: tz}
}

// CheckInput is one serviceability question.
type CheckInput struct {
	Lat        float64
	Lng        float64
	SlotID     uuid.UUID
	Date       time.Time
	Qty        int
	District   string
	City       string
	Source     string // WIDGET | ADDRESS_FORM | CHECKOUT
	CustomerID *uuid.UUID
	OrderValue money.IDR
	// NoLog suppresses the out-of-range record. Set ONLY by callers that ask
	// the same question many times for one customer intent — the slot picker
	// asks it once per slot, and four rows per page view would turn the
	// coverage report (artboard S5) from "where demand exists that we cannot
	// serve" into a page-view counter. A real checkout attempt always logs.
	NoLog bool
}

// Check resolves the kitchen, or logs why it could not.
func (s *Serviceability) Check(ctx context.Context, in CheckInput) (ServiceabilityResult, error) {
	at := routing.Point{Lat: in.Lat, Lng: in.Lng}

	if !s.envelope(ctx).Contains(at) {
		return ServiceabilityResult{}, apierror.BadRequest(
			apierror.CodeValidation,
			"Those coordinates do not look like a place we operate. Please drop the pin on the map again.",
		).WithCause(ErrOutsideEnvelope)
	}

	var slot *uuid.UUID
	if in.SlotID != uuid.Nil {
		slot = &in.SlotID
	}
	candidates, err := s.kitchens.Candidates(ctx, at, slot, in.Date)
	if err != nil {
		return ServiceabilityResult{}, apierror.Internal(err)
	}

	qty := in.Qty
	if qty < 1 {
		qty = 1
	}
	assignment, err := routing.Route(routing.Request{To: at, ServiceDate: in.Date, QtyNeeded: qty}, candidates)
	if err != nil {
		if errors.Is(err, routing.ErrNotServiceable) {
			if !in.NoLog {
				s.logMiss(ctx, in, at, candidates)
			}
			return ServiceabilityResult{
				Serviceable: false,
				Message:     "We do not deliver to this address yet. Leave your email and we will tell you when we do.",
			}, nil
		}
		return ServiceabilityResult{}, apierror.Internal(err)
	}

	fee := s.fee(ctx, float64(assignment.DistanceM)/1000, in.OrderValue)
	return ServiceabilityResult{
		Serviceable:    true,
		KitchenID:      &assignment.KitchenID,
		KitchenCode:    assignment.KitchenCode,
		KitchenName:    assignment.KitchenName,
		DistanceKM:     float64(assignment.DistanceM) / 1000,
		DeliveryFee:    fee,
		DeliveryFeeFmt: money.Format(fee),
		Reason:         assignment.Reason,
		Message:        fmt.Sprintf("Dilayani oleh %s.", assignment.KitchenName),
	}, nil
}

// logMiss records an uncovered address, enriched with the nearest kitchen so
// operations can see how far off coverage was.
func (s *Serviceability) logMiss(ctx context.Context, in CheckInput, at routing.Point, candidates []routing.Kitchen) {
	var nearestID *uuid.UUID
	var nearestM *int
	if k, d, ok := routing.Nearest(at, candidates); ok {
		nearestID, nearestM = &k.ID, &d
	}
	source := in.Source
	if source == "" {
		source = "WIDGET"
	}
	var slot *uuid.UUID
	if in.SlotID != uuid.Nil {
		slot = &in.SlotID
	}
	var date *time.Time
	if !in.Date.IsZero() {
		date = &in.Date
	}
	// A failure to log must not fail the request: the customer's answer is
	// already correct, and this is analytics.
	_ = s.kitchens.LogOutOfRange(ctx, at, in.District, in.City, slot, date, source,
		in.CustomerID, nearestID, nearestM)
}

// fee applies the distance bands from the assigned kitchen, with free delivery
// above a threshold. Both are parameters (docs/03 Q-3 — the figures are
// placeholders until Steven gives the real ones).
func (s *Serviceability) fee(ctx context.Context, km float64, orderValue money.IDR) money.IDR {
	freeAbove := s.params.Money(ctx, sysparam.KeyDeliveryFreeAbove, 0)
	if freeAbove > 0 && orderValue >= freeAbove {
		return 0
	}
	var bands []FeeBand
	if err := s.params.JSON(ctx, sysparam.KeyDeliveryFeeBands, &bands); err != nil || len(bands) == 0 {
		return 0
	}
	for _, b := range bands {
		if b.MaxKM == nil || km <= *b.MaxKM {
			return b.Fee
		}
	}
	return bands[len(bands)-1].Fee
}

func (s *Serviceability) envelope(ctx context.Context) routing.Envelope {
	var e struct {
		MinLat float64 `json:"min_lat"`
		MaxLat float64 `json:"max_lat"`
		MinLng float64 `json:"min_lng"`
		MaxLng float64 `json:"max_lng"`
	}
	if err := s.params.JSON(ctx, sysparam.KeyGeoEnvelope, &e); err != nil {
		return routing.JabodetabekEnvelope
	}
	return routing.Envelope{MinLat: e.MinLat, MaxLat: e.MaxLat, MinLng: e.MinLng, MaxLng: e.MaxLng}
}

// Slots exposes the customer-facing delivery slots.
// SlotOffer is one delivery slot as the customer sees it on artboards 04 and
// M3: the alias, and whether this address can actually take it on that date.
type SlotOffer struct {
	SlotID      uuid.UUID `json:"slot_id"`
	Alias       string    `json:"alias"`
	Serviceable bool      `json:"serviceable"`
	KitchenName string    `json:"kitchen_name,omitempty"`
	DeliveryFee string    `json:"delivery_fee,omitempty"`
	// Reason is why not, for the customer. Artboard 04 strikes the slot
	// through and says "12.30 sudah penuh untuk area ini" rather than hiding
	// it — a slot that silently disappears reads as a bug, and the customer
	// cannot tell "full" from "we never deliver then".
	Reason string `json:"reason,omitempty"`
}

// SlotAvailability answers, for one address and one service date, which slots
// can be booked. One Check per active slot: the routing already accounts for
// the kitchen's capacity on that date and slot, so asking it per slot is the
// same question the checkout will ask when it commits.
//
// It never logs a miss. See CheckInput.NoLog.
func (s *Serviceability) SlotAvailability(
	ctx context.Context, lat, lng float64, on time.Time, qty int,
) ([]SlotOffer, error) {
	slots, err := s.Slots(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	out := make([]SlotOffer, 0, len(slots))
	for _, sl := range slots {
		res, err := s.Check(ctx, CheckInput{
			Lat: lat, Lng: lng, SlotID: sl.ID, Date: on, Qty: qty,
			Source: "CHECKOUT", NoLog: true,
		})
		if err != nil {
			// One unroutable slot must not blank the whole picker: the
			// customer still needs to see the slots that DO work.
			out = append(out, SlotOffer{SlotID: sl.ID, Alias: sl.Alias, Reason: "UNAVAILABLE"})
			continue
		}
		out = append(out, SlotOffer{
			SlotID: sl.ID, Alias: sl.Alias,
			Serviceable: res.Serviceable, KitchenName: res.KitchenName,
			DeliveryFee: res.DeliveryFeeFmt,
			Reason: func() string {
				if res.Serviceable {
					return ""
				}
				return "FULL_OR_OUT_OF_RANGE"
			}(),
		})
	}
	return out, nil
}

func (s *Serviceability) Slots(ctx context.Context) ([]postgres.Slot, error) {
	return s.kitchens.ActiveSlots(ctx)
}
