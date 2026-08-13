package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
	"github.com/stevenwilliam/healthy_catering/internal/domain/pricing"
)

// PricingRepo owns the tiers, the four price tables and the packages.
type PricingRepo struct{ db *gorm.DB }

func NewPricingRepo(db *gorm.DB) *PricingRepo { return &PricingRepo{db: db} }

// ErrPriceOverlap means the database's exclusion constraint refused a row
// because another price already covers those dates for that scope.
//
// The constraint is the authority, not an application check: a concurrent
// double-submit beats any read-then-write (PROMPT §5.3).
type ErrPriceOverlap struct {
	Conflicting string
}

func (e *ErrPriceOverlap) Error() string {
	return "postgres: price overlaps an existing row " + e.Conflicting
}

// PriceRow is one row of any of the four price tables, as the admin sees it.
type PriceRow struct {
	ID             uuid.UUID  `json:"id"`
	CustomerTypeID *uuid.UUID `json:"customer_type_id,omitempty"`
	ScopeKey       string     `json:"scope_key"`
	ScopeName      string     `json:"scope"`
	DietTypeID     *uuid.UUID `json:"diet_type_id,omitempty"`
	DietTypeName   string     `json:"diet_type,omitempty"`
	TierID         *uuid.UUID `json:"tier_id,omitempty"`
	TierLabel      string     `json:"tier,omitempty"`
	PackageID      *uuid.UUID `json:"package_id,omitempty"`
	PackageName    string     `json:"package,omitempty"`
	PriceIDR       int64      `json:"price_idr" gorm:"column:price_idr"`
	PriceFormatted string     `json:"price"`
	PromoLabel     string     `json:"promo_label,omitempty"`
	ValidFrom      string     `json:"valid_from"`
	ValidTo        *string    `json:"valid_to,omitempty"`
	Note           string     `json:"note,omitempty"`
	IsActive       bool       `json:"is_active"`
}

// priceTable names the four tables and what varies between them.
type priceTable struct {
	name     string
	isPromo  bool
	isMeal   bool
	priceCol string
}

var tables = map[string]priceTable{
	"meal_normal":    {"meal_price_normal", false, true, "unit_price_idr"},
	"meal_promo":     {"meal_price_promo", true, true, "unit_price_idr"},
	"package_normal": {"package_price_normal", false, false, "price_idr"},
	"package_promo":  {"package_price_promo", true, false, "price_idr"},
}

// TableNames returns the four keys, for validation at the edge.
func TableNames() []string {
	return []string{"meal_normal", "meal_promo", "package_normal", "package_promo"}
}

// ListPrices returns a searchable page from one of the four tables.
//
// Four tables and four forms is an explicit requirement (PROMPT §5.2), so the
// table is a parameter rather than a union: an admin editing promo prices must
// never see a normal price in the same grid and edit the wrong one.
func (r *PricingRepo) ListPrices(ctx context.Context, table string, p ListParams) (Page[PriceRow], error) {
	t, ok := tables[table]
	if !ok {
		return Page[PriceRow]{}, fmt.Errorf("postgres: unknown price table %q", table)
	}
	p = p.Normalise("valid_from", "valid_from", "price")

	sel := fmt.Sprintf(`pt.id, pt.customer_type_id, pt.scope_key,
		COALESCE(ct.name, 'All customers (DEFAULT)') AS scope_name,
		pt.%s AS price_idr, pt.valid_from::text AS valid_from,
		pt.valid_to::text AS valid_to, pt.is_active`, t.priceCol)
	if t.isPromo {
		sel += ", pt.promo_label"
	} else {
		sel += ", pt.note"
	}
	if t.isMeal {
		sel += ", pt.diet_type_id, dt.name AS diet_type_name, pt.tier_id, tr.label AS tier_label"
	} else {
		sel += ", pt.package_id, pk.name AS package_name"
	}

	db := r.db.WithContext(ctx).Table(t.name + " pt").
		Joins("LEFT JOIN customer_type ct ON ct.id = pt.customer_type_id")
	if t.isMeal {
		db = db.Joins("JOIN diet_type dt ON dt.id = pt.diet_type_id").
			Joins("JOIN meal_price_tier tr ON tr.id = pt.tier_id")
	} else {
		db = db.Joins("JOIN package pk ON pk.id = pt.package_id")
	}

	if p.Q != "" {
		pattern := SearchPattern(p.Q)
		if t.isMeal {
			db = db.Where(`lower(COALESCE(ct.name,'default')) LIKE ? OR lower(dt.name) LIKE ?
			               OR lower(tr.label) LIKE ? OR pt.`+t.priceCol+`::text LIKE ?`,
				pattern, pattern, pattern, pattern)
		} else {
			db = db.Where(`lower(COALESCE(ct.name,'default')) LIKE ? OR lower(pk.name) LIKE ?
			               OR pt.`+t.priceCol+`::text LIKE ?`, pattern, pattern, pattern)
		}
	}
	if p.Active != nil {
		db = db.Where("pt.is_active = ?", *p.Active)
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[PriceRow]{}, fmt.Errorf("postgres: count prices: %w", err)
	}

	order := "pt.valid_from DESC"
	if p.Sort == "price" {
		order = "pt." + t.priceCol + " " + map[bool]string{true: "DESC", false: "ASC"}[p.Desc]
	}

	var items []PriceRow
	if err := db.Session(&gorm.Session{}).Select(sel).Order(order).
		Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error; err != nil {
		return Page[PriceRow]{}, fmt.Errorf("postgres: list prices: %w", err)
	}
	for i := range items {
		items[i].PriceFormatted = money.Format(money.IDR(items[i].PriceIDR))
	}
	return NewPage(items, total, p), nil
}

// SavePriceInput is a new or edited price row.
type SavePriceInput struct {
	ID             uuid.UUID
	Table          string
	CustomerTypeID *uuid.UUID
	DietTypeID     *uuid.UUID
	TierID         *uuid.UUID
	PackageID      *uuid.UUID
	PriceIDR       int64 `gorm:"column:price_idr"`
	PromoLabel     string
	Note           string
	ValidFrom      string
	ValidTo        *string
	IsActive       bool
}

// SavePrice inserts or updates a price row, translating the exclusion
// constraint into a message that names the conflicting row.
func (r *PricingRepo) SavePrice(ctx context.Context, in SavePriceInput, by uuid.UUID) (uuid.UUID, error) {
	t, ok := tables[in.Table]
	if !ok {
		return uuid.Nil, fmt.Errorf("postgres: unknown price table %q", in.Table)
	}

	labelCol, labelVal := "note", in.Note
	if t.isPromo {
		labelCol, labelVal = "promo_label", in.PromoLabel
	}

	id := in.ID
	var err error
	if id == uuid.Nil {
		id = uuid.Must(uuid.NewV7())
		if t.isMeal {
			err = r.db.WithContext(ctx).Exec(fmt.Sprintf(`
				INSERT INTO %s (id, customer_type_id, diet_type_id, tier_id, %s, %s,
				                valid_from, valid_to, is_active, created_by, updated_by)
				VALUES (?,?,?,?,?,?,?::date,?::date,?,?,?)`, t.name, t.priceCol, labelCol),
				id, in.CustomerTypeID, in.DietTypeID, in.TierID, in.PriceIDR, labelVal,
				in.ValidFrom, in.ValidTo, in.IsActive, by, by).Error
		} else {
			err = r.db.WithContext(ctx).Exec(fmt.Sprintf(`
				INSERT INTO %s (id, customer_type_id, package_id, %s, %s,
				                valid_from, valid_to, is_active, created_by, updated_by)
				VALUES (?,?,?,?,?,?::date,?::date,?,?,?)`, t.name, t.priceCol, labelCol),
				id, in.CustomerTypeID, in.PackageID, in.PriceIDR, labelVal,
				in.ValidFrom, in.ValidTo, in.IsActive, by, by).Error
		}
	} else {
		res := r.db.WithContext(ctx).Exec(fmt.Sprintf(`
			UPDATE %s SET %s = ?, %s = ?, valid_from = ?::date, valid_to = ?::date,
			              is_active = ?, updated_by = ?
			 WHERE id = ?`, t.name, t.priceCol, labelCol),
			in.PriceIDR, labelVal, in.ValidFrom, in.ValidTo, in.IsActive, by, id)
		err = res.Error
		if err == nil && res.RowsAffected == 0 {
			return uuid.Nil, ErrNotFound
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "no_overlap") {
			conflict, _ := r.findConflict(ctx, in)
			return uuid.Nil, &ErrPriceOverlap{Conflicting: conflict}
		}
		return uuid.Nil, fmt.Errorf("postgres: save price: %w", err)
	}
	return id, nil
}

// findConflict locates the row that blocked a save, so the admin form can say
// "overlaps with the price valid 01–15 Aug" rather than throwing a 500
// (PROMPT §5.3).
func (r *PricingRepo) findConflict(ctx context.Context, in SavePriceInput) (string, error) {
	t := tables[in.Table]
	scope := "DEFAULT"
	if in.CustomerTypeID != nil {
		scope = "CT:" + in.CustomerTypeID.String()
	}

	q := fmt.Sprintf(`
		SELECT valid_from::text || COALESCE(' to ' || (valid_to - 1)::text, ' onwards')
		       || ' at ' || %s::text
		  FROM %s
		 WHERE scope_key = ? AND is_active
		   AND validity && daterange(?::date, ?::date, '[)')`, t.priceCol, t.name)
	args := []any{scope, in.ValidFrom, in.ValidTo}

	if t.isMeal {
		q += " AND diet_type_id = ? AND tier_id = ?"
		args = append(args, in.DietTypeID, in.TierID)
	} else {
		q += " AND package_id = ?"
		args = append(args, in.PackageID)
	}
	if in.ID != uuid.Nil {
		q += " AND id <> ?"
		args = append(args, in.ID)
	}
	q += " LIMIT 1"

	var out []string
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&out).Error; err != nil {
		return "", err
	}
	if len(out) == 0 {
		return "", nil
	}
	return out[0], nil
}

// LoadMealCatalogue loads the candidate rows the resolver needs for one diet
// type: the tiers, plus every active normal and promo row for both the
// customer's scope and DEFAULT.
//
// Both scopes are loaded in ONE query rather than falling back with a second:
// the fallback is a business rule and belongs in the pure resolver, not in the
// number of round trips the repository happened to make.
func (r *PricingRepo) LoadMealCatalogue(ctx context.Context, customerTypeID uuid.UUID,
	dietTypeID uuid.UUID, on time.Time) (pricing.Catalogue, error) {

	var cat pricing.Catalogue

	var tiers []struct {
		ID     uuid.UUID
		Label  string
		MinQty int
		MaxQty *int
		Active bool
	}
	if err := r.db.WithContext(ctx).Raw(
		`SELECT id, label, min_qty, max_qty, is_active AS active
		   FROM meal_price_tier WHERE is_active ORDER BY min_qty`).Scan(&tiers).Error; err != nil {
		return cat, fmt.Errorf("postgres: tiers: %w", err)
	}
	for _, t := range tiers {
		cat.Tiers = append(cat.Tiers, pricing.Tier{
			ID: t.ID, Label: t.Label, MinQty: t.MinQty, MaxQty: t.MaxQty, Active: t.Active,
		})
	}

	scope := pricing.ScopeFor(customerTypeID)
	load := func(table string, promo bool) ([]pricing.Row, error) {
		var rows []struct {
			ID         uuid.UUID
			ScopeKey   string
			TierID     uuid.UUID
			PriceIDR   int64 `gorm:"column:price_idr"`
			ValidFrom  time.Time
			ValidTo    *time.Time
			PromoLabel string
			IsActive   bool
		}
		sel := `id, scope_key, tier_id, unit_price_idr AS price_idr,
		        valid_from, valid_to, is_active`
		if promo {
			sel += ", promo_label"
		} else {
			sel += ", '' AS promo_label"
		}
		err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
			SELECT %s FROM %s
			 WHERE is_active AND diet_type_id = ?
			   AND scope_key IN (?, 'DEFAULT')
			   AND validity @> ?::date`, sel, table),
			dietTypeID, string(scope), on.Format("2006-01-02")).Scan(&rows).Error
		if err != nil {
			return nil, fmt.Errorf("postgres: load %s: %w", table, err)
		}
		out := make([]pricing.Row, 0, len(rows))
		for _, x := range rows {
			out = append(out, pricing.Row{
				ID: x.ID, Scope: pricing.Scope(x.ScopeKey), TierID: x.TierID,
				PriceIDR: money.IDR(x.PriceIDR), ValidFrom: x.ValidFrom, ValidTo: x.ValidTo,
				IsPromo: promo, PromoLabel: x.PromoLabel, Table: table, Active: x.IsActive,
			})
		}
		return out, nil
	}

	var err error
	if cat.Normals, err = load("meal_price_normal", false); err != nil {
		return cat, err
	}
	if cat.Promos, err = load("meal_price_promo", true); err != nil {
		return cat, err
	}
	// The table names the resolver records must match the price_table CHECK on
	// order_line, so the snapshot can be traced back.
	for i := range cat.Normals {
		cat.Normals[i].Table = "meal_normal"
	}
	for i := range cat.Promos {
		cat.Promos[i].Table = "meal_promo"
	}
	return cat, nil
}

// LoadPackageCatalogue loads the price rows for one package.
func (r *PricingRepo) LoadPackageCatalogue(ctx context.Context, customerTypeID, packageID uuid.UUID,
	on time.Time) (normals, promos []pricing.Row, err error) {

	scope := pricing.ScopeFor(customerTypeID)
	load := func(table string, promo bool) ([]pricing.Row, error) {
		var rows []struct {
			ID         uuid.UUID
			ScopeKey   string
			PriceIDR   int64 `gorm:"column:price_idr"`
			ValidFrom  time.Time
			ValidTo    *time.Time
			PromoLabel string
			IsActive   bool
		}
		sel := `id, scope_key, price_idr, valid_from, valid_to, is_active`
		if promo {
			sel += ", promo_label"
		} else {
			sel += ", '' AS promo_label"
		}
		e := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
			SELECT %s FROM %s
			 WHERE is_active AND package_id = ?
			   AND scope_key IN (?, 'DEFAULT')
			   AND validity @> ?::date`, sel, table),
			packageID, string(scope), on.Format("2006-01-02")).Scan(&rows).Error
		if e != nil {
			return nil, fmt.Errorf("postgres: load %s: %w", table, e)
		}
		label := "package_normal"
		if promo {
			label = "package_promo"
		}
		out := make([]pricing.Row, 0, len(rows))
		for _, x := range rows {
			out = append(out, pricing.Row{
				ID: x.ID, Scope: pricing.Scope(x.ScopeKey), PriceIDR: money.IDR(x.PriceIDR),
				ValidFrom: x.ValidFrom, ValidTo: x.ValidTo, IsPromo: promo,
				PromoLabel: x.PromoLabel, Table: label, Active: x.IsActive,
			})
		}
		return out, nil
	}

	if normals, err = load("package_price_normal", false); err != nil {
		return nil, nil, err
	}
	if promos, err = load("package_price_promo", true); err != nil {
		return nil, nil, err
	}
	return normals, promos, nil
}

// Tiers returns the configured tiers, for validation and the admin form.
func (r *PricingRepo) Tiers(ctx context.Context) ([]pricing.Tier, error) {
	var rows []struct {
		ID     uuid.UUID
		Label  string
		MinQty int
		MaxQty *int
		Active bool
	}
	if err := r.db.WithContext(ctx).Raw(
		`SELECT id, label, min_qty, max_qty, is_active AS active
		   FROM meal_price_tier ORDER BY min_qty`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("postgres: tiers: %w", err)
	}
	out := make([]pricing.Tier, 0, len(rows))
	for _, t := range rows {
		out = append(out, pricing.Tier{
			ID: t.ID, Label: t.Label, MinQty: t.MinQty, MaxQty: t.MaxQty, Active: t.Active,
		})
	}
	return out, nil
}

// CorporateScopesDearerThan lists the customer types whose normal price is now
// higher than a DEFAULT promo (docs/02 D-9).
//
// This is the warning the promo form shows. The rule — scope first, then promo
// — means a public promotion does NOT undercut a negotiated corporate rate, so
// a corporate customer can end up paying MORE than a walk-in. That is correct,
// and it is also the kind of correct that generates an angry phone call, so the
// admin gets told at the moment they create the promo.
func (r *PricingRepo) CorporateScopesDearerThan(ctx context.Context, dietTypeID, tierID uuid.UUID,
	promoPrice int64, from string, to *string) ([]string, error) {

	var names []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT ct.name
		  FROM meal_price_normal n
		  JOIN customer_type ct ON ct.id = n.customer_type_id
		 WHERE n.is_active AND n.diet_type_id = ? AND n.tier_id = ?
		   AND n.customer_type_id IS NOT NULL
		   AND n.unit_price_idr > ?
		   AND n.validity && daterange(?::date, ?::date, '[)')
		   AND NOT EXISTS (
		         SELECT 1 FROM meal_price_promo p
		          WHERE p.is_active AND p.customer_type_id = n.customer_type_id
		            AND p.diet_type_id = n.diet_type_id AND p.tier_id = n.tier_id
		            AND p.validity && daterange(?::date, ?::date, '[)'))
		 ORDER BY ct.name`,
		dietTypeID, tierID, promoPrice, from, to, from, to).Scan(&names).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: corporate scopes: %w", err)
	}
	return names, nil
}
