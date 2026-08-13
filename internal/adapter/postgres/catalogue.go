package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/nutrition"
)

// CatalogueRepo owns foods, their nutrition, photos, allergens and diet
// mappings.
type CatalogueRepo struct{ db *gorm.DB }

func NewCatalogueRepo(db *gorm.DB) *CatalogueRepo { return &CatalogueRepo{db: db} }

// Food is a dish. Nutrition lives on it, and a meal's panel is the sum of its
// foods' (docs/02 D-33).
type Food struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	PortionSize string    `json:"portion_size"`
	IsActive    bool      `json:"is_active"`

	CaloriesKcal   int  `json:"calories_kcal"`
	ProteinMg      int  `json:"protein_mg"`
	FatMg          int  `json:"fat_mg"`
	SaturatedFatMg int  `json:"saturated_fat_mg"`
	CarbohydrateMg int  `json:"carbohydrate_mg"`
	SugarMg        int  `json:"sugar_mg"`
	FibreMg        int  `json:"fibre_mg"`
	SodiumMg       int  `json:"sodium_mg"`
	CholesterolMg  int  `json:"cholesterol_mg"`
	IsComplete     bool `json:"nutrition_complete"`

	DietTypeIDs []uuid.UUID `json:"diet_type_ids" gorm:"-"`
	AllergenIDs []uuid.UUID `json:"allergen_ids"  gorm:"-"`
	PhotoKeys   []string    `json:"photo_keys"    gorm:"-"`
}

// Facts converts the stored columns into the domain panel.
func (f Food) Facts() nutrition.Facts {
	return nutrition.Facts{
		CaloriesKcal: f.CaloriesKcal, ProteinMg: f.ProteinMg, FatMg: f.FatMg,
		SaturatedFatMg: f.SaturatedFatMg, CarbohydrateMg: f.CarbohydrateMg,
		SugarMg: f.SugarMg, FibreMg: f.FibreMg, SodiumMg: f.SodiumMg,
		CholesterolMg: f.CholesterolMg, Complete: f.IsComplete,
	}
}

// ListFoods returns a searchable page.
//
// The search covers the allergen names too: a kitchen lead asked to pull
// everything with peanuts should be able to type "kacang" into the food list
// and get an answer, rather than cross-referencing two screens.
func (r *CatalogueRepo) ListFoods(ctx context.Context, p ListParams, dietTypeID *uuid.UUID) (Page[Food], error) {
	p = p.Normalise("name", "name", "slug", "calories_kcal")
	pattern := SearchPattern(p.Q)

	base := r.db.WithContext(ctx).Table("food f").
		Joins("LEFT JOIN food_nutrition n ON n.food_id = f.id")
	if p.Q != "" {
		base = base.Where(`lower(f.name) LIKE ? OR lower(f.slug) LIKE ? OR lower(f.description) LIKE ?
			OR EXISTS (SELECT 1 FROM food_allergen fa JOIN allergen a ON a.id = fa.allergen_id
			            WHERE fa.food_id = f.id
			              AND (lower(a.name_id) LIKE ? OR lower(a.name_en) LIKE ? OR lower(a.code) LIKE ?))`,
			pattern, pattern, pattern, pattern, pattern, pattern)
	}
	if p.Active != nil {
		base = base.Where("f.is_active = ?", *p.Active)
	}
	if dietTypeID != nil {
		base = base.Where(`EXISTS (SELECT 1 FROM food_diet_type fd
		                            WHERE fd.food_id = f.id AND fd.diet_type_id = ?)`, *dietTypeID)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[Food]{}, fmt.Errorf("postgres: count foods: %w", err)
	}

	var items []Food
	err := base.Session(&gorm.Session{}).
		Select(`f.id, f.name, f.slug, f.description, f.portion_size, f.is_active,
		        COALESCE(n.calories_kcal,0) AS calories_kcal, COALESCE(n.protein_mg,0) AS protein_mg,
		        COALESCE(n.fat_mg,0) AS fat_mg, COALESCE(n.saturated_fat_mg,0) AS saturated_fat_mg,
		        COALESCE(n.carbohydrate_mg,0) AS carbohydrate_mg, COALESCE(n.sugar_mg,0) AS sugar_mg,
		        COALESCE(n.fibre_mg,0) AS fibre_mg, COALESCE(n.sodium_mg,0) AS sodium_mg,
		        COALESCE(n.cholesterol_mg,0) AS cholesterol_mg,
		        COALESCE(n.is_complete,false) AS is_complete`).
		Order("f." + p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[Food]{}, fmt.Errorf("postgres: list foods: %w", err)
	}
	return NewPage(items, total, p), nil
}

// GetFood loads one food with its relations.
func (r *CatalogueRepo) GetFood(ctx context.Context, id uuid.UUID) (Food, error) {
	var f Food
	err := r.db.WithContext(ctx).Raw(`
		SELECT f.id, f.name, f.slug, f.description, f.portion_size, f.is_active,
		       COALESCE(n.calories_kcal,0) AS calories_kcal, COALESCE(n.protein_mg,0) AS protein_mg,
		       COALESCE(n.fat_mg,0) AS fat_mg, COALESCE(n.saturated_fat_mg,0) AS saturated_fat_mg,
		       COALESCE(n.carbohydrate_mg,0) AS carbohydrate_mg, COALESCE(n.sugar_mg,0) AS sugar_mg,
		       COALESCE(n.fibre_mg,0) AS fibre_mg, COALESCE(n.sodium_mg,0) AS sodium_mg,
		       COALESCE(n.cholesterol_mg,0) AS cholesterol_mg,
		       COALESCE(n.is_complete,false) AS is_complete
		  FROM food f LEFT JOIN food_nutrition n ON n.food_id = f.id
		 WHERE f.id = ?`, id).Scan(&f).Error
	if err != nil {
		return Food{}, fmt.Errorf("postgres: get food: %w", err)
	}
	if f.ID == uuid.Nil {
		return Food{}, ErrNotFound
	}
	if err := r.db.WithContext(ctx).Raw(
		`SELECT diet_type_id FROM food_diet_type WHERE food_id = ?`, id).Scan(&f.DietTypeIDs).Error; err != nil {
		return Food{}, fmt.Errorf("postgres: food diets: %w", err)
	}
	if err := r.db.WithContext(ctx).Raw(
		`SELECT allergen_id FROM food_allergen WHERE food_id = ?`, id).Scan(&f.AllergenIDs).Error; err != nil {
		return Food{}, fmt.Errorf("postgres: food allergens: %w", err)
	}
	if err := r.db.WithContext(ctx).Raw(
		`SELECT object_key FROM food_photo WHERE food_id = ? ORDER BY sort_order`, id).Scan(&f.PhotoKeys).Error; err != nil {
		return Food{}, fmt.Errorf("postgres: food photos: %w", err)
	}
	return f, nil
}

// SaveFood inserts or updates a food with its nutrition and relations, in one
// transaction.
//
// Nutrition is written with the food rather than in a second call: a dish that
// exists for a few seconds with no panel is a dish that can be scheduled with
// no panel, and the meal aggregate would silently under-report.
func (r *CatalogueRepo) SaveFood(ctx context.Context, f Food, by uuid.UUID) (uuid.UUID, error) {
	id := f.ID
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id == uuid.Nil {
			id = uuid.Must(uuid.NewV7())
			if err := tx.Exec(`
				INSERT INTO food (id, name, slug, description, portion_size, is_active, updated_by)
				VALUES (?,?,?,?,?,?,?)`,
				id, f.Name, f.Slug, f.Description, f.PortionSize, f.IsActive, by).Error; err != nil {
				return err
			}
		} else {
			res := tx.Exec(`
				UPDATE food SET name=?, slug=?, description=?, portion_size=?, is_active=?, updated_by=?
				 WHERE id=?`,
				f.Name, f.Slug, f.Description, f.PortionSize, f.IsActive, by, id)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrNotFound
			}
		}

		if err := tx.Exec(`
			INSERT INTO food_nutrition (food_id, calories_kcal, protein_mg, fat_mg, saturated_fat_mg,
			                            carbohydrate_mg, sugar_mg, fibre_mg, sodium_mg, cholesterol_mg,
			                            is_complete, updated_by)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT (food_id) DO UPDATE SET
			  calories_kcal=EXCLUDED.calories_kcal, protein_mg=EXCLUDED.protein_mg,
			  fat_mg=EXCLUDED.fat_mg, saturated_fat_mg=EXCLUDED.saturated_fat_mg,
			  carbohydrate_mg=EXCLUDED.carbohydrate_mg, sugar_mg=EXCLUDED.sugar_mg,
			  fibre_mg=EXCLUDED.fibre_mg, sodium_mg=EXCLUDED.sodium_mg,
			  cholesterol_mg=EXCLUDED.cholesterol_mg, is_complete=EXCLUDED.is_complete,
			  updated_by=EXCLUDED.updated_by`,
			id, f.CaloriesKcal, f.ProteinMg, f.FatMg, f.SaturatedFatMg, f.CarbohydrateMg,
			f.SugarMg, f.FibreMg, f.SodiumMg, f.CholesterolMg, f.IsComplete, by).Error; err != nil {
			return err
		}

		if err := tx.Exec(`DELETE FROM food_diet_type WHERE food_id = ?`, id).Error; err != nil {
			return err
		}
		for _, dt := range f.DietTypeIDs {
			if err := tx.Exec(
				`INSERT INTO food_diet_type (food_id, diet_type_id) VALUES (?,?)`, id, dt).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec(`DELETE FROM food_allergen WHERE food_id = ?`, id).Error; err != nil {
			return err
		}
		for _, a := range f.AllergenIDs {
			if err := tx.Exec(
				`INSERT INTO food_allergen (food_id, allergen_id) VALUES (?,?)`, id, a).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// FoodAllergenNames returns the allergen labels for a set of foods, for the
// packing label and the clash warning.
func (r *CatalogueRepo) FoodAllergenNames(ctx context.Context, foodIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	out := map[uuid.UUID][]string{}
	if len(foodIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		FoodID uuid.UUID
		Code   string
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT fa.food_id, a.code
		  FROM food_allergen fa JOIN allergen a ON a.id = fa.allergen_id
		 WHERE fa.food_id IN ? AND a.is_active`, foodIDs).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: food allergens: %w", err)
	}
	for _, r := range rows {
		out[r.FoodID] = append(out[r.FoodID], r.Code)
	}
	return out, nil
}
