package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seed-menu fills the calendar with a sample menu for the next N days.
//
// A COMMAND rather than a migration, deliberately. Migrations are forward-only
// and permanent, and these rows are dated relative to the day they are made —
// a migration with 2026-08-18 baked into it is wrong on 2026-08-19 and wrong
// forever after. This is re-runnable, idempotent, and safe to run on any day.
//
//	./bin/api seed-menu        # today + 2, i.e. three days
//	./bin/api seed-menu 7      # a week
//
// Idempotent on both halves: foods conflict on slug, scheduled meals on the
// (service_date, diet_type_id, slot_id) unique the schema already has. Running
// it twice changes nothing.
func runSeedMenu(ctx context.Context, gdb *gorm.DB, log *slog.Logger, days int) error {
	if days <= 0 {
		days = 3
	}

	// The dishes the sample menus are built from. Nutrition is per portion,
	// integers only, milligrams for everything but calories (docs/02 D-24).
	type foodSeed struct {
		slug, name, portion string
		kcal                int
		proteinMg           int
		fatMg               int
		carbMg              int
	}
	foods := []foodSeed{
		{"nasi-merah", "Nasi Merah", "150 g", 180, 4000, 1500, 38000},
		{"dada-ayam-panggang", "Dada Ayam Panggang", "120 g", 220, 31000, 9000, 0},
		{"ikan-dori-lemon", "Ikan Dori Saus Lemon", "130 g", 190, 26000, 8000, 2000},
		{"tempe-bacem", "Tempe Bacem", "80 g", 160, 12000, 8000, 9000},
		{"tahu-kukus-jamur", "Tahu Kukus Jamur", "100 g", 120, 11000, 6000, 4000},
		{"telur-rebus", "Telur Rebus", "1 butir", 78, 6300, 5300, 600},
		{"brokoli-kukus", "Brokoli Kukus", "100 g", 55, 3700, 600, 11000},
		{"salad-sayur-wijen", "Salad Sayur Saus Wijen", "120 g", 90, 3000, 5000, 8000},
		{"sup-bening-bayam", "Sup Bening Bayam", "200 ml", 60, 3500, 1000, 9000},
		{"daging-sapi-lada-hitam", "Daging Sapi Lada Hitam", "120 g", 260, 28000, 15000, 3000},
		{"udang-saus-padang", "Udang Saus Padang", "110 g", 210, 24000, 9000, 6000},
		{"alpukat-potong", "Alpukat Potong", "80 g", 160, 2000, 15000, 7000},
		{"kentang-panggang-herb", "Kentang Panggang Herb", "150 g", 150, 3000, 4000, 26000},
		{"quinoa-sayur", "Quinoa Sayur", "150 g", 170, 6000, 3000, 30000},
		{"sayur-lodeh-ringan", "Sayur Lodeh Ringan", "180 ml", 110, 3000, 6000, 10000},
		{"mie-shirataki-ayam", "Mie Shirataki Ayam", "200 g", 140, 20000, 4000, 5000},
		{"keju-cheddar", "Keju Cheddar", "30 g", 110, 7000, 9000, 1000},
		{"smoothie-buah-naga", "Smoothie Buah Naga", "200 ml", 130, 2000, 1000, 28000},
	}

	// One menu per diet type per day. Three variants each, cycled, so a week's
	// seed does not serve the same plate every day.
	type menu struct {
		name  string
		items []string // food slugs; the first is the MAIN
	}
	menus := map[string][]menu{
		"healthy": {
			{"Ayam Panggang & Nasi Merah", []string{"dada-ayam-panggang", "nasi-merah", "brokoli-kukus"}},
			{"Dori Lemon & Quinoa", []string{"ikan-dori-lemon", "quinoa-sayur", "salad-sayur-wijen"}},
			{"Tempe Bacem & Sup Bayam", []string{"tempe-bacem", "nasi-merah", "sup-bening-bayam"}},
		},
		"weight-gain": {
			{"Sapi Lada Hitam & Kentang", []string{"daging-sapi-lada-hitam", "kentang-panggang-herb", "alpukat-potong"}},
			{"Udang Padang & Nasi Merah", []string{"udang-saus-padang", "nasi-merah", "keju-cheddar"}},
			{"Ayam Panggang & Smoothie", []string{"dada-ayam-panggang", "kentang-panggang-herb", "smoothie-buah-naga"}},
		},
		"weight-loss": {
			{"Dori Lemon & Salad", []string{"ikan-dori-lemon", "salad-sayur-wijen", "sup-bening-bayam"}},
			{"Ayam Panggang & Brokoli", []string{"dada-ayam-panggang", "brokoli-kukus", "salad-sayur-wijen"}},
			{"Tahu Jamur & Sup Bayam", []string{"tahu-kukus-jamur", "salad-sayur-wijen", "sup-bening-bayam"}},
		},
		"high-protein": {
			{"Ayam, Telur & Brokoli", []string{"dada-ayam-panggang", "telur-rebus", "brokoli-kukus"}},
			{"Sapi Lada Hitam & Quinoa", []string{"daging-sapi-lada-hitam", "telur-rebus", "quinoa-sayur"}},
			{"Udang & Tempe Bacem", []string{"udang-saus-padang", "tempe-bacem", "brokoli-kukus"}},
		},
		"special-diet": {
			{"Tahu Kukus & Sup Bening", []string{"tahu-kukus-jamur", "sup-bening-bayam", "nasi-merah"}},
			{"Dori Lemon & Sayur Lodeh", []string{"ikan-dori-lemon", "kentang-panggang-herb", "sayur-lodeh-ringan"}},
			{"Tempe Bacem & Sayur Lodeh", []string{"tempe-bacem", "sayur-lodeh-ringan", "pepaya-potong"}},
		},
		"keto": {
			{"Ayam Panggang & Alpukat", []string{"dada-ayam-panggang", "alpukat-potong", "brokoli-kukus"}},
			{"Sapi Lada Hitam & Keju", []string{"daging-sapi-lada-hitam", "keju-cheddar", "salad-sayur-wijen"}},
			{"Shirataki Ayam & Telur", []string{"mie-shirataki-ayam", "telur-rebus", "alpukat-potong"}},
		},
	}

	return gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ── Foods ───────────────────────────────────────────────────────────
		for _, f := range foods {
			id := uuid.Must(uuid.NewV7())
			if err := tx.Exec(`
				INSERT INTO food (id, name, slug, portion_size, is_active)
				VALUES (?, ?, ?, ?, TRUE)
				ON CONFLICT (slug) DO NOTHING`,
				id, f.name, f.slug, f.portion).Error; err != nil {
				return fmt.Errorf("seed food %s: %w", f.slug, err)
			}
			// is_complete TRUE: these panels are filled in, so the meal
			// aggregate is exact rather than an estimate. A dish with an
			// incomplete panel makes the whole meal show "perkiraan".
			if err := tx.Exec(`
				INSERT INTO food_nutrition
				  (food_id, calories_kcal, protein_mg, fat_mg, carbohydrate_mg, is_complete)
				SELECT id, ?, ?, ?, ?, TRUE FROM food WHERE slug = ?
				ON CONFLICT (food_id) DO NOTHING`,
				f.kcal, f.proteinMg, f.fatMg, f.carbMg, f.slug).Error; err != nil {
				return fmt.Errorf("seed nutrition %s: %w", f.slug, err)
			}
		}

		// ── The lunch slot ──────────────────────────────────────────────────
		// Scanned as text, not uuid.UUID: gorm's Raw().Scan() into a bare
		// non-struct destination hands the driver string straight to a
		// numeric conversion and fails.
		var slotID string
		if err := tx.Raw(`
			SELECT id::text FROM delivery_time_slot
			 WHERE is_active ORDER BY sort_order, slot_time LIMIT 1`).
			Scan(&slotID).Error; err != nil {
			return fmt.Errorf("seed: reading the delivery slot: %w", err)
		}
		if slotID == "" {
			return fmt.Errorf("seed: no active delivery slot — seed the slots first")
		}

		// ── Menus ───────────────────────────────────────────────────────────
		type dt struct {
			ID   string
			Slug string
		}
		var diets []dt
		if err := tx.Raw(
			`SELECT id::text AS id, slug FROM diet_type WHERE is_active ORDER BY sort_order`).
			Scan(&diets).Error; err != nil {
			return fmt.Errorf("seed: reading diet types: %w", err)
		}

		// Business dates, not server-local ones: a menu is scheduled against
		// the operating day in Jakarta (CLAUDE.md §10).
		loc, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			loc = time.UTC
		}
		today := time.Now().In(loc)

		created := 0
		for d := 0; d < days; d++ {
			date := today.AddDate(0, 0, d).Format("2006-01-02")
			for _, diet := range diets {
				variants, ok := menus[diet.Slug]
				if !ok || len(variants) == 0 {
					// A diet type added later has no sample menu; skip it
					// rather than inventing one.
					log.Warn("seed-menu: no sample menu for diet type", "slug", diet.Slug)
					continue
				}
				m := variants[d%len(variants)]

				mealID := uuid.Must(uuid.NewV7())
				res := tx.Exec(`
					INSERT INTO scheduled_meal
					  (id, service_date, diet_type_id, slot_id, name, qty_capacity,
					   status, published_at)
					VALUES (?, ?, ?, ?, ?, 40, 'PUBLISHED', now())
					ON CONFLICT (service_date, diet_type_id, slot_id) DO NOTHING`,
					mealID, date, diet.ID, slotID, m.name)
				if res.Error != nil {
					return fmt.Errorf("seed meal %s/%s: %w", date, diet.Slug, res.Error)
				}
				if res.RowsAffected == 0 {
					continue // already scheduled; leave the real one alone
				}
				created++

				for i, slug := range m.items {
					role := "SIDE"
					if i == 0 {
						role = "MAIN"
					}
					if err := tx.Exec(`
						INSERT INTO scheduled_meal_item
						  (id, scheduled_meal_id, food_id, item_role, sort_order)
						SELECT ?, ?, f.id, ?, ? FROM food f WHERE f.slug = ?
						ON CONFLICT (scheduled_meal_id, food_id) DO NOTHING`,
						uuid.Must(uuid.NewV7()), mealID, role, i, slug).Error; err != nil {
						return fmt.Errorf("seed item %s on %s: %w", slug, m.name, err)
					}
				}
			}
		}

		log.Info("seed-menu complete",
			"days", days, "diet_types", len(diets), "meals_created", created,
			"note", "existing meals were left untouched")
		return nil
	})
}

// seedDays reads the optional day count from the command line.
func seedDays(args []string) (int, error) {
	if len(args) < 3 {
		return 3, nil
	}
	n, err := strconv.Atoi(args[2])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("seed-menu: day count must be a positive integer, got %q", args[2])
	}
	return n, nil
}
