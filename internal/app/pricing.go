package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
	"github.com/stevenwilliam/healthy_catering/internal/domain/pricing"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Pricing is the four price tables and the quote endpoint.
type Pricing struct {
	repo   *postgres.PricingRepo
	audit  *postgres.AuditRepo
	params *sysparam.Store
	tz     *time.Location
}

func NewPricing(r *postgres.PricingRepo, a *postgres.AuditRepo,
	p *sysparam.Store, tz *time.Location) *Pricing {
	return &Pricing{repo: r, audit: a, params: p, tz: tz}
}

func (s *Pricing) ListPrices(ctx context.Context, table string, p postgres.ListParams) (postgres.Page[postgres.PriceRow], error) {
	if _, err := sanitize.Enum("table", table, postgres.TableNames()...); err != nil {
		return postgres.Page[postgres.PriceRow]{}, validationFrom(err)
	}
	page, err := s.repo.ListPrices(ctx, table, p)
	if err != nil {
		return postgres.Page[postgres.PriceRow]{}, apierror.Internal(err)
	}
	return page, nil
}

func (s *Pricing) Tiers(ctx context.Context) ([]pricing.Tier, error) {
	t, err := s.repo.Tiers(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return t, nil
}

// PriceInput is a create or edit on one of the four tables.
type PriceInput struct {
	ID             uuid.UUID
	Table          string
	CustomerTypeID *uuid.UUID
	DietTypeID     *uuid.UUID
	TierID         *uuid.UUID
	PackageID      *uuid.UUID
	PriceIDR       int64
	PromoLabel     string
	Note           string
	ValidFrom      string
	ValidTo        *string
	IsActive       bool
}

// SavePriceResult carries the warning the promo form shows (D-9).
type SavePriceResult struct {
	ID           uuid.UUID `json:"id"`
	Warning      string    `json:"warning,omitempty"`
	DearerScopes []string  `json:"corporate_scopes_now_dearer,omitempty"`
}

// SavePrice validates and stores a price row.
func (s *Pricing) SavePrice(ctx context.Context, in PriceInput, by Actor) (SavePriceResult, error) {
	table, err := sanitize.Enum("table", in.Table, postgres.TableNames()...)
	if err != nil {
		return SavePriceResult{}, validationFrom(err)
	}
	isMeal := table == "meal_normal" || table == "meal_promo"
	isPromo := table == "meal_promo" || table == "package_promo"

	if err := money.Validate(money.IDR(in.PriceIDR)); err != nil {
		return SavePriceResult{}, apierror.Validation(
			"That price is not a valid rupiah amount.",
			map[string]any{"price_idr": "a whole number of rupiah, zero or more"})
	}

	from, err := s.parseDate("valid_from", in.ValidFrom)
	if err != nil {
		return SavePriceResult{}, err
	}
	var to *time.Time
	if in.ValidTo != nil && *in.ValidTo != "" {
		t, err := s.parseDate("valid_to", *in.ValidTo)
		if err != nil {
			return SavePriceResult{}, err
		}
		if !t.After(from) {
			return SavePriceResult{}, apierror.Validation(
				"The end date must be after the start date.",
				map[string]any{"valid_to": "after valid_from"})
		}
		to = &t
	}

	if isMeal {
		if in.DietTypeID == nil || in.TierID == nil {
			return SavePriceResult{}, apierror.Validation(
				"A meal price needs a diet type and a quantity tier.", nil)
		}
	} else if in.PackageID == nil {
		return SavePriceResult{}, apierror.Validation("A package price needs a package.", nil)
	}

	label, err := sanitize.Text("promo_label", in.PromoLabel, 80)
	if err != nil {
		return SavePriceResult{}, validationFrom(err)
	}
	if isPromo && label == "" {
		// The cart shows the promo label next to the struck-through price
		// (PROMPT §5.2); an unlabelled promo renders as a bare discount with
		// no explanation of why or until when.
		return SavePriceResult{}, apierror.Validation(
			"A promotional price needs a label — customers see it next to the old price.",
			map[string]any{"promo_label": `e.g. "Promo Agustus"`})
	}
	note, err := sanitize.Text("note", in.Note, 200)
	if err != nil {
		return SavePriceResult{}, validationFrom(err)
	}

	save := postgres.SavePriceInput{
		ID: in.ID, Table: table, CustomerTypeID: in.CustomerTypeID,
		DietTypeID: in.DietTypeID, TierID: in.TierID, PackageID: in.PackageID,
		PriceIDR: in.PriceIDR, PromoLabel: label, Note: note,
		ValidFrom: from.Format("2006-01-02"), IsActive: in.IsActive,
	}
	if to != nil {
		v := to.Format("2006-01-02")
		save.ValidTo = &v
	}

	id, err := s.repo.SavePrice(ctx, save, by.UserID)
	if err != nil {
		var overlap *postgres.ErrPriceOverlap
		if errors.As(err, &overlap) {
			msg := "Another price already covers those dates for this scope."
			if overlap.Conflicting != "" {
				msg = "Overlaps with the price valid " + overlap.Conflicting + "."
			}
			return SavePriceResult{}, apierror.Conflict(apierror.CodeConflict, msg)
		}
		if errors.Is(err, postgres.ErrNotFound) {
			return SavePriceResult{}, apierror.NotFound("That price row no longer exists.")
		}
		return SavePriceResult{}, apierror.Internal(err)
	}

	res := SavePriceResult{ID: id}

	// D-9's operational fix: creating a DEFAULT promo can leave corporate
	// customers paying more than walk-ins, because scope resolves before promo.
	// One screen, no schema change — say so at the moment it happens.
	if isPromo && isMeal && in.CustomerTypeID == nil {
		names, err := s.repo.CorporateScopesDearerThan(ctx, *in.DietTypeID, *in.TierID,
			in.PriceIDR, save.ValidFrom, save.ValidTo)
		if err == nil && len(names) > 0 {
			res.DearerScopes = names
			res.Warning = fmt.Sprintf(
				"This public promotion does not apply to %d corporate scope(s), whose negotiated price is now HIGHER than the public one: %v. "+
					"Create matching promo rows for them if that is not what you want.",
				len(names), names)
		}
	}

	action := "price.update"
	if in.ID == uuid.Nil {
		action = "price.create"
	}
	_ = s.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &by.UserID, ActorEmail: by.Email, Action: action,
		EntityType: table, EntityID: &id, After: save, IP: by.IP, UserAgent: by.UA,
	})
	return res, nil
}

// Quote is a priced answer for the cart.
type Quote struct {
	Qty            int    `json:"qty"`
	UnitPriceIDR   int64  `json:"unit_price_idr"`
	UnitPrice      string `json:"unit_price"`
	NormalPriceIDR int64  `json:"normal_price_idr"`
	NormalPrice    string `json:"normal_price"`
	IsPromo        bool   `json:"is_promo"`
	PromoLabel     string `json:"promo_label,omitempty"`
	LineTotalIDR   int64  `json:"line_total_idr"`
	LineTotal      string `json:"line_total"`
	TaxBaseIDR     int64  `json:"tax_base_idr"`
	TaxIDR         int64  `json:"tax_idr"`
	TaxRateBps     int    `json:"tax_rate_bps"`
	Tier           string `json:"tier"`
	Scope          string `json:"scope"`
	// Savings is what the promo is worth, for the cart's "you save" line.
	SavingsIDR int64  `json:"savings_idr,omitempty"`
	Savings    string `json:"savings,omitempty"`
}

// QuoteMeals prices a quantity of meals of one diet type on one date.
func (s *Pricing) QuoteMeals(ctx context.Context, customerTypeID, dietTypeID uuid.UUID,
	qty int, onDate time.Time) (Quote, pricing.Resolved, error) {

	cat, err := s.repo.LoadMealCatalogue(ctx, customerTypeID, dietTypeID, onDate)
	if err != nil {
		return Quote{}, pricing.Resolved{}, apierror.Internal(err)
	}

	req := pricing.Request{
		CustomerType: customerTypeID, Qty: qty, OnDate: onDate,
		MaxQty:     s.params.Int(ctx, sysparam.KeyMaxQtyPerLine, 999),
		TaxRateBps: s.params.Int(ctx, sysparam.KeyTaxRateBps, 0),
	}

	res, err := pricing.ResolveMeal(req, cat)
	if err != nil {
		return Quote{}, pricing.Resolved{}, priceError(err)
	}
	return quoteFrom(res, qty), res, nil
}

// QuotePackage prices one package.
func (s *Pricing) QuotePackage(ctx context.Context, customerTypeID, packageID uuid.UUID,
	onDate time.Time) (Quote, pricing.Resolved, error) {

	normals, promos, err := s.repo.LoadPackageCatalogue(ctx, customerTypeID, packageID, onDate)
	if err != nil {
		return Quote{}, pricing.Resolved{}, apierror.Internal(err)
	}
	req := pricing.Request{
		CustomerType: customerTypeID, Qty: 1, OnDate: onDate,
		TaxRateBps: s.params.Int(ctx, sysparam.KeyTaxRateBps, 0),
	}
	res, err := pricing.ResolvePackage(req, normals, promos)
	if err != nil {
		return Quote{}, pricing.Resolved{}, priceError(err)
	}
	return quoteFrom(res, 1), res, nil
}

func quoteFrom(res pricing.Resolved, qty int) Quote {
	q := Quote{
		Qty:            qty,
		UnitPriceIDR:   int64(res.UnitPrice),
		UnitPrice:      money.Format(res.UnitPrice),
		NormalPriceIDR: int64(res.NormalPrice),
		NormalPrice:    money.Format(res.NormalPrice),
		IsPromo:        res.IsPromo,
		PromoLabel:     res.PromoLabel,
		LineTotalIDR:   int64(res.LineTotal),
		LineTotal:      money.Format(res.LineTotal),
		TaxBaseIDR:     int64(res.Split.Base),
		TaxIDR:         int64(res.Split.Tax),
		TaxRateBps:     res.Split.RateBps,
		Tier:           res.TierLabel,
		Scope:          string(res.Scope),
	}
	if res.IsPromo && res.NormalPrice > res.UnitPrice {
		saving := (res.NormalPrice - res.UnitPrice) * money.IDR(qty)
		q.SavingsIDR = int64(saving)
		q.Savings = money.Format(saving)
	}
	return q
}

// priceError turns a resolver failure into a customer-facing message.
//
// A missing price BLOCKS the purchase (PROMPT §5.1.3). The system never invents
// a number, and the message says so plainly rather than pretending the item is
// unavailable — staff need to know it is a pricing gap, not a stock problem.
func priceError(err error) error {
	switch {
	case errors.Is(err, pricing.ErrNotConfigured):
		return apierror.Unprocessable(apierror.CodeValidation,
			"No price is configured for this item on that date. Please contact us — we will not guess a price.")
	case errors.Is(err, pricing.ErrNoTier):
		return apierror.Unprocessable(apierror.CodeValidation,
			"No price tier covers that quantity.")
	case errors.Is(err, pricing.ErrQtyOutOfRange):
		return apierror.Validation("That quantity is out of range.",
			map[string]any{"qty": err.Error()})
	default:
		return apierror.Internal(err)
	}
}

func (s *Pricing) parseDate(field, v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, apierror.Validation("A date is required.",
			map[string]any{field: "YYYY-MM-DD"})
	}
	t, err := time.ParseInLocation("2006-01-02", v, s.tz)
	if err != nil {
		return time.Time{}, apierror.Validation("That date is not valid.",
			map[string]any{field: "YYYY-MM-DD"})
	}
	return t, nil
}
