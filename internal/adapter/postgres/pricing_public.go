package postgres

import "context"

// The public price list.
//
// Separate from the admin pricing queries on purpose: this reads only what a
// visitor may see. The DEFAULT scope only — a customer-type price is a
// negotiated corporate rate and must never appear on a marketing page — and
// only rows that are active and valid today.

// PublicMealPrice is one diet type at one quantity tier.
type PublicMealPrice struct {
	DietName   string `json:"diet_name"`
	DietSlug   string `json:"diet_slug"`
	TierLabel  string `json:"tier_label"`
	TierMinQty int    `json:"tier_min_qty"`
	// Explicit column tag, and NOT optional: gorm's NamingStrategy renders
	// "UnitPriceIDR" as unit_price_id_r, which matches nothing, and a scan
	// into a column that does not exist does not error — it leaves the field
	// at zero. That is how a price list silently advertises Rp 0.
	// TestMoneyFieldsHaveExplicitColumnTags guards it.
	UnitPriceIDR int64 `json:"unit_price_idr" gorm:"column:unit_price_idr"`
}

// PublicMealPrices returns the standard per-meal prices.
func (r *PricingRepo) PublicMealPrices(ctx context.Context) ([]PublicMealPrice, error) {
	var out []PublicMealPrice
	err := r.db.WithContext(ctx).Raw(`
		SELECT d.name  AS diet_name,
		       d.slug  AS diet_slug,
		       t.label AS tier_label,
		       t.min_qty AS tier_min_qty,
		       p.unit_price_idr
		  FROM meal_price_normal p
		  JOIN diet_type d        ON d.id = p.diet_type_id
		  JOIN meal_price_tier t  ON t.id = p.tier_id
		 WHERE p.is_active
		   AND d.is_active
		   -- DEFAULT scope only: a CT: row is a negotiated rate.
		   AND p.customer_type_id IS NULL
		   AND p.validity @> CURRENT_DATE
		 ORDER BY d.sort_order, t.min_qty`).Scan(&out).Error
	return out, err
}

// PublicTier is a quantity band, listed even where no price is set yet so the
// page can show the structure rather than an empty table.
type PublicTier struct {
	Label  string `json:"label"`
	MinQty int    `json:"min_qty"`
	MaxQty *int   `json:"max_qty,omitempty"`
}

func (r *PricingRepo) PublicTiers(ctx context.Context) ([]PublicTier, error) {
	var out []PublicTier
	err := r.db.WithContext(ctx).Raw(`
		SELECT label, min_qty, max_qty FROM meal_price_tier
		 WHERE is_active ORDER BY min_qty`).Scan(&out).Error
	return out, err
}

// PublicPackage is a prepaid credit package with its current price.
type PublicPackage struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	MealCredits  int    `json:"meal_credits"`
	ValidityDays int    `json:"validity_days"`
	PriceIDR     *int64 `json:"price_idr,omitempty" gorm:"column:price_idr"`
}

// PublicPackages returns active packages. The price is a LEFT JOIN: a package
// with no current price still appears, with the page saying so, rather than
// vanishing from the list for a reason no visitor could guess.
func (r *PricingRepo) PublicPackages(ctx context.Context) ([]PublicPackage, error) {
	var out []PublicPackage
	err := r.db.WithContext(ctx).Raw(`
		SELECT k.name, k.description, k.meal_credits, k.validity_days,
		       p.price_idr
		  FROM package k
		  LEFT JOIN package_price_normal p
		         ON p.package_id = k.id
		        AND p.is_active
		        AND p.customer_type_id IS NULL
		        AND p.validity @> CURRENT_DATE
		 WHERE k.is_active
		 ORDER BY k.sort_order, k.meal_credits`).Scan(&out).Error
	return out, err
}
