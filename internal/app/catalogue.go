package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/domain/schedule"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Catalogue is foods, nutrition and the menu calendar.
type Catalogue struct {
	foods  *postgres.CatalogueRepo
	sched  *postgres.ScheduleRepo
	master *postgres.MasterDataRepo
	audit  *postgres.AuditRepo
	params *sysparam.Store
	tz     *time.Location
}

func NewCatalogue(f *postgres.CatalogueRepo, s *postgres.ScheduleRepo,
	m *postgres.MasterDataRepo, a *postgres.AuditRepo, p *sysparam.Store, tz *time.Location) *Catalogue {
	return &Catalogue{foods: f, sched: s, master: m, audit: a, params: p, tz: tz}
}

// ── Foods ───────────────────────────────────────────────────────────────────

func (c *Catalogue) ListFoods(ctx context.Context, p postgres.ListParams, diet *uuid.UUID) (postgres.Page[postgres.Food], error) {
	return c.foods.ListFoods(ctx, p, diet)
}

func (c *Catalogue) GetFood(ctx context.Context, id uuid.UUID) (postgres.Food, error) {
	f, err := c.foods.GetFood(ctx, id)
	if err != nil {
		return postgres.Food{}, notFoundOr(err, "That dish no longer exists.")
	}
	return f, nil
}

// FoodInput is a create or edit.
type FoodInput struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description string
	PortionSize string
	IsActive    bool

	CaloriesKcal   int
	ProteinMg      int
	FatMg          int
	SaturatedFatMg int
	CarbohydrateMg int
	SugarMg        int
	FibreMg        int
	SodiumMg       int
	CholesterolMg  int

	DietTypeIDs []uuid.UUID
	AllergenIDs []uuid.UUID
}

// SaveFood validates and stores a dish with its nutrition panel.
func (c *Catalogue) SaveFood(ctx context.Context, in FoodInput, by Actor) (uuid.UUID, error) {
	name, err := sanitize.Required("name", in.Name, 160)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	slug, err := sanitize.Slug("slug", in.Slug, 160)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	desc, err := sanitize.Text("description", in.Description, 4000)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	portion, err := sanitize.Text("portion_size", in.PortionSize, 60)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	if len(in.DietTypeIDs) == 0 {
		return uuid.Nil, apierror.Validation(
			"A dish must belong to at least one diet type, or it can never be scheduled.",
			map[string]any{"diet_type_ids": "choose at least one"})
	}

	// Nutrition is integers in milligrams (D-24) and none of them can be
	// negative. A panel is COMPLETE only when calories are filled in — the
	// aggregate marks a whole meal incomplete from one blank panel, so this
	// flag decides whether a customer sees a number they can rely on.
	fields := map[string]int{
		"calories_kcal": in.CaloriesKcal, "protein_mg": in.ProteinMg, "fat_mg": in.FatMg,
		"saturated_fat_mg": in.SaturatedFatMg, "carbohydrate_mg": in.CarbohydrateMg,
		"sugar_mg": in.SugarMg, "fibre_mg": in.FibreMg, "sodium_mg": in.SodiumMg,
		"cholesterol_mg": in.CholesterolMg,
	}
	for field, v := range fields {
		if v < 0 {
			return uuid.Nil, apierror.Validation("Nutrition values cannot be negative.",
				map[string]any{field: "must be zero or more"})
		}
		// A single dish over 10,000 kcal or 10 kg of anything is a unit
		// mistake — grams typed where milligrams were meant.
		if v > 10_000_000 {
			return uuid.Nil, apierror.Validation(
				"That nutrition value looks like a unit mistake — these fields are in milligrams.",
				map[string]any{field: "milligrams, not grams"})
		}
	}

	f := postgres.Food{
		ID: in.ID, Name: name, Slug: slug, Description: desc, PortionSize: portion,
		IsActive:     in.IsActive,
		CaloriesKcal: in.CaloriesKcal, ProteinMg: in.ProteinMg, FatMg: in.FatMg,
		SaturatedFatMg: in.SaturatedFatMg, CarbohydrateMg: in.CarbohydrateMg,
		SugarMg: in.SugarMg, FibreMg: in.FibreMg, SodiumMg: in.SodiumMg,
		CholesterolMg: in.CholesterolMg,
		IsComplete:    in.CaloriesKcal > 0,
		DietTypeIDs:   in.DietTypeIDs, AllergenIDs: in.AllergenIDs,
	}

	id, err := c.foods.SaveFood(ctx, f, by.UserID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, apierror.NotFound("That dish no longer exists.")
		}
		return uuid.Nil, conflictOrInternal(err, "A dish with that slug already exists.")
	}

	action := "food.update"
	if in.ID == uuid.Nil {
		action = "food.create"
	}
	c.log(ctx, by, action, "food", &id, nil, f, "")
	return id, nil
}

// DietTypeBySlug is the public lookup for /menu?diet=weight-loss.
func (c *Catalogue) DietTypeBySlug(ctx context.Context, slug string) (postgres.DietType, error) {
	clean, err := sanitize.Slug("diet", slug, 80)
	if err != nil {
		return postgres.DietType{}, validationFrom(err)
	}
	dt, err := c.master.GetDietTypeBySlug(ctx, clean)
	if err != nil {
		return postgres.DietType{}, notFoundOr(err, "No such diet type.")
	}
	return dt, nil
}

// ── Menu calendar ───────────────────────────────────────────────────────────

// CalendarQuery is a staff or public calendar read.
type CalendarQuery struct {
	From       string
	To         string
	DietTypeID *uuid.UUID
	SlotID     *uuid.UUID
	Q          string
	PublicOnly bool
}

// Calendar returns the menu for a date range.
func (c *Catalogue) Calendar(ctx context.Context, q CalendarQuery) ([]postgres.Meal, error) {
	from, err := c.parseDate("from", q.From)
	if err != nil {
		return nil, err
	}
	to, err := c.parseDate("to", q.To)
	if err != nil {
		return nil, err
	}
	if to.Before(from) {
		return nil, apierror.Validation("The end date is before the start date.", nil)
	}
	// A range cap, because "give me every meal ever" is one query away from
	// being the slowest thing in the system.
	if to.Sub(from) > 120*24*time.Hour {
		return nil, apierror.Validation("Ask for at most 120 days at a time.",
			map[string]any{"to": "within 120 days of from"})
	}

	meals, err := c.sched.ListMeals(ctx, postgres.MealQuery{
		From: from, To: to, DietTypeID: q.DietTypeID, SlotID: q.SlotID,
		PublishedOnly: q.PublicOnly, Q: q.Q,
	})
	if err != nil {
		return nil, apierror.Internal(err)
	}
	if meals == nil {
		// A JSON `null` where the client expects an array is a frontend crash;
		// an empty calendar is a normal answer, not a missing one.
		meals = []postgres.Meal{}
	}
	return meals, nil
}

// MealInput is a create or edit of one sitting.
type MealInput struct {
	ID          uuid.UUID
	ServiceDate string
	DietTypeID  uuid.UUID
	SlotID      uuid.UUID
	Name        string
	Description string
	QtyCapacity *int
	Items       []MealItemInput
}

// MealItemInput is one food in the meal.
type MealItemInput struct {
	FoodID   uuid.UUID
	ItemRole string
}

// SaveMeal validates and stores a sitting.
func (c *Catalogue) SaveMeal(ctx context.Context, in MealInput, by Actor) (uuid.UUID, error) {
	date, err := c.parseDate("service_date", in.ServiceDate)
	if err != nil {
		return uuid.Nil, err
	}
	if in.DietTypeID == uuid.Nil || in.SlotID == uuid.Nil {
		return uuid.Nil, apierror.Validation("A meal needs a diet type and a delivery slot.", nil)
	}
	if len(in.Items) == 0 {
		return uuid.Nil, apierror.Validation(
			"A meal needs at least one dish — one credit buys this whole meal.",
			map[string]any{"items": "add at least one dish"})
	}

	name, err := sanitize.Text("name", in.Name, 160)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	desc, err := sanitize.Text("description", in.Description, 2000)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	if in.QtyCapacity != nil && *in.QtyCapacity < 1 {
		return uuid.Nil, apierror.Validation("Capacity must be at least 1, or left empty for unlimited.",
			map[string]any{"qty_capacity": "1 or more, or empty"})
	}

	seen := map[uuid.UUID]bool{}
	items := make([]postgres.MealItem, 0, len(in.Items))
	mains := 0
	for _, it := range in.Items {
		if it.FoodID == uuid.Nil {
			return uuid.Nil, apierror.Validation("Every item needs a dish.", nil)
		}
		if seen[it.FoodID] {
			return uuid.Nil, apierror.Validation("The same dish is listed twice in this meal.",
				map[string]any{"items": "each dish may appear once"})
		}
		seen[it.FoodID] = true

		role, err := sanitize.Enum("item_role", orDefault(it.ItemRole, "MAIN"),
			"MAIN", "SIDE", "DESSERT", "DRINK")
		if err != nil {
			return uuid.Nil, validationFrom(err)
		}
		if role == "MAIN" {
			mains++
		}
		items = append(items, postgres.MealItem{FoodID: it.FoodID, ItemRole: role})
	}
	if mains == 0 {
		return uuid.Nil, apierror.Validation("A meal needs a main dish.",
			map[string]any{"items": "mark one item as MAIN"})
	}

	m := postgres.Meal{
		ID: in.ID, ServiceDate: date.Format("2006-01-02"),
		DietTypeID: in.DietTypeID, SlotID: in.SlotID,
		Description: desc, QtyCapacity: in.QtyCapacity,
	}
	if name != "" {
		m.Name = &name
	}

	id, err := c.sched.SaveMeal(ctx, m, items, by.UserID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, apierror.NotFound("That meal no longer exists.")
		}
		return uuid.Nil, conflictOrInternal(err,
			"There is already a meal for that date, diet type and slot.")
	}

	action := "meal.update"
	if in.ID == uuid.Nil {
		action = "meal.create"
	}
	c.log(ctx, by, action, "scheduled_meal", &id, nil, m, "")
	return id, nil
}

// PublishMeals publishes a set of meals. Customers only ever see PUBLISHED.
func (c *Catalogue) PublishMeals(ctx context.Context, ids []uuid.UUID, by Actor) (int64, error) {
	if len(ids) == 0 {
		return 0, apierror.Validation("Select at least one meal to publish.", nil)
	}
	if len(ids) > 500 {
		return 0, apierror.Validation("Publish at most 500 meals at a time.", nil)
	}
	n, err := c.sched.Publish(ctx, ids, by.UserID)
	if err != nil {
		return 0, apierror.Internal(err)
	}
	c.log(ctx, by, "meal.publish", "scheduled_meal", nil, nil,
		map[string]any{"count": n, "ids": ids}, "")
	return n, nil
}

// UnpublishMeal returns one meal to draft, unless deliveries already exist.
func (c *Catalogue) UnpublishMeal(ctx context.Context, id uuid.UUID, by Actor) error {
	err := c.sched.Unpublish(ctx, id, by.UserID)
	switch {
	case errors.Is(err, postgres.ErrHasDependents):
		return apierror.Conflict(apierror.CodeConflict,
			"Customers have already ordered this meal. Substitute a dish instead of unpublishing it — "+
				"a menu that disappears after an order is how someone receives an empty box.")
	case errors.Is(err, postgres.ErrNotFound):
		return apierror.NotFound("That meal is not published.")
	case err != nil:
		return apierror.Internal(err)
	}
	c.log(ctx, by, "meal.unpublish", "scheduled_meal", &id, nil, nil, "")
	return nil
}

// CopyWeek duplicates a week of the calendar, landing as DRAFT.
func (c *Catalogue) CopyWeek(ctx context.Context, fromDate, toDate string, by Actor) (int64, error) {
	from, err := c.parseDate("from", fromDate)
	if err != nil {
		return 0, err
	}
	to, err := c.parseDate("to", toDate)
	if err != nil {
		return 0, err
	}
	if from.Equal(to) {
		return 0, apierror.Validation("The source and target weeks are the same.", nil)
	}
	n, err := c.sched.CopyWeek(ctx, from, to, by.UserID)
	if err != nil {
		return 0, apierror.Internal(err)
	}
	c.log(ctx, by, "meal.copy_week", "scheduled_meal", nil,
		map[string]any{"from": fromDate}, map[string]any{"to": toDate, "copied": n}, "")
	return n, nil
}

// Horizon reports how far ahead the menu is published, and whether that clears
// the operational target (docs/03 Q-17).
func (c *Catalogue) Horizon(ctx context.Context) (map[string]any, error) {
	last, err := c.sched.PublishHorizon(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	target := c.params.Int(ctx, sysparam.KeyPublishHorizonDays, 7)
	days, healthy := schedule.PublishHorizon(last, time.Now(), c.tz, target)
	out := map[string]any{"days_ahead": days, "target_days": target, "healthy": healthy}
	if !last.IsZero() {
		out["published_to"] = last.Format("2006-01-02")
	}
	if !healthy {
		out["warning"] = fmt.Sprintf(
			"The menu is published only %d days ahead; package customers cannot book beyond that.", days)
	}
	return out, nil
}

func (c *Catalogue) parseDate(field, v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, apierror.Validation("A date is required.",
			map[string]any{field: "YYYY-MM-DD"})
	}
	t, err := time.ParseInLocation("2006-01-02", v, c.tz)
	if err != nil {
		return time.Time{}, apierror.Validation("That date is not valid.",
			map[string]any{field: "YYYY-MM-DD"})
	}
	return t, nil
}

func (c *Catalogue) log(ctx context.Context, by Actor, action, entity string,
	id *uuid.UUID, before, after any, reason string) {
	_ = c.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &by.UserID, ActorEmail: by.Email, Action: action,
		EntityType: entity, EntityID: id, Before: before, After: after,
		Reason: reason, IP: by.IP, UserAgent: by.UA,
	})
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
