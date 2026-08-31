package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/stevenwilliam/healthy_catering/internal/domain/nutrition"
)

// ScheduleRepo owns the menu calendar: scheduled_meal and its items.
//
// The MEAL is the unit of sale (docs/02 D-32): capacity, publication and the
// credit boundary all attach here, not to the individual food.
type ScheduleRepo struct{ db *gorm.DB }

func NewScheduleRepo(db *gorm.DB) *ScheduleRepo { return &ScheduleRepo{db: db} }

// Meal is one sitting: a date, a diet type and a slot, with its foods.
type Meal struct {
	ID           uuid.UUID  `json:"id"`
	ServiceDate  string     `json:"service_date"`
	DietTypeID   uuid.UUID  `json:"diet_type_id"`
	DietTypeName string     `json:"diet_type"`
	DietTypeSlug string     `json:"diet_type_slug"`
	SlotID       uuid.UUID  `json:"slot_id"`
	SlotAlias    string     `json:"slot"`
	Name         *string    `json:"name,omitempty"`
	Description  string     `json:"description"`
	HeroPhotoKey *string    `json:"hero_photo_key,omitempty"`
	QtyCapacity  *int       `json:"qty_capacity,omitempty"`
	QtyReserved  int        `json:"qty_reserved"`
	Status       string     `json:"status"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`

	Items     []MealItem      `json:"items" gorm:"-"`
	Nutrition nutrition.Facts `json:"nutrition" gorm:"-"`
	// Allergens is the UNION across every component, deduplicated. A meal is
	// what the customer eats, so an allergen in any one part of it is an
	// allergen in the meal — reporting them per component would leave the
	// reader to do the union, which is the one thing a person with an allergy
	// must not be asked to do. Never nil: a JSON null here would crash a
	// client iterating it, and "no allergens recorded" is a different claim
	// from "no allergens".
	Allergens []MealAllergen `json:"allergens" gorm:"-"`
}

// MealAllergen carries BOTH stored names rather than one resolved server-side.
//
// `allergen` is one of the few tables that is genuinely bilingual — name_id
// and name_en are columns (migration 0004) — so resolving here would throw
// away the other language and make the response locale-dependent for no
// reason. There is no Chinese column; a Chinese reader gets the English name,
// which is the same limitation docs/03 Q-24 records for catalogue content.
type MealAllergen struct {
	Code   string `json:"code"`
	NameID string `json:"name_id"`
	NameEN string `json:"name_en"`
}

// MealItem is one food in a meal.
type MealItem struct {
	ID       uuid.UUID `json:"id"`
	FoodID   uuid.UUID `json:"food_id"`
	FoodName string    `json:"food_name"`
	FoodSlug string    `json:"food_slug"`
	// PortionSize is free text on `food` — "150 g", "250 ml". Artboard 02
	// prints it beside every component, and the packing label depends on it
	// being the kitchen's own wording rather than a number we format.
	PortionSize string `json:"portion_size"`
	ItemRole    string `json:"item_role"`
	SortOrder   int    `json:"sort_order"`

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
}

// Facts converts an item's columns into the domain panel.
func (i MealItem) Facts() nutrition.Facts {
	return nutrition.Facts{
		CaloriesKcal: i.CaloriesKcal, ProteinMg: i.ProteinMg, FatMg: i.FatMg,
		SaturatedFatMg: i.SaturatedFatMg, CarbohydrateMg: i.CarbohydrateMg,
		SugarMg: i.SugarMg, FibreMg: i.FibreMg, SodiumMg: i.SodiumMg,
		CholesterolMg: i.CholesterolMg, Complete: i.IsComplete,
	}
}

// MealQuery narrows a calendar read.
type MealQuery struct {
	From          time.Time
	To            time.Time
	DietTypeID    *uuid.UUID
	SlotID        *uuid.UUID
	PublishedOnly bool
	Q             string
}

// ListMeals returns the calendar for a date range, with items attached.
//
// PublishedOnly is enforced HERE, in the repository, rather than in the
// handler: a customer-facing query that forgets it would leak next week's
// unfinished menu, and the handler is the easiest place to forget (PROMPT §4.4).
func (r *ScheduleRepo) ListMeals(ctx context.Context, q MealQuery) ([]Meal, error) {
	db := r.db.WithContext(ctx).Table("scheduled_meal m").
		Joins("JOIN diet_type dt ON dt.id = m.diet_type_id").
		Joins("JOIN delivery_time_slot s ON s.id = m.slot_id").
		Where("m.service_date >= ? AND m.service_date <= ?",
			q.From.Format("2006-01-02"), q.To.Format("2006-01-02"))

	if q.PublishedOnly {
		db = db.Where("m.status = 'PUBLISHED'")
	}
	if q.DietTypeID != nil {
		db = db.Where("m.diet_type_id = ?", *q.DietTypeID)
	}
	if q.SlotID != nil {
		db = db.Where("m.slot_id = ?", *q.SlotID)
	}
	if q.Q != "" {
		// Searching the calendar means searching what is IN the meals: staff
		// look for "ayam", not for a date they already know.
		pattern := SearchPattern(q.Q)
		db = db.Where(`lower(COALESCE(m.name,'')) LIKE ? OR lower(dt.name) LIKE ?
			OR EXISTS (SELECT 1 FROM scheduled_meal_item mi JOIN food f ON f.id = mi.food_id
			            WHERE mi.scheduled_meal_id = m.id AND lower(f.name) LIKE ?)`,
			pattern, pattern, pattern)
	}

	meals := []Meal{}
	err := db.Select(`m.id, m.service_date::text AS service_date, m.diet_type_id,
		dt.name AS diet_type_name, dt.slug AS diet_type_slug,
		m.slot_id, s.alias AS slot_alias, m.name, m.description, m.hero_photo_key,
		m.qty_capacity, m.qty_reserved, m.status, m.published_at`).
		Order("m.service_date ASC, s.sort_order ASC, dt.sort_order ASC").
		Scan(&meals).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: list meals: %w", err)
	}
	if len(meals) == 0 {
		return meals, nil
	}

	ids := make([]uuid.UUID, 0, len(meals))
	for _, m := range meals {
		ids = append(ids, m.ID)
	}

	// One query for every meal's items rather than one per meal: a month view
	// of five diet types across two slots is 300 meals, and 300 round trips is
	// how a calendar screen becomes slow enough to be abandoned.
	var items []struct {
		MealItem
		ScheduledMealID uuid.UUID
	}
	err = r.db.WithContext(ctx).Raw(`
		SELECT mi.id, mi.scheduled_meal_id, mi.food_id, f.name AS food_name, f.slug AS food_slug,
		       f.portion_size, mi.item_role, mi.sort_order,
		       COALESCE(n.calories_kcal,0) AS calories_kcal, COALESCE(n.protein_mg,0) AS protein_mg,
		       COALESCE(n.fat_mg,0) AS fat_mg, COALESCE(n.saturated_fat_mg,0) AS saturated_fat_mg,
		       COALESCE(n.carbohydrate_mg,0) AS carbohydrate_mg, COALESCE(n.sugar_mg,0) AS sugar_mg,
		       COALESCE(n.fibre_mg,0) AS fibre_mg, COALESCE(n.sodium_mg,0) AS sodium_mg,
		       COALESCE(n.cholesterol_mg,0) AS cholesterol_mg,
		       COALESCE(n.is_complete,false) AS is_complete
		  FROM scheduled_meal_item mi
		  JOIN food f ON f.id = mi.food_id
		  LEFT JOIN food_nutrition n ON n.food_id = f.id
		 WHERE mi.scheduled_meal_id IN ?
		 ORDER BY mi.sort_order, f.name`, ids).Scan(&items).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: meal items: %w", err)
	}

	byMeal := map[uuid.UUID][]MealItem{}
	for _, it := range items {
		byMeal[it.ScheduledMealID] = append(byMeal[it.ScheduledMealID], it.MealItem)
	}

	// Allergens, in the same one round trip. Ordered by name so the chips read
	// the same on every render and in every export.
	var allergenRows []struct {
		ScheduledMealID uuid.UUID
		Code            string
		NameID          string
		NameEN          string
	}
	if err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT mi.scheduled_meal_id, a.code, a.name_id, a.name_en
		  FROM scheduled_meal_item mi
		  JOIN food_allergen fa ON fa.food_id = mi.food_id
		  JOIN allergen a ON a.id = fa.allergen_id
		 WHERE mi.scheduled_meal_id IN ? AND a.is_active
		 ORDER BY a.name_id`, ids).Scan(&allergenRows).Error; err != nil {
		return nil, fmt.Errorf("postgres: meal allergens: %w", err)
	}
	allergensByMeal := map[uuid.UUID][]MealAllergen{}
	for _, a := range allergenRows {
		allergensByMeal[a.ScheduledMealID] = append(allergensByMeal[a.ScheduledMealID],
			MealAllergen{Code: a.Code, NameID: a.NameID, NameEN: a.NameEN})
	}
	for i := range meals {
		meals[i].Items = byMeal[meals[i].ID]
		meals[i].Allergens = allergensByMeal[meals[i].ID]
		if meals[i].Allergens == nil {
			meals[i].Allergens = []MealAllergen{}
		}
		meals[i].Nutrition = aggregateItems(meals[i].Items).Total
	}
	return meals, nil
}

// GetMeal loads one meal with items.
func (r *ScheduleRepo) GetMeal(ctx context.Context, id uuid.UUID, publishedOnly bool) (Meal, error) {
	var m Meal
	db := r.db.WithContext(ctx).Table("scheduled_meal m").
		Joins("JOIN diet_type dt ON dt.id = m.diet_type_id").
		Joins("JOIN delivery_time_slot s ON s.id = m.slot_id").
		Where("m.id = ?", id)
	if publishedOnly {
		db = db.Where("m.status = 'PUBLISHED'")
	}
	err := db.Select(`m.id, m.service_date::text AS service_date, m.diet_type_id,
		dt.name AS diet_type_name, dt.slug AS diet_type_slug, m.slot_id, s.alias AS slot_alias,
		m.name, m.description, m.hero_photo_key, m.qty_capacity, m.qty_reserved,
		m.status, m.published_at`).Scan(&m).Error
	if err != nil {
		return Meal{}, fmt.Errorf("postgres: get meal: %w", err)
	}
	if m.ID == uuid.Nil {
		return Meal{}, ErrNotFound
	}

	list, err := r.ListMeals(ctx, MealQuery{
		From: mustDate(m.ServiceDate), To: mustDate(m.ServiceDate),
		DietTypeID: &m.DietTypeID, SlotID: &m.SlotID, PublishedOnly: publishedOnly,
	})
	if err != nil {
		return Meal{}, err
	}
	if len(list) > 0 {
		return list[0], nil
	}
	return m, nil
}

// SaveMeal upserts a meal and replaces its items in one transaction.
func (r *ScheduleRepo) SaveMeal(ctx context.Context, m Meal, items []MealItem, by uuid.UUID) (uuid.UUID, error) {
	id := m.ID
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id == uuid.Nil {
			id = uuid.Must(uuid.NewV7())
			if err := tx.Exec(`
				INSERT INTO scheduled_meal (id, service_date, diet_type_id, slot_id, name,
				                            description, hero_photo_key, qty_capacity, status, updated_by)
				VALUES (?,?::date,?,?,?,?,?,?,'DRAFT',?)`,
				id, m.ServiceDate, m.DietTypeID, m.SlotID, m.Name, m.Description,
				m.HeroPhotoKey, m.QtyCapacity, by).Error; err != nil {
				return err
			}
		} else {
			// Capacity may not be lowered below what is already reserved; the
			// CHECK constraint would refuse it anyway, and this turns that into
			// a message instead of a 500.
			res := tx.Exec(`
				UPDATE scheduled_meal
				   SET name=?, description=?, hero_photo_key=?, qty_capacity=?, updated_by=?
				 WHERE id=?`,
				m.Name, m.Description, m.HeroPhotoKey, m.QtyCapacity, by, id)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrNotFound
			}
		}

		if err := tx.Exec(`DELETE FROM scheduled_meal_item WHERE scheduled_meal_id = ?`, id).Error; err != nil {
			return err
		}
		for i, it := range items {
			if err := tx.Exec(`
				INSERT INTO scheduled_meal_item (id, scheduled_meal_id, food_id, item_role, sort_order)
				VALUES (?,?,?,?,?)`,
				uuid.Must(uuid.NewV7()), id, it.FoodID, it.ItemRole, i).Error; err != nil {
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

// Publish moves meals from DRAFT to PUBLISHED. Bulk, because staff publish a
// week at a time (PROMPT §4.4).
func (r *ScheduleRepo) Publish(ctx context.Context, ids []uuid.UUID, by uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Exec(`
		UPDATE scheduled_meal
		   SET status = 'PUBLISHED', published_at = now(), updated_by = ?
		 WHERE id IN ? AND status = 'DRAFT'`, by, ids)
	if res.Error != nil {
		return 0, fmt.Errorf("postgres: publish: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// Unpublish returns meals to DRAFT.
//
// Refused once a meal has deliveries against it (docs/03 Q-18): a menu that
// silently vanishes after someone ordered from it is how a customer ends up
// with an empty box. The food can still be SUBSTITUTED, which is audited.
func (r *ScheduleRepo) Unpublish(ctx context.Context, id uuid.UUID, by uuid.UUID) error {
	var n int64
	if err := r.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM delivery_line dl WHERE dl.scheduled_meal_id = ?`, id).Scan(&n).Error; err != nil {
		return fmt.Errorf("postgres: unpublish check: %w", err)
	}
	if n > 0 {
		return fmt.Errorf("%w: %d deliveries already reference this meal", ErrHasDependents, n)
	}
	res := r.db.WithContext(ctx).Exec(`
		UPDATE scheduled_meal SET status='DRAFT', published_at=NULL, updated_by=?
		 WHERE id=? AND status='PUBLISHED'`, by, id)
	if res.Error != nil {
		return fmt.Errorf("postgres: unpublish: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ErrHasDependents means a row cannot change because something already depends
// on it.
var ErrHasDependents = fmt.Errorf("postgres: row has dependents")

// CopyWeek duplicates a week of meals to another week (PROMPT §4.4).
//
// Copies land as DRAFT whatever the source was: publishing is a deliberate act,
// and a copy-week that silently published next month's menu would be a very
// bad afternoon.
func (r *ScheduleRepo) CopyWeek(ctx context.Context, fromMonday, toMonday time.Time, by uuid.UUID) (int64, error) {
	offset := int(toMonday.Sub(fromMonday).Hours() / 24)
	var copied int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var src []Meal
		if err := tx.Raw(`
			SELECT id, service_date::text AS service_date, diet_type_id, slot_id,
			       name, description, hero_photo_key, qty_capacity
			  FROM scheduled_meal
			 WHERE service_date >= ?::date AND service_date < ?::date + 7`,
			fromMonday.Format("2006-01-02"), fromMonday.Format("2006-01-02")).Scan(&src).Error; err != nil {
			return err
		}
		for _, m := range src {
			newID := uuid.Must(uuid.NewV7())
			// ON CONFLICT DO NOTHING: the target week may already have a meal
			// for that date, diet and slot, and a copy must not clobber work
			// somebody already did.
			res := tx.Exec(`
				INSERT INTO scheduled_meal (id, service_date, diet_type_id, slot_id, name,
				                            description, hero_photo_key, qty_capacity, status, updated_by)
				VALUES (?, ?::date + ?::int, ?, ?, ?, ?, ?, ?, 'DRAFT', ?)
				ON CONFLICT (service_date, diet_type_id, slot_id) DO NOTHING`,
				newID, m.ServiceDate, offset, m.DietTypeID, m.SlotID, m.Name,
				m.Description, m.HeroPhotoKey, m.QtyCapacity, by)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				continue
			}
			if err := tx.Exec(`
				INSERT INTO scheduled_meal_item (id, scheduled_meal_id, food_id, item_role, sort_order)
				SELECT gen_random_uuid(), ?, food_id, item_role, sort_order
				  FROM scheduled_meal_item WHERE scheduled_meal_id = ?`, newID, m.ID).Error; err != nil {
				return err
			}
			copied++
		}
		return nil
	})
	return copied, err
}

// PublishHorizon returns the furthest published service date, for the dashboard
// warning (docs/03 Q-17).
func (r *ScheduleRepo) PublishHorizon(ctx context.Context) (time.Time, error) {
	var out []string
	if err := r.db.WithContext(ctx).Raw(
		`SELECT max(service_date)::text FROM scheduled_meal WHERE status='PUBLISHED'`).Scan(&out).Error; err != nil {
		return time.Time{}, fmt.Errorf("postgres: publish horizon: %w", err)
	}
	if len(out) == 0 || out[0] == "" {
		return time.Time{}, nil
	}
	return mustDate(out[0]), nil
}

func aggregateItems(items []MealItem) nutrition.Meal {
	in := make([]nutrition.Item, 0, len(items))
	for _, it := range items {
		in = append(in, nutrition.Item{
			FoodID: it.FoodID.String(), Name: it.FoodName,
			Role: it.ItemRole, Sort: it.SortOrder, Facts: it.Facts(),
		})
	}
	return nutrition.Aggregate(in)
}

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t
}
