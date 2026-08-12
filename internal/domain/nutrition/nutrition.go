// Package nutrition aggregates a meal's facts from its foods.
//
// Steven, 2026-08-12: "each food can have nutrition fact; meal nutrition will
// be aggregated from food" (D-33). Nutrition is therefore never typed at the
// meal level — it is summed, in pure code, and snapshotted onto the order line
// at purchase so a later recipe edit cannot rewrite what someone was told they
// ate.
//
// Every field is an integer: calories in whole kcal, everything else in
// milligrams (D-24). That is what makes the sum exact — decimals would drift a
// few milligrams per dish and visibly per week on the intake chart.
package nutrition

import "sort"

// Facts is one panel, per portion.
type Facts struct {
	CaloriesKcal   int            `json:"calories_kcal"`
	ProteinMg      int            `json:"protein_mg"`
	FatMg          int            `json:"fat_mg"`
	SaturatedFatMg int            `json:"saturated_fat_mg"`
	CarbohydrateMg int            `json:"carbohydrate_mg"`
	SugarMg        int            `json:"sugar_mg"`
	FibreMg        int            `json:"fibre_mg"`
	SodiumMg       int            `json:"sodium_mg"`
	CholesterolMg  int            `json:"cholesterol_mg"`
	Extras         map[string]int `json:"extras,omitempty"`
	// Complete is false when the food's panel has not been filled in. It
	// propagates to the meal, so an incomplete meal says so rather than
	// under-reporting.
	Complete bool `json:"complete"`
}

// Item is one food in a meal.
type Item struct {
	FoodID string
	Name   string
	Role   string // MAIN | SIDE | DESSERT | DRINK
	Sort   int
	Facts  Facts
}

// Meal is the aggregate the customer sees.
type Meal struct {
	Total    Facts  `json:"total"`
	Items    []Item `json:"items"`
	Complete bool   `json:"complete"`
}

// Aggregate sums the items into a meal panel.
//
// The meal is complete only if every item is. One unfilled panel makes the
// total a lower bound, and presenting a lower bound as a fact is how a diabetic
// customer gets a number they cannot rely on.
func Aggregate(items []Item) Meal {
	out := Meal{Complete: len(items) > 0}
	total := Facts{Complete: true}

	sorted := make([]Item, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Sort != sorted[j].Sort {
			return sorted[i].Sort < sorted[j].Sort
		}
		return roleRank(sorted[i].Role) < roleRank(sorted[j].Role)
	})

	for _, it := range sorted {
		f := it.Facts
		total.CaloriesKcal += f.CaloriesKcal
		total.ProteinMg += f.ProteinMg
		total.FatMg += f.FatMg
		total.SaturatedFatMg += f.SaturatedFatMg
		total.CarbohydrateMg += f.CarbohydrateMg
		total.SugarMg += f.SugarMg
		total.FibreMg += f.FibreMg
		total.SodiumMg += f.SodiumMg
		total.CholesterolMg += f.CholesterolMg

		// Numeric extras sum by key; anything else is dropped from the
		// aggregate rather than guessed at.
		for k, v := range f.Extras {
			if total.Extras == nil {
				total.Extras = map[string]int{}
			}
			total.Extras[k] += v
		}
		if !f.Complete {
			total.Complete = false
			out.Complete = false
		}
	}

	out.Total = total
	out.Items = sorted
	return out
}

// Scale multiplies a panel by a portion count, for a line of several meals.
func Scale(f Facts, qty int) Facts {
	if qty <= 0 {
		return Facts{Complete: f.Complete}
	}
	out := Facts{
		CaloriesKcal:   f.CaloriesKcal * qty,
		ProteinMg:      f.ProteinMg * qty,
		FatMg:          f.FatMg * qty,
		SaturatedFatMg: f.SaturatedFatMg * qty,
		CarbohydrateMg: f.CarbohydrateMg * qty,
		SugarMg:        f.SugarMg * qty,
		FibreMg:        f.FibreMg * qty,
		SodiumMg:       f.SodiumMg * qty,
		CholesterolMg:  f.CholesterolMg * qty,
		Complete:       f.Complete,
	}
	for k, v := range f.Extras {
		if out.Extras == nil {
			out.Extras = map[string]int{}
		}
		out.Extras[k] = v * qty
	}
	return out
}

// Grams renders a milligram value for display, e.g. 12500 → "12.5". Display
// divides at the edge; storage never does.
func Grams(mg int) string {
	whole := mg / 1000
	frac := (mg % 1000) / 100
	if frac == 0 {
		return itoa(whole)
	}
	return itoa(whole) + "." + itoa(frac)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func roleRank(role string) int {
	switch role {
	case "MAIN":
		return 0
	case "SIDE":
		return 1
	case "DESSERT":
		return 2
	case "DRINK":
		return 3
	default:
		return 4
	}
}
