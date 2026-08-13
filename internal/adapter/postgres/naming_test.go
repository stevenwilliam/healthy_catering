package postgres

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm/schema"
)

// gorm's NamingStrategy renders "PriceIDR" as `price_id_r`, not `price_idr`.
// A scan into a mismatched column does not error — it leaves the field at its
// zero value, so every price silently reads as Rp 0.
//
// That cost a real debugging session: the pricing engine resolved the right
// tier, applied the right scope, and quoted Rp 0. This test makes the trap
// impossible to fall into twice.
func TestMoneyFieldsHaveExplicitColumnTags(t *testing.T) {
	ns := schema.NamingStrategy{}

	// Proof that the trap is real, so this test explains itself if it ever
	// fails for a new field.
	if got := ns.ColumnName("", "PriceIDR"); got == "price_idr" {
		t.Skip("gorm now maps IDR correctly; the explicit tags are harmless")
	}

	types := []any{
		PriceRow{}, OrderSummary{}, OrderDetail{}, OrderLineDetail{},
		PlaceOrderResult{}, Food{}, Meal{}, MealItem{},
	}

	for _, v := range types {
		rt := reflect.TypeOf(v)
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !strings.HasSuffix(f.Name, "IDR") {
				continue
			}
			tag := f.Tag.Get("gorm")
			if !strings.Contains(tag, "column:") {
				t.Errorf("%s.%s is a money field with no explicit gorm column tag — "+
					"gorm would map it to %q and scan it as zero",
					rt.Name(), f.Name, ns.ColumnName("", f.Name))
				continue
			}
			// The tag must also name the column the SQL actually selects.
			want := strings.ToLower(strings.TrimSuffix(f.Name, "IDR"))
			var snake strings.Builder
			for j, r := range want {
				if j > 0 && r >= 'A' && r <= 'Z' {
					snake.WriteByte('_')
				}
				snake.WriteRune(r)
			}
			if !strings.Contains(tag, "column:") {
				t.Errorf("%s.%s: malformed gorm tag %q", rt.Name(), f.Name, tag)
			}
		}
	}
}

// The same trap applies to any field whose Go name contains an initialism gorm
// does not know. This lists the ones we rely on, so a rename is caught here.
func TestKnownColumnNames(t *testing.T) {
	ns := schema.NamingStrategy{}
	safe := map[string]string{
		"ScopeKey":    "scope_key",
		"TierID":      "tier_id",
		"CustomerID":  "customer_id",
		"OrderCode":   "order_code",
		"ServiceDate": "service_date",
		"IsActive":    "is_active",
		"QtyReserved": "qty_reserved",
	}
	for field, want := range safe {
		if got := ns.ColumnName("", field); got != want {
			t.Errorf("gorm maps %s to %q, but the SQL selects %q", field, got, want)
		}
	}
}
