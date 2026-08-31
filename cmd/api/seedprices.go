package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seed-prices fills all four price tables with a coherent demo ladder.
//
//	./bin/api seed-prices
//
// A COMMAND rather than a migration, for the same reason as seed-menu: every
// row here is dated relative to the day it is made, and a migration with a
// baked-in valid_from is wrong the day after it is written.
//
// Three properties this seeder holds to, because a price list that violates
// any of them is worse than an empty one:
//
//   - Integer rupiah end to end. No float touches a price (CLAUDE.md §4).
//   - Every ladder DESCENDS with quantity. A tier that charges more for more
//     is not a bug the UI can hide; the cart's "add four more and every
//     portion drops" nudge would then be a lie.
//   - Every active diet × tier × scope is covered. A gap resolves to
//     PRICE_NOT_CONFIGURED and blocks checkout, which is the correct
//     behaviour and a terrible demo.
//
// Idempotent, and NON-DESTRUCTIVE. The price tables carry a GiST exclusion
// constraint — one active row per (scope, diet, tier) per date range — so
// ON CONFLICT cannot help here: an exclusion violation is not a conflict
// target. Each insert is therefore guarded by NOT EXISTS against an
// overlapping active row, which means running this twice changes nothing and
// running it against real prices leaves them alone.
func runSeedPrices(ctx context.Context, gdb *gorm.DB, log *slog.Logger, replace bool) error {
	// Per-portion prices in WHOLE RUPIAH, cheapest tier last. The first tier
	// is the single-portion price the packages are compared against.
	//
	// Keyed by diet slug so a diet type that does not exist is skipped rather
	// than guessed at.
	ladders := map[string][]int64{
		"healthy":      {78_000, 75_000, 71_000, 68_000},
		"weight-gain":  {88_000, 85_000, 81_000, 78_000},
		"weight-loss":  {76_000, 73_000, 69_000, 66_000},
		"high-protein": {92_000, 89_000, 85_000, 82_000},
		"special-diet": {95_000, 92_000, 88_000, 85_000},
		"keto":         {98_000, 95_000, 91_000, 88_000},
	}

	// Negotiated corporate rates, as a whole-rupiah discount per portion off
	// the default ladder. A subtraction, never a percentage: a percentage of
	// an integer needs rounding, and rounding a price is how two systems come
	// to disagree about what somebody owes.
	corporate := map[string]int64{
		"Siloam Customer": 8_000,
		"Company A":       6_000,
		"Company B":       4_000,
	}

	return gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		loc, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			loc = time.UTC
		}
		// A business calendar date in the operating zone (CLAUDE.md §10).
		from := time.Now().In(loc).Format("2006-01-02")

		// ── replace ─────────────────────────────────────────────────────────
		//
		// Without this, the guard below leaves any row that already covers a
		// range — which is correct, and on this database produced an
		// INCOHERENT ladder: reference data from migration 0011 priced healthy
		// at 55.000 for 1–4 and 48.000 for 10–19, the seeder filled the two
		// gaps at 75.000 and 68.000, and the result charged MORE per portion
		// for five meals than for one. The cart's "add four more and every
		// portion drops" nudge would have been a lie on screen.
		//
		// Superseded, NOT deleted: is_active = FALSE keeps the history, frees
		// the exclusion constraint, and gives artboard S3 its "Arsip" state
		// with real rows behind it. Reversible with a single UPDATE.
		//
		// It supersedes EVERY active row, not a subset. The first attempt
		// spared rows with a created_by, on the theory that those were entered
		// by hand — and the two rows that made the ladder incoherent turned out
		// to have one, so the guard below still refused and `replace` achieved
		// nothing. A discriminator that does not actually separate seed data
		// from real data is worse than none, because it looks like a
		// safeguard.
		//
		// So this is blunt and says so: it is a DEMO command, it needs an extra
		// word on the command line, and it deactivates rather than deletes.
		// Do not run it against production pricing.
		if replace {
			for _, table := range []string{
				"meal_price_normal", "meal_price_promo",
				"package_price_normal", "package_price_promo",
			} {
				res := tx.Exec(`UPDATE ` + table + `
				   SET is_active = FALSE, updated_at = now()
				 WHERE is_active`)
				if res.Error != nil {
					return fmt.Errorf("seed-prices: superseding %s: %w", table, res.Error)
				}
				log.Warn("seed-prices: superseded ALL active rows (reversible: is_active)",
					"table", table, "rows", res.RowsAffected)
			}
		}

		var diets []row
		if err := tx.Raw(`SELECT id::text AS id, slug, name FROM diet_type
		                   WHERE is_active ORDER BY sort_order`).Scan(&diets).Error; err != nil {
			return fmt.Errorf("seed-prices: reading diet types: %w", err)
		}
		var tiers []row
		if err := tx.Raw(`SELECT id::text AS id, label AS slug, label AS name
		                    FROM meal_price_tier WHERE is_active ORDER BY min_qty`).
			Scan(&tiers).Error; err != nil {
			return fmt.Errorf("seed-prices: reading tiers: %w", err)
		}
		if len(tiers) == 0 {
			return fmt.Errorf("seed-prices: no active price tier — seed reference data first")
		}
		var types []row
		if err := tx.Raw(`SELECT id::text AS id, name AS slug, name FROM customer_type`).
			Scan(&types).Error; err != nil {
			return fmt.Errorf("seed-prices: reading customer types: %w", err)
		}

		// ── Meal prices, normal ─────────────────────────────────────────────
		//
		// The DEFAULT scope (customer_type_id NULL) is what the public price
		// list and every retail customer resolves against. The corporate
		// scopes sit beside it and win for their own customers.
		normal := 0
		for _, d := range diets {
			ladder, ok := ladders[d.Slug]
			if !ok {
				log.Warn("seed-prices: no ladder for diet type, skipped", "slug", d.Slug)
				continue
			}
			for i, tier := range tiers {
				// A ladder shorter than the tier list repeats its last (and
				// cheapest) step rather than running off the end.
				price := ladder[min(i, len(ladder)-1)]

				// DEFAULT scope.
				n, err := insertMealPrice(tx, nil, d.ID, tier.ID, price, from)
				if err != nil {
					return fmt.Errorf("seed-prices %s/%s: %w", d.Slug, tier.Name, err)
				}
				normal += n

				// Corporate scopes.
				for _, ct := range types {
					off, negotiated := corporate[ct.Name]
					if !negotiated {
						continue // "Customer Default" resolves through DEFAULT
					}
					id := ct.ID
					n, err := insertMealPrice(tx, &id, d.ID, tier.ID, price-off, from)
					if err != nil {
						return fmt.Errorf("seed-prices %s/%s/%s: %w",
							ct.Name, d.Slug, tier.Name, err)
					}
					normal += n
				}
			}
		}

		// ── Meal prices, promo ──────────────────────────────────────────────
		//
		// One live promo, on the two largest tiers of the default diet. A
		// promo is ALLOWED to overlap a normal price — that is what a promo is
		// — so the exclusion constraint here is scoped to the promo table
		// alone (migration 0007).
		promoUntil := time.Now().In(loc).AddDate(0, 0, 30).Format("2006-01-02")
		promo := 0
		if healthy := findBySlug(diets, "healthy"); healthy != "" {
			for i, tier := range tiers {
				if i < 2 {
					continue // the promo is a volume offer
				}
				price := ladders["healthy"][min(i, len(ladders["healthy"])-1)] - 3_000
				n, err := insertMealPromo(tx, healthy, tier.ID, price,
					"Promo pembukaan", from, promoUntil)
				if err != nil {
					return fmt.Errorf("seed-prices promo: %w", err)
				}
				promo += n
			}
		}

		// ── Package prices ──────────────────────────────────────────────────
		//
		// Priced so the per-portion figure beats the single-portion rate and
		// improves with size — which is the claim artboard M2 makes on screen
		// ("Hemat X dibanding harga satuan"). Whole rupiah, and divisible by
		// the credit count so the per-portion figure is exact rather than a
		// floor that quietly loses rupiah.
		perPortion := map[int]int64{10: 75_000, 20: 71_000, 30: 68_000}
		var packages []struct {
			ID      string
			Name    string
			Credits int
		}
		if err := tx.Raw(`SELECT id::text AS id, name, meal_credits AS credits
		                    FROM package WHERE is_active ORDER BY meal_credits`).
			Scan(&packages).Error; err != nil {
			return fmt.Errorf("seed-prices: reading packages: %w", err)
		}
		pkgNormal, pkgPromo := 0, 0
		for _, p := range packages {
			rate, ok := perPortion[p.Credits]
			if !ok {
				// Fall back to the cheapest meal tier rather than skipping the
				// package: a package with no price appears on the page with
				// "price not set", which is a worse demo than a fair rate.
				rate = 68_000
			}
			total := rate * int64(p.Credits)
			n, err := insertPackagePrice(tx, p.ID, total, from)
			if err != nil {
				return fmt.Errorf("seed-prices package %s: %w", p.Name, err)
			}
			pkgNormal += n

			// The middle package carries the promo, which is the one artboard
			// M2 marks "Paling laris".
			if p.Credits == 20 {
				n, err := insertPackagePromo(tx, p.ID, total-70_000,
					"Promo pembukaan", from, promoUntil)
				if err != nil {
					return fmt.Errorf("seed-prices package promo %s: %w", p.Name, err)
				}
				pkgPromo += n
			}
		}

		// ── Verify, do not assume ───────────────────────────────────────────
		//
		// The two properties this seeder claims are checked against what is
		// actually in the table, and a violation FAILS the transaction. An
		// ascending ladder is invisible until a customer adds a portion and
		// watches the total go up, and a diet with a gap blocks checkout with
		// PRICE_NOT_CONFIGURED — neither is something to discover in a demo.
		var ascending []string
		if err := tx.Raw(`
			WITH l AS (
			  SELECT p.scope_key, d.slug, t.min_qty, p.unit_price_idr,
			         lag(p.unit_price_idr) OVER (
			           PARTITION BY p.scope_key, d.slug ORDER BY t.min_qty) AS prev
			    FROM meal_price_normal p
			    JOIN diet_type d ON d.id = p.diet_type_id
			    JOIN meal_price_tier t ON t.id = p.tier_id
			   WHERE p.is_active)
			SELECT scope_key || ' ' || slug || ' @' || min_qty
			  FROM l WHERE prev IS NOT NULL AND unit_price_idr >= prev`).
			Scan(&ascending).Error; err != nil {
			return fmt.Errorf("seed-prices: checking the ladders: %w", err)
		}
		if len(ascending) > 0 {
			return fmt.Errorf(
				"seed-prices: %d tier step(s) do not get cheaper with quantity: %v — "+
					"re-run with `seed-prices replace` to supersede the rows already there",
				len(ascending), ascending)
		}

		var gaps []string
		if err := tx.Raw(`
			SELECT d.slug || '/' || t.label
			  FROM diet_type d CROSS JOIN meal_price_tier t
			 WHERE d.is_active AND t.is_active
			   AND NOT EXISTS (
			     SELECT 1 FROM meal_price_normal p
			      WHERE p.is_active AND p.scope_key = 'DEFAULT'
			        AND p.diet_type_id = d.id AND p.tier_id = t.id)`).
			Scan(&gaps).Error; err != nil {
			return fmt.Errorf("seed-prices: checking coverage: %w", err)
		}
		if len(gaps) > 0 {
			return fmt.Errorf("seed-prices: %d diet/tier combination(s) have no "+
				"DEFAULT price and would block checkout: %v", len(gaps), gaps)
		}

		log.Info("seed-prices complete",
			"meal_normal", normal, "meal_promo", promo,
			"package_normal", pkgNormal, "package_promo", pkgPromo,
			"diets", len(diets), "tiers", len(tiers),
			"ladders_descend", true, "coverage_complete", true,
			"note", "rows that already covered a date range were left untouched "+
				"unless `replace` was given")
		return nil
	})
}

// row is the small (id, slug, name) shape every reference query above scans
// into. Package-level, so the helpers below share it.
type row struct{ ID, Slug, Name string }

func findBySlug(rows []row, slug string) string {
	for _, r := range rows {
		if r.Slug == slug {
			return r.ID
		}
	}
	return ""
}

// insertMealPrice adds one normal price unless an active row already covers
// that scope, diet and tier for an overlapping range.
//
// The NOT EXISTS is doing the work an ON CONFLICT normally would: the table's
// no-overlap rule is a GiST EXCLUDE constraint, and an exclusion violation
// cannot be named as a conflict target. Without this guard a second run aborts
// the whole transaction.
func insertMealPrice(tx *gorm.DB, customerType *string, dietID, tierID string,
	price int64, from string) (int, error) {
	res := tx.Exec(`
		INSERT INTO meal_price_normal
		  (id, customer_type_id, diet_type_id, tier_id, unit_price_idr, valid_from, is_active)
		SELECT ?, ?::uuid, ?::uuid, ?::uuid, ?, ?::date, TRUE
		 WHERE NOT EXISTS (
		   SELECT 1 FROM meal_price_normal
		    WHERE is_active
		      AND diet_type_id = ?::uuid AND tier_id = ?::uuid
		      AND scope_key = COALESCE('CT:' || ?::text, 'DEFAULT')
		      AND validity && daterange(?::date, NULL, '[)'))`,
		uuid.Must(uuid.NewV7()), customerType, dietID, tierID, price, from,
		dietID, tierID, customerType, from)
	return int(res.RowsAffected), res.Error
}

func insertMealPromo(tx *gorm.DB, dietID, tierID string, price int64,
	label, from, to string) (int, error) {
	res := tx.Exec(`
		INSERT INTO meal_price_promo
		  (id, customer_type_id, diet_type_id, tier_id, unit_price_idr,
		   promo_label, valid_from, valid_to, is_active)
		SELECT ?, NULL, ?::uuid, ?::uuid, ?, ?, ?::date, ?::date, TRUE
		 WHERE NOT EXISTS (
		   SELECT 1 FROM meal_price_promo
		    WHERE is_active AND scope_key = 'DEFAULT'
		      AND diet_type_id = ?::uuid AND tier_id = ?::uuid
		      AND validity && daterange(?::date, ?::date, '[)'))`,
		uuid.Must(uuid.NewV7()), dietID, tierID, price, label, from, to,
		dietID, tierID, from, to)
	return int(res.RowsAffected), res.Error
}

func insertPackagePrice(tx *gorm.DB, packageID string, price int64, from string) (int, error) {
	res := tx.Exec(`
		INSERT INTO package_price_normal
		  (id, customer_type_id, package_id, price_idr, valid_from, is_active)
		SELECT ?, NULL, ?::uuid, ?, ?::date, TRUE
		 WHERE NOT EXISTS (
		   SELECT 1 FROM package_price_normal
		    WHERE is_active AND scope_key = 'DEFAULT' AND package_id = ?::uuid
		      AND validity && daterange(?::date, NULL, '[)'))`,
		uuid.Must(uuid.NewV7()), packageID, price, from, packageID, from)
	return int(res.RowsAffected), res.Error
}

func insertPackagePromo(tx *gorm.DB, packageID string, price int64,
	label, from, to string) (int, error) {
	res := tx.Exec(`
		INSERT INTO package_price_promo
		  (id, customer_type_id, package_id, price_idr, promo_label,
		   valid_from, valid_to, is_active)
		SELECT ?, NULL, ?::uuid, ?, ?, ?::date, ?::date, TRUE
		 WHERE NOT EXISTS (
		   SELECT 1 FROM package_price_promo
		    WHERE is_active AND scope_key = 'DEFAULT' AND package_id = ?::uuid
		      AND validity && daterange(?::date, ?::date, '[)'))`,
		uuid.Must(uuid.NewV7()), packageID, price, label, from, to,
		packageID, from, to)
	return int(res.RowsAffected), res.Error
}
