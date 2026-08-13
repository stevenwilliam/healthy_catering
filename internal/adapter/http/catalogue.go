package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// registerCatalogue mounts the food and menu-calendar endpoints, staff and
// public.
func registerCatalogue(g *gin.RouterGroup, d Deps) {
	// ── Public menu ─────────────────────────────────────────────────────────
	// No authentication: this is the SEO surface (PROMPT §13.3). It serves
	// PUBLISHED meals only, enforced in the repository.
	g.GET("/menu", func(c *gin.Context) {
		q := app.CalendarQuery{
			From: c.Query("from"), To: c.Query("to"), Q: c.Query("q"), PublicOnly: true,
		}
		if q.From == "" {
			// Default to the week ahead, which is what a customer landing on
			// the menu page wants to see.
			loc := tzOf(d)
			now := time.Now().In(loc)
			q.From = now.Format("2006-01-02")
			q.To = now.AddDate(0, 0, 7).Format("2006-01-02")
		}
		if q.To == "" {
			q.To = q.From
		}
		if slug := c.Query("diet"); slug != "" {
			dt, err := d.Catalogue.DietTypeBySlug(c.Request.Context(), slug)
			if err != nil {
				Fail(c, err)
				return
			}
			q.DietTypeID = &dt.ID
		}
		meals, err := d.Catalogue.Calendar(c.Request.Context(), q)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, meals)
	})

	g.GET("/diet-types", func(c *gin.Context) {
		p := listParams(c)
		active := true
		p.Active = &active
		page, err := d.Admin.ListDietTypes(c.Request.Context(), p)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	// ── Staff: foods ────────────────────────────────────────────────────────
	admin := g.Group("/admin", RequireAuth(d.Auth, d.Signer))

	admin.GET("/foods", RequirePermission(security.PermCatalogueRead), func(c *gin.Context) {
		var diet *uuid.UUID
		if v := c.Query("diet_type_id"); v != "" {
			id, err := uuid.Parse(v)
			if err != nil {
				Fail(c, apierror.Validation("diet_type_id must be a UUID.", nil))
				return
			}
			diet = &id
		}
		page, err := d.Catalogue.ListFoods(c.Request.Context(), listParams(c), diet)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.GET("/foods/:id", RequirePermission(security.PermCatalogueRead), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		f, err := d.Catalogue.GetFood(c.Request.Context(), id)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, f)
	})

	admin.POST("/foods", RequirePermission(security.PermCatalogueWrite), func(c *gin.Context) {
		var body foodBody
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send name, slug and at least one diet type."))
			return
		}
		id, err := d.Catalogue.SaveFood(c.Request.Context(), body.input(uuid.Nil), actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusCreated, gin.H{"id": id})
	})

	admin.PUT("/foods/:id", RequirePermission(security.PermCatalogueWrite), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		var body foodBody
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send the food fields."))
			return
		}
		if _, err := d.Catalogue.SaveFood(c.Request.Context(), body.input(id), actorFrom(c)); err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"id": id})
	})

	// ── Staff: the menu calendar ────────────────────────────────────────────
	admin.GET("/calendar", RequirePermission(security.PermScheduleRead), func(c *gin.Context) {
		q := app.CalendarQuery{From: c.Query("from"), To: c.Query("to"), Q: c.Query("q")}
		if q.From == "" {
			loc := tzOf(d)
			now := time.Now().In(loc)
			q.From = now.Format("2006-01-02")
			q.To = now.AddDate(0, 0, 30).Format("2006-01-02")
		}
		if q.To == "" {
			q.To = q.From
		}
		if v := c.Query("diet_type_id"); v != "" {
			id, err := uuid.Parse(v)
			if err != nil {
				Fail(c, apierror.Validation("diet_type_id must be a UUID.", nil))
				return
			}
			q.DietTypeID = &id
		}
		meals, err := d.Catalogue.Calendar(c.Request.Context(), q)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, meals)
	})

	admin.POST("/calendar/meals", RequirePermission(security.PermScheduleWrite), func(c *gin.Context) {
		var body mealBody
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send service_date, diet_type_id, slot_id and items."))
			return
		}
		id, err := d.Catalogue.SaveMeal(c.Request.Context(), body.input(uuid.Nil), actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusCreated, gin.H{"id": id})
	})

	admin.PUT("/calendar/meals/:id", RequirePermission(security.PermScheduleWrite), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		var body mealBody
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send the meal fields."))
			return
		}
		if _, err := d.Catalogue.SaveMeal(c.Request.Context(), body.input(id), actorFrom(c)); err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"id": id})
	})

	admin.POST("/calendar/publish", RequirePermission(security.PermScheduleWrite), func(c *gin.Context) {
		var body struct {
			IDs []uuid.UUID `json:"ids"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send ids."))
			return
		}
		n, err := d.Catalogue.PublishMeals(c.Request.Context(), body.IDs, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"published": n})
	})

	admin.POST("/calendar/meals/:id/unpublish", RequirePermission(security.PermScheduleWrite), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		if err := d.Catalogue.UnpublishMeal(c.Request.Context(), id, actorFrom(c)); err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"id": id, "status": "DRAFT"})
	})

	admin.POST("/calendar/copy-week", RequirePermission(security.PermScheduleWrite), func(c *gin.Context) {
		var body struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send from and to as the Mondays of each week."))
			return
		}
		n, err := d.Catalogue.CopyWeek(c.Request.Context(), body.From, body.To, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"copied": n})
	})

	admin.GET("/calendar/horizon", RequirePermission(security.PermScheduleRead), func(c *gin.Context) {
		h, err := d.Catalogue.Horizon(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, h)
	})
}

type foodBody struct {
	Name           string      `json:"name"`
	Slug           string      `json:"slug"`
	Description    string      `json:"description"`
	PortionSize    string      `json:"portion_size"`
	IsActive       *bool       `json:"is_active"`
	CaloriesKcal   int         `json:"calories_kcal"`
	ProteinMg      int         `json:"protein_mg"`
	FatMg          int         `json:"fat_mg"`
	SaturatedFatMg int         `json:"saturated_fat_mg"`
	CarbohydrateMg int         `json:"carbohydrate_mg"`
	SugarMg        int         `json:"sugar_mg"`
	FibreMg        int         `json:"fibre_mg"`
	SodiumMg       int         `json:"sodium_mg"`
	CholesterolMg  int         `json:"cholesterol_mg"`
	DietTypeIDs    []uuid.UUID `json:"diet_type_ids"`
	AllergenIDs    []uuid.UUID `json:"allergen_ids"`
}

func (b foodBody) input(id uuid.UUID) app.FoodInput {
	return app.FoodInput{
		ID: id, Name: b.Name, Slug: b.Slug, Description: b.Description,
		PortionSize: b.PortionSize, IsActive: boolOr(b.IsActive, true),
		CaloriesKcal: b.CaloriesKcal, ProteinMg: b.ProteinMg, FatMg: b.FatMg,
		SaturatedFatMg: b.SaturatedFatMg, CarbohydrateMg: b.CarbohydrateMg,
		SugarMg: b.SugarMg, FibreMg: b.FibreMg, SodiumMg: b.SodiumMg,
		CholesterolMg: b.CholesterolMg,
		DietTypeIDs:   b.DietTypeIDs, AllergenIDs: b.AllergenIDs,
	}
}

type mealBody struct {
	ServiceDate string    `json:"service_date"`
	DietTypeID  uuid.UUID `json:"diet_type_id"`
	SlotID      uuid.UUID `json:"slot_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	QtyCapacity *int      `json:"qty_capacity"`
	Items       []struct {
		FoodID   uuid.UUID `json:"food_id"`
		ItemRole string    `json:"item_role"`
	} `json:"items"`
}

func (b mealBody) input(id uuid.UUID) app.MealInput {
	items := make([]app.MealItemInput, 0, len(b.Items))
	for _, it := range b.Items {
		items = append(items, app.MealItemInput{FoodID: it.FoodID, ItemRole: it.ItemRole})
	}
	return app.MealInput{
		ID: id, ServiceDate: b.ServiceDate, DietTypeID: b.DietTypeID, SlotID: b.SlotID,
		Name: b.Name, Description: b.Description, QtyCapacity: b.QtyCapacity, Items: items,
	}
}

func tzOf(d Deps) *time.Location {
	loc, err := d.Config.Locale.TZ()
	if err != nil || loc == nil {
		return time.UTC
	}
	return loc
}
