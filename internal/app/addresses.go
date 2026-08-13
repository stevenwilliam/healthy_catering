package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
)

// AddressInput is a new delivery address.
type AddressInput struct {
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

// SavedAddress is the reply, which includes the serviceability answer.
//
// The check runs on SAVE, not at checkout (PROMPT §8.2): telling someone their
// area is not covered while they are trying to pay is the worst possible moment
// to tell them.
type SavedAddress struct {
	ID          uuid.UUID `json:"id"`
	Serviceable bool      `json:"serviceable"`
	KitchenName string    `json:"kitchen_name,omitempty"`
	DistanceKM  float64   `json:"distance_km,omitempty"`
	DeliveryFee string    `json:"delivery_fee,omitempty"`
	Message     string    `json:"message"`
}

// SaveAddress validates, geocodes-checks and stores a delivery address.
func (o *Ordering) SaveAddress(ctx context.Context, ident Identity, in AddressInput) (SavedAddress, error) {
	if ident.CustomerID == nil {
		return SavedAddress{}, apierror.Forbidden(apierror.CodeForbidden,
			"Only customers have delivery addresses.")
	}

	label, err := sanitize.Required("label", in.Label, 60)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	recipient, err := sanitize.Required("recipient_name", in.RecipientName, 120)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	phone, err := sanitize.Phone("recipient_phone", in.RecipientPhone)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	line, err := sanitize.Required("address_line", in.AddressLine, 300)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	district, err := sanitize.Text("district", in.District, 120)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	city, err := sanitize.Text("city", in.City, 120)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	province, err := sanitize.Text("province", in.Province, 120)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	postal, err := sanitize.Text("postal_code", in.PostalCode, 20)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	note, err := sanitize.Text("driver_note", in.DriverNote, 300)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}
	placeID, err := sanitize.Text("google_place_id", in.GooglePlaceID, 200)
	if err != nil {
		return SavedAddress{}, validationFrom(err)
	}

	// The pin is mandatory and is the source of truth for routing; the text is
	// for the driver (PROMPT §8.2). A missing pin is (0,0), which the envelope
	// rejects with a message about the map rather than about the address.
	if in.Latitude == 0 && in.Longitude == 0 {
		return SavedAddress{}, apierror.Validation(
			"Please drop the pin on the map — we route deliveries by the pin, not the text.",
			map[string]any{"latitude": "required", "longitude": "required"})
	}

	// Serviceability first: a saved address we cannot reach is a checkout
	// failure deferred.
	check, err := o.service.Check(ctx, CheckInput{
		Lat: in.Latitude, Lng: in.Longitude, District: district, City: city,
		Source: "ADDRESS_FORM", CustomerID: ident.CustomerID,
	})
	if err != nil {
		return SavedAddress{}, err
	}

	id, err := o.orders.SaveAddress(ctx, postgres.SaveAddressParams{
		CustomerID: *ident.CustomerID, Label: label, RecipientName: recipient,
		RecipientPhone: phone, AddressLine: line, District: district, City: city,
		Province: province, PostalCode: postal, Latitude: in.Latitude,
		Longitude: in.Longitude, GooglePlaceID: placeID, DriverNote: note,
		IsDefault: in.IsDefault,
	})
	if err != nil {
		return SavedAddress{}, apierror.Internal(err)
	}

	out := SavedAddress{ID: id, Serviceable: check.Serviceable, Message: check.Message}
	if check.Serviceable {
		out.KitchenName = check.KitchenName
		out.DistanceKM = check.DistanceKM
		out.DeliveryFee = money.Format(check.DeliveryFee)
	}
	return out, nil
}

// ListAddresses returns the caller's own addresses.
func (o *Ordering) ListAddresses(ctx context.Context, ident Identity) ([]postgres.Address, error) {
	if ident.CustomerID == nil {
		return nil, apierror.Forbidden(apierror.CodeForbidden,
			"Only customers have delivery addresses.")
	}
	list, err := o.orders.ListAddresses(ctx, *ident.CustomerID)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return list, nil
}

// ListMyOrders returns the caller's own orders — scoped by customer id in the
// query, so there is nothing to forget at the handler.
func (o *Ordering) ListMyOrders(ctx context.Context, ident Identity,
	p postgres.ListParams) (postgres.Page[postgres.OrderSummary], error) {

	if ident.CustomerID == nil {
		return postgres.Page[postgres.OrderSummary]{}, apierror.Forbidden(
			apierror.CodeForbidden, "Only customers have orders.")
	}
	page, err := o.orders.ListForCustomer(ctx, *ident.CustomerID, p)
	if err != nil {
		return postgres.Page[postgres.OrderSummary]{}, apierror.Internal(err)
	}
	return page, nil
}

// GetMyOrder returns one of the caller's own orders.
func (o *Ordering) GetMyOrder(ctx context.Context, ident Identity, id uuid.UUID) (postgres.OrderDetail, error) {
	if ident.CustomerID == nil {
		return postgres.OrderDetail{}, apierror.Forbidden(apierror.CodeForbidden,
			"Only customers have orders.")
	}
	// Scoped by owner in the WHERE clause: another customer's id returns NOT
	// FOUND, which is also what a non-existent id returns, so the endpoint
	// cannot be used to discover which order ids exist.
	o2, err := o.orders.GetForCustomer(ctx, *ident.CustomerID, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return postgres.OrderDetail{}, apierror.NotFound("No such order.")
		}
		return postgres.OrderDetail{}, apierror.Internal(err)
	}
	return o2, nil
}

// countdown is the live cut-off timer the menu page shows (PROMPT §6).
func (o *Ordering) countdown(ctx context.Context, serviceDate time.Time) time.Duration {
	return o.cutoffRule(ctx).TimeUntilCutoff(serviceDate, o.now())
}

// ProofInput is a transfer proof reference.
type ProofInput struct {
	ObjectKey   string
	ContentType string
	Bytes       int64
	Checksum    string
}

// SubmitPaymentProof records a customer's transfer proof against their own
// order.
func (o *Ordering) SubmitPaymentProof(ctx context.Context, ident Identity,
	orderID uuid.UUID, in ProofInput) error {

	if ident.CustomerID == nil {
		return apierror.Forbidden(apierror.CodeForbidden, "Only customers upload payment proof.")
	}

	// The object key is validated as a KEY, never as a path: a client-supplied
	// "../" is how an upload escapes its prefix (99 §7).
	key, err := sanitize.Required("object_key", in.ObjectKey, 300)
	if err != nil {
		return validationFrom(err)
	}
	if strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return apierror.Validation("That object key is not valid.",
			map[string]any{"object_key": "no leading slash and no .."})
	}

	// Magic-byte checking happens in the storage adapter on upload; the
	// declared content type is still allow-listed here so a rejected type never
	// reaches the finance queue (99 §7).
	ct, err := sanitize.Enum("content_type", in.ContentType,
		"image/jpeg", "image/png", "image/webp", "application/pdf")
	if err != nil {
		return apierror.Validation("Upload a JPEG, PNG, WebP or PDF.",
			map[string]any{"content_type": "image/jpeg, image/png, image/webp or application/pdf"})
	}
	const maxProofBytes = 5 << 20 // PROMPT §10
	if in.Bytes <= 0 || in.Bytes > maxProofBytes {
		return apierror.Validation("The file must be 5 MB or smaller.",
			map[string]any{"bytes": "1 byte to 5 MB"})
	}

	if err := o.payments.SubmitProof(ctx, *ident.CustomerID, orderID,
		key, ct, in.Bytes, in.Checksum); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return apierror.NotFound("No such order awaiting payment.")
		}
		return apierror.Internal(err)
	}
	return nil
}
