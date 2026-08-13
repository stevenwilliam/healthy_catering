// Package security_test is the security suite that ships with the product
// (99 §7): negative authz per role, IDOR per resource, rate limits, injection,
// concurrency, and JWT tampering.
//
// It runs against a REAL database — the controls being tested are enforced by
// constraints, locks and repository scoping, and an in-memory fake would prove
// nothing about any of them. Set TEST_DATABASE_URL to run it; without it the
// suite skips loudly rather than passing silently.
package security_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

var db *sql.DB

func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr,
			"SKIP: security suite needs TEST_DATABASE_URL (see docs/RUN-WHEN-BACK.md §0)")
		os.Exit(0)
	}
	var err error
	db, err = sql.Open("pgx", url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "security suite: open: %v\n", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "security suite: ping: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// ── Injection ───────────────────────────────────────────────────────────────

// Every query uses placeholders; this proves a classic payload lands as DATA.
func TestSQLInjectionIsData(t *testing.T) {
	ctx := context.Background()
	payloads := []string{
		"'; DROP TABLE customer_order; --",
		"' OR '1'='1",
		"\\'; DELETE FROM app_user WHERE '1'='1",
		"admin'--",
		"1; SELECT pg_sleep(10)",
	}

	for _, p := range payloads {
		t.Run(truncate(p), func(t *testing.T) {
			var n int
			// The same shape as a real search: parameterised, never concatenated.
			err := db.QueryRowContext(ctx,
				`SELECT count(*) FROM diet_type WHERE lower(name) LIKE '%' || lower($1) || '%'`,
				p).Scan(&n)
			if err != nil {
				t.Fatalf("payload should be inert data, got an error: %v", err)
			}
			if n != 0 {
				t.Errorf("payload matched %d rows — it was interpreted, not stored", n)
			}
		})
	}

	// And the tables are all still here.
	var tables int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM information_schema.tables WHERE table_schema='public'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables < 30 {
		t.Fatalf("only %d tables remain — something executed", tables)
	}
}

// ── Concurrency: the two invariants that cost money ────────────────────────

// A meal cannot be oversold, whatever the race (PROMPT §6, §14).
func TestCapacityCannotOversell(t *testing.T) {
	ctx := context.Background()
	mealID := seedMeal(t, 5) // capacity 5
	defer cleanupMeal(t, mealID)

	const attempts = 20
	var wg sync.WaitGroup
	results := make([]bool, attempts)

	for i := range attempts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := db.ExecContext(ctx, `
				UPDATE scheduled_meal SET qty_reserved = qty_reserved + 1
				 WHERE id = $1 AND (qty_capacity IS NULL OR qty_reserved + 1 <= qty_capacity)`,
				mealID)
			if err != nil {
				return
			}
			n, _ := res.RowsAffected()
			results[i] = n == 1
		}(i)
	}
	wg.Wait()

	won := 0
	for _, ok := range results {
		if ok {
			won++
		}
	}
	if won != 5 {
		t.Errorf("%d of %d writers won, want exactly 5", won, attempts)
	}

	var reserved, capacity int
	if err := db.QueryRowContext(ctx,
		`SELECT qty_reserved, qty_capacity FROM scheduled_meal WHERE id=$1`,
		mealID).Scan(&reserved, &capacity); err != nil {
		t.Fatal(err)
	}
	if reserved > capacity {
		t.Fatalf("OVERSOLD: reserved %d of %d", reserved, capacity)
	}
}

// The database refuses the oversell even if the application forgets its guard.
func TestCheckConstraintIsTheLastLine(t *testing.T) {
	mealID := seedMeal(t, 2)
	defer cleanupMeal(t, mealID)

	_, err := db.Exec(`UPDATE scheduled_meal SET qty_reserved = 99 WHERE id=$1`, mealID)
	if err == nil {
		t.Fatal("the database accepted an oversell; the CHECK constraint is missing")
	}
	if !strings.Contains(err.Error(), "no_oversell") {
		t.Errorf("refused for the wrong reason: %v", err)
	}
}

// ── Append-only history ─────────────────────────────────────────────────────

func TestLedgerAndAuditAreAppendOnly(t *testing.T) {
	for _, table := range []string{"credit_ledger", "audit_log"} {
		t.Run(table, func(t *testing.T) {
			// A statement that matches no rows still fires the trigger, so this
			// proves the rule without needing a fixture row.
			_, err := db.Exec(fmt.Sprintf(`UPDATE %s SET id = id WHERE false`, table))
			if err == nil {
				// A WHERE false UPDATE touches no rows and a row-level trigger
				// will not fire, so try a real one against any existing row.
				var id string
				if e := db.QueryRow(fmt.Sprintf(`SELECT id FROM %s LIMIT 1`, table)).Scan(&id); e == nil {
					_, err = db.Exec(fmt.Sprintf(`UPDATE %s SET id = id WHERE id = $1`, table), id)
					if err == nil {
						t.Fatalf("%s accepted an UPDATE — history can be rewritten", table)
					}
				} else {
					t.Skipf("%s is empty; nothing to attempt an update against", table)
				}
			}
			if !strings.Contains(err.Error(), "append-only") {
				t.Errorf("%s refused for the wrong reason: %v", table, err)
			}
		})
	}
}

// ── Price integrity ─────────────────────────────────────────────────────────

// The exclusion constraint, not an application check, is what stops two prices
// covering the same date for the same scope (PROMPT §5.3).
func TestPriceOverlapRefusedUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	// The fixture is created fresh rather than borrowed from seeded data. The
	// first version reused an existing diet type and tier, and every insert was
	// correctly refused — because a real OPEN-ENDED price from 2026-01-01
	// covers the year 2099 too. The constraint was right and the test was
	// wrong, which is the more dangerous way round.
	dietID, tierID := seedIsolatedDietAndTier(t)
	defer cleanupIsolated(t, dietID, tierID)

	const writers = 10
	var wg sync.WaitGroup
	accepted := make([]bool, writers)

	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := db.ExecContext(ctx, `
				INSERT INTO meal_price_normal
				  (id, diet_type_id, tier_id, unit_price_idr, valid_from, valid_to, is_active)
				VALUES ($1,$2,$3,$4,'2099-01-01','2099-02-01',TRUE)`,
				uuid.New(), dietID, tierID, 50000+i)
			accepted[i] = err == nil
		}(i)
	}
	wg.Wait()

	won := 0
	for _, ok := range accepted {
		if ok {
			won++
		}
	}
	if won != 1 {
		t.Errorf("%d of %d overlapping prices were accepted, want exactly 1", won, writers)
	}
}

// ── Money integrity ─────────────────────────────────────────────────────────

// Every stored order must reconcile: base + tax = total, and payment = total +
// the unique suffix. A drift here is an invoice that disagrees with itself.
func TestEveryOrderReconciles(t *testing.T) {
	rows, err := db.Query(`
		SELECT order_code, total_idr, tax_base_idr, tax_idr,
		       payment_amount_idr, payment_rounding_idr,
		       subtotal_idr, delivery_fee_idr, discount_idr
		  FROM customer_order`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	checked := 0
	for rows.Next() {
		var code string
		var total, base, tax, pay, rounding, subtotal, fee, discount int64
		if err := rows.Scan(&code, &total, &base, &tax, &pay, &rounding,
			&subtotal, &fee, &discount); err != nil {
			t.Fatal(err)
		}
		checked++
		if base+tax != total {
			t.Errorf("%s: base %d + tax %d != total %d", code, base, tax, total)
		}
		if pay != total+rounding {
			t.Errorf("%s: payment %d != total %d + rounding %d", code, pay, total, rounding)
		}
		if total != subtotal+fee-discount {
			t.Errorf("%s: total %d != subtotal %d + fee %d - discount %d",
				code, total, subtotal, fee, discount)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating orders: %v", err)
	}
	if checked == 0 {
		t.Skip("no orders to reconcile yet")
	}
	t.Logf("reconciled %d orders", checked)
}

// A credit balance can never go negative — the ledger is the source of truth.
func TestNoNegativeCreditBalances(t *testing.T) {
	rows, err := db.Query(`
		SELECT cp.id, COALESCE(SUM(cl.qty),0) AS balance
		  FROM customer_package cp
		  LEFT JOIN credit_ledger cl ON cl.customer_package_id = cp.id
		 GROUP BY cp.id HAVING COALESCE(SUM(cl.qty),0) < 0`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var balance int
		_ = rows.Scan(&id, &balance)
		t.Errorf("package %s has a NEGATIVE balance of %d — a credit was double-spent", id, balance)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating balances: %v", err)
	}
}

// One REDEEM per delivery: a retry must not spend a second credit.
func TestOneRedeemPerDelivery(t *testing.T) {
	rows, err := db.Query(`
		SELECT reference_id, count(*)
		  FROM credit_ledger
		 WHERE entry_type = 'REDEEM' AND reference_id IS NOT NULL
		 GROUP BY reference_id HAVING count(*) > 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		_ = rows.Scan(&id, &n)
		t.Errorf("delivery %s spent %d credits — the unique index is not holding", id, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating redemptions: %v", err)
	}
}

// ── PII ─────────────────────────────────────────────────────────────────────

// Coordinates are personal data under UU PDP (PROMPT §14), so a customer's
// address must never be readable without its owner id in the query. This
// asserts the SHAPE of the table rather than a query, so a new query that
// forgets the scope still has to pass review.
func TestAddressesAreOwnedRows(t *testing.T) {
	var hasCustomer bool
	err := db.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM information_schema.columns
		                WHERE table_name='customer_address' AND column_name='customer_id'
		                  AND is_nullable='NO')`).Scan(&hasCustomer)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCustomer {
		t.Fatal("customer_address has no mandatory customer_id — ownership cannot be enforced")
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func seedMeal(t *testing.T, capacity int) uuid.UUID {
	t.Helper()
	var dietID, slotID uuid.UUID
	if err := db.QueryRow(`SELECT id FROM diet_type LIMIT 1`).Scan(&dietID); err != nil {
		t.Skipf("no diet types seeded: %v", err)
	}
	if err := db.QueryRow(`SELECT id FROM delivery_time_slot LIMIT 1`).Scan(&slotID); err != nil {
		t.Skipf("no slots seeded: %v", err)
	}

	id := uuid.New()
	date := time.Now().AddDate(0, 6, 0).Format("2006-01-02")
	_, err := db.Exec(`
		INSERT INTO scheduled_meal (id, service_date, diet_type_id, slot_id,
		                            qty_capacity, status, published_at)
		VALUES ($1, $2::date + (random()*300)::int, $3, $4, $5, 'PUBLISHED', now())`,
		id, date, dietID, slotID, capacity)
	if err != nil {
		t.Fatalf("seed meal: %v", err)
	}
	return id
}

func cleanupMeal(t *testing.T, id uuid.UUID) {
	t.Helper()
	_, _ = db.Exec(`DELETE FROM scheduled_meal WHERE id=$1`, id)
}

// seedIsolatedDietAndTier creates a diet type and tier that no existing price
// row can possibly reference, so the race tests only the constraint.
func seedIsolatedDietAndTier(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	dietID, tierID := uuid.New(), uuid.New()
	suffix := dietID.String()[:8]

	if _, err := db.Exec(`
		INSERT INTO diet_type (id, name, slug, is_active)
		VALUES ($1, $2, $3, FALSE)`,
		dietID, "sec-test-"+suffix, "sec-test-"+suffix); err != nil {
		t.Fatalf("seed diet type: %v", err)
	}
	// A tier far above the configured range, so it cannot overlap the real
	// tiers the exclusion constraint also guards.
	if _, err := db.Exec(`
		INSERT INTO meal_price_tier (id, label, min_qty, max_qty, is_active)
		VALUES ($1, $2, 900000, 900001, FALSE)`,
		tierID, "sec-test-"+suffix); err != nil {
		t.Fatalf("seed tier: %v", err)
	}
	return dietID, tierID
}

func cleanupIsolated(t *testing.T, dietID, tierID uuid.UUID) {
	t.Helper()
	_, _ = db.Exec(`DELETE FROM meal_price_normal WHERE diet_type_id=$1`, dietID)
	_, _ = db.Exec(`DELETE FROM meal_price_tier WHERE id=$1`, tierID)
	_, _ = db.Exec(`DELETE FROM diet_type WHERE id=$1`, dietID)
}

func truncate(s string) string {
	if len(s) > 24 {
		return s[:24]
	}
	return s
}
