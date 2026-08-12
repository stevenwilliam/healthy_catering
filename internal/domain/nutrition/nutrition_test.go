package nutrition

import "testing"

// D-33: a meal's panel is the sum of its foods'. Integers make the sum exact.

func food(name, role string, kcal, protein int, complete bool) Item {
	return Item{
		FoodID: name, Name: name, Role: role,
		Facts: Facts{CaloriesKcal: kcal, ProteinMg: protein * 1000, Complete: complete},
	}
}

func TestAggregateSumsItems(t *testing.T) {
	meal := Aggregate([]Item{
		food("Ayam bakar", "MAIN", 320, 35, true),
		food("Tumis buncis", "SIDE", 80, 3, true),
		food("Pepaya", "DESSERT", 60, 1, true),
	})
	if meal.Total.CaloriesKcal != 460 {
		t.Errorf("kcal = %d, want 460", meal.Total.CaloriesKcal)
	}
	if meal.Total.ProteinMg != 39_000 {
		t.Errorf("protein = %d mg, want 39000", meal.Total.ProteinMg)
	}
	if !meal.Complete {
		t.Error("all items complete means the meal is complete")
	}
}

// A single-food meal is still one meal — the credit rule does not change and
// neither does the aggregation.
func TestAggregateSingleFood(t *testing.T) {
	meal := Aggregate([]Item{food("Salad", "MAIN", 210, 12, true)})
	if meal.Total.CaloriesKcal != 210 || len(meal.Items) != 1 {
		t.Errorf("got %d kcal over %d items, want 210 over 1", meal.Total.CaloriesKcal, len(meal.Items))
	}
}

// One unfilled panel makes the total a lower bound. Presenting a lower bound as
// a fact is how a diabetic customer gets a number they cannot rely on.
func TestIncompleteFoodMarksTheMealIncomplete(t *testing.T) {
	meal := Aggregate([]Item{
		food("Ayam bakar", "MAIN", 320, 35, true),
		food("Sambal", "SIDE", 0, 0, false),
	})
	if meal.Complete || meal.Total.Complete {
		t.Error("a meal containing an unfilled panel must not claim to be complete")
	}
	if meal.Total.CaloriesKcal != 320 {
		t.Errorf("kcal = %d, want the known 320 still reported", meal.Total.CaloriesKcal)
	}
}

func TestAggregateEmpty(t *testing.T) {
	meal := Aggregate(nil)
	if meal.Complete {
		t.Error("an empty meal is not a complete meal")
	}
	if meal.Total.CaloriesKcal != 0 {
		t.Error("empty total must be zero")
	}
}

func TestAggregateOrdersByRoleThenSort(t *testing.T) {
	meal := Aggregate([]Item{
		{FoodID: "d", Role: "DRINK", Sort: 0},
		{FoodID: "m", Role: "MAIN", Sort: 0},
		{FoodID: "s", Role: "SIDE", Sort: 0},
	})
	want := []string{"m", "s", "d"}
	for i, w := range want {
		if meal.Items[i].FoodID != w {
			t.Errorf("position %d = %s, want %s (main, side, drink)", i, meal.Items[i].FoodID, w)
		}
	}

	// An explicit sort order wins over the role ranking.
	meal = Aggregate([]Item{
		{FoodID: "m", Role: "MAIN", Sort: 5},
		{FoodID: "s", Role: "SIDE", Sort: 1},
	})
	if meal.Items[0].FoodID != "s" {
		t.Error("an explicit sort order must win over the role ranking")
	}
}

func TestExtrasSumByKey(t *testing.T) {
	meal := Aggregate([]Item{
		{FoodID: "a", Facts: Facts{Extras: map[string]int{"potassium_mg": 300, "iron_mg": 2}, Complete: true}},
		{FoodID: "b", Facts: Facts{Extras: map[string]int{"potassium_mg": 150}, Complete: true}},
	})
	if meal.Total.Extras["potassium_mg"] != 450 {
		t.Errorf("potassium = %d, want 450", meal.Total.Extras["potassium_mg"])
	}
	if meal.Total.Extras["iron_mg"] != 2 {
		t.Errorf("iron = %d, want 2", meal.Total.Extras["iron_mg"])
	}
}

// Integers are exact where decimals drift. A week of meals must sum to the
// rupiah-equivalent of exactness.
func TestSumsAreExactOverAWeek(t *testing.T) {
	one := Aggregate([]Item{food("x", "MAIN", 333, 11, true)}).Total
	var totalKcal, totalProtein int
	for i := 0; i < 7; i++ {
		totalKcal += one.CaloriesKcal
		totalProtein += one.ProteinMg
	}
	if totalKcal != 2331 || totalProtein != 77_000 {
		t.Errorf("week = %d kcal / %d mg, want 2331 / 77000", totalKcal, totalProtein)
	}
}

func TestScale(t *testing.T) {
	f := Facts{CaloriesKcal: 460, ProteinMg: 39_000, Complete: true,
		Extras: map[string]int{"potassium_mg": 100}}
	got := Scale(f, 3)
	if got.CaloriesKcal != 1380 || got.ProteinMg != 117_000 {
		t.Errorf("got %d kcal / %d mg, want 1380 / 117000", got.CaloriesKcal, got.ProteinMg)
	}
	if got.Extras["potassium_mg"] != 300 {
		t.Errorf("extras scaled to %d, want 300", got.Extras["potassium_mg"])
	}
	if z := Scale(f, 0); z.CaloriesKcal != 0 {
		t.Error("scaling by zero must zero the panel")
	}
}

func TestGramsDisplay(t *testing.T) {
	tests := []struct {
		mg   int
		want string
	}{
		{12_500, "12.5"},
		{39_000, "39"},
		{0, "0"},
		{500, "0.5"},
		{1_050, "1"}, // rounds down to one decimal place
	}
	for _, tc := range tests {
		if got := Grams(tc.mg); got != tc.want {
			t.Errorf("Grams(%d) = %q, want %q", tc.mg, got, tc.want)
		}
	}
}
