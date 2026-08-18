package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/i18n"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// listParams reads the query string every list endpoint accepts.
//
// `q` is here rather than per-handler because the house rule has no exceptions:
// every list screen has a search box that filters it (CLAUDE.md §7).
func listParams(c *gin.Context) postgres.ListParams {
	p := postgres.ListParams{
		Q:    c.Query("q"),
		Sort: c.Query("sort"),
		Desc: c.Query("dir") == "desc",
	}
	p.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	p.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "25"))
	if v := c.Query("active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			p.Active = &b
		}
	}
	return p
}

// listParamsWithActive is the server-side equivalent for pages that have no
// request query, such as the rendered public pages.
func listParamsWithActive(active *bool) postgres.ListParams {
	return postgres.ListParams{Page: 1, PageSize: 50, Active: active}
}

// actorFrom builds the audit actor from the authenticated request.
func actorFrom(c *gin.Context) app.Actor {
	ident, _ := Authenticated(c)
	return app.Actor{
		UserID: ident.UserID, Email: ident.Email,
		IP: c.ClientIP(), UA: c.Request.UserAgent(),
	}
}

// pathUUID parses a UUID path parameter, rejecting anything else.
func pathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		Fail(c, apierror.Validation("That identifier is not valid.", map[string]any{name: "must be a UUID"}))
		return uuid.Nil, false
	}
	return id, true
}

// registerAdmin mounts the back office.
//
// Every route declares its permission inline. Reading this function is reading
// the authorization model — which is the point of deny-by-default: there is no
// second place where access is granted (99 §7).
func registerAdmin(g *gin.RouterGroup, d Deps) {
	admin := g.Group("/admin", RequireAuth(d.Auth, d.Signer))

	// ── Customer types ──────────────────────────────────────────────────────
	admin.GET("/customer-types", RequirePermission(security.PermCustomerRead), func(c *gin.Context) {
		page, err := d.Admin.ListCustomerTypes(c.Request.Context(), listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.POST("/customer-types", RequirePermission(security.PermCustomerTypeChange), func(c *gin.Context) {
		var body struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			Description string `json:"description"`
			IsCorporate bool   `json:"is_corporate"`
			IsActive    *bool  `json:"is_active"`
			SortOrder   int    `json:"sort_order"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send name and slug."))
			return
		}
		id, err := d.Admin.CreateCustomerType(c.Request.Context(), app.CustomerTypeInput{
			Name: body.Name, Slug: body.Slug, Description: body.Description,
			IsCorporate: body.IsCorporate, IsActive: boolOr(body.IsActive, true),
			SortOrder: body.SortOrder,
		}, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusCreated, gin.H{"id": id})
	})

	admin.PATCH("/customer-types/:id", RequirePermission(security.PermCustomerTypeChange), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		var body struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			Description string `json:"description"`
			IsCorporate bool   `json:"is_corporate"`
			IsActive    *bool  `json:"is_active"`
			SortOrder   int    `json:"sort_order"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send the fields to update."))
			return
		}
		err := d.Admin.UpdateCustomerType(c.Request.Context(), app.CustomerTypeInput{
			ID: id, Name: body.Name, Slug: body.Slug, Description: body.Description,
			IsCorporate: body.IsCorporate, IsActive: boolOr(body.IsActive, true),
			SortOrder: body.SortOrder,
		}, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"id": id})
	})

	// ── Diet types ──────────────────────────────────────────────────────────
	admin.GET("/diet-types", RequirePermission(security.PermCatalogueRead), func(c *gin.Context) {
		page, err := d.Admin.ListDietTypes(c.Request.Context(), listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.POST("/diet-types", RequirePermission(security.PermCatalogueWrite), func(c *gin.Context) {
		var body dietTypeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send name and slug."))
			return
		}
		id, err := d.Admin.CreateDietType(c.Request.Context(), body.input(uuid.Nil), actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusCreated, gin.H{"id": id})
	})

	admin.PATCH("/diet-types/:id", RequirePermission(security.PermCatalogueWrite), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		var body dietTypeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send the fields to update."))
			return
		}
		if err := d.Admin.UpdateDietType(c.Request.Context(), body.input(id), actorFrom(c)); err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"id": id})
	})

	// ── Allergens ───────────────────────────────────────────────────────────
	admin.GET("/allergens", RequirePermission(security.PermCatalogueRead), func(c *gin.Context) {
		page, err := d.Admin.ListAllergens(c.Request.Context(), listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	// ── Delivery slots ──────────────────────────────────────────────────────
	admin.GET("/delivery-slots", RequirePermission(security.PermScheduleRead), func(c *gin.Context) {
		page, err := d.Admin.ListSlots(c.Request.Context(), listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.PATCH("/delivery-slots/:id", RequirePermission(security.PermSettingsWrite), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		var body struct {
			SlotTime       string `json:"slot_time"`
			Alias          string `json:"alias"`
			SortOrder      int    `json:"sort_order"`
			IsActive       bool   `json:"is_active"`
			CutoffTime     string `json:"cutoff_time"`
			CutoffLeadDays *int   `json:"cutoff_lead_days"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send slot_time and alias."))
			return
		}
		err := d.Admin.UpdateSlot(c.Request.Context(), app.SlotInput{
			ID: id, SlotTime: body.SlotTime, Alias: body.Alias,
			SortOrder: body.SortOrder, IsActive: body.IsActive,
			CutoffTime: body.CutoffTime, CutoffLeadDays: body.CutoffLeadDays,
		}, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"id": id})
	})

	// ── Organisations ───────────────────────────────────────────────────────
	admin.GET("/organisations", RequirePermission(security.PermOrganisationManage), func(c *gin.Context) {
		page, err := d.Admin.ListOrganisations(c.Request.Context(), listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.POST("/organisations", RequirePermission(security.PermOrganisationManage), func(c *gin.Context) {
		var body struct {
			Name             string `json:"name"`
			Slug             string `json:"slug"`
			PICName          string `json:"pic_name"`
			PICPhone         string `json:"pic_phone"`
			BillingEmail     string `json:"billing_email"`
			BillingAddress   string `json:"billing_address"`
			NPWP             string `json:"npwp"`
			PONumber         string `json:"po_number"`
			IsInvoiceBilling bool   `json:"is_invoice_billing"`
			InvoiceDay       *int   `json:"invoice_day"`
			Notes            string `json:"notes"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send name and slug."))
			return
		}
		id, err := d.Admin.CreateOrganisation(c.Request.Context(), app.OrganisationInput{
			Name: body.Name, Slug: body.Slug, PICName: body.PICName, PICPhone: body.PICPhone,
			BillingEmail: body.BillingEmail, BillingAddress: body.BillingAddress,
			NPWP: body.NPWP, PONumber: body.PONumber,
			IsInvoiceBilling: body.IsInvoiceBilling, InvoiceDay: body.InvoiceDay, Notes: body.Notes,
		}, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusCreated, gin.H{"id": id})
	})

	// ── Career: openings and applications ───────────────────────────────────
	//
	// Both are data grids, so both export (99 §8). The export honours the
	// current search and status filter — an export of something other than
	// what is on screen is worse than none.
	admin.GET("/job-applications", RequirePermission(security.PermSettingsRead), func(c *gin.Context) {
		page, err := d.Career.ListApplications(c.Request.Context(), listParams(c), c.Query("status"))
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			rows := make([][]string, 0, len(page.Items))
			for _, a := range page.Items {
				rows = append(rows, []string{
					a.CreatedAt.Format("2006-01-02 15:04"), a.FullName, a.Email,
					a.Phone, a.Position, a.Locale, a.Status, a.Message,
				})
			}
			csvOut(c, "job-applications",
				[]string{"received", "name", "email", "phone", "position",
					"language", "status", "message"}, rows)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.PATCH("/job-applications/:id", RequirePermission(security.PermSettingsWrite), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			Fail(c, apierror.BadRequest(apierror.CodeValidation, "Not a valid id."))
			return
		}
		var in struct {
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			Fail(c, apierror.BadRequest(apierror.CodeValidation, "Send a JSON body with a status."))
			return
		}
		if err := d.Career.SetStatus(c.Request.Context(), id, in.Status, actorFrom(c)); err != nil {
			Fail(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	admin.GET("/job-openings", RequirePermission(security.PermSettingsRead), func(c *gin.Context) {
		rows, err := d.Career.AllOpenings(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			out := make([][]string, 0, len(rows))
			for _, o := range rows {
				active := "no"
				if o.IsActive {
					active = "yes"
				}
				out = append(out, []string{o.Title, o.Slug, o.Summary,
					strconv.Itoa(o.SortOrder), active})
			}
			csvOut(c, "job-openings",
				[]string{"title", "slug", "summary", "sort order", "active"}, out)
			return
		}
		OK(c, http.StatusOK, rows)
	})

	// ── Public content ──────────────────────────────────────────────────────
	//
	// The hero wording, in three languages. Indonesian is the source and the
	// other two are machine-translated from it unless a human has overridden
	// them (docs/11 §6). Same permission as settings: this is site
	// configuration, edited by the same people.
	admin.GET("/content", RequirePermission(security.PermSettingsRead), func(c *gin.Context) {
		entries, err := d.Content.List(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"entries":    entries,
			"translator": d.Content.TranslatorStatus(),
			"locales":    i18n.All(),
		})
	})

	// Replace the Indonesian text. The response says what happened to each
	// translation rather than just "ok" — "translated", "kept-override" and
	// "no-translator" are three different things for the editor to see.
	admin.PUT("/content/:key", RequirePermission(security.PermSettingsWrite), func(c *gin.Context) {
		var in struct {
			Value string `json:"value"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			Fail(c, apierror.BadRequest(apierror.CodeValidation, "Send a JSON body with a value."))
			return
		}
		out, err := d.Content.SetSource(c.Request.Context(), c.Param("key"), in.Value, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"translations": out})
	})

	// Pin a translation by hand. It is never overwritten afterwards.
	admin.PUT("/content/:key/:locale", RequirePermission(security.PermSettingsWrite), func(c *gin.Context) {
		var in struct {
			Value string `json:"value"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			Fail(c, apierror.BadRequest(apierror.CodeValidation, "Send a JSON body with a value."))
			return
		}
		if err := d.Content.SetOverride(c.Request.Context(),
			c.Param("key"), c.Param("locale"), in.Value, actorFrom(c)); err != nil {
			Fail(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// Release a translation back to the machine and re-run it immediately, so
	// the editor sees the machine's version rather than an empty box.
	admin.DELETE("/content/:key/:locale", RequirePermission(security.PermSettingsWrite), func(c *gin.Context) {
		outcome, err := d.Content.ClearOverride(c.Request.Context(),
			c.Param("key"), c.Param("locale"), actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"outcome": outcome})
	})

	// Re-translate everything not overridden — the button for the first time a
	// provider is configured.
	admin.POST("/content/retranslate", RequirePermission(security.PermSettingsWrite), func(c *gin.Context) {
		out, err := d.Content.RetranslateAll(c.Request.Context(), actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"results": out})
	})

	// ── Settings ────────────────────────────────────────────────────────────
	admin.GET("/settings", RequirePermission(security.PermSettingsRead), func(c *gin.Context) {
		page, err := d.Admin.ListSettings(c.Request.Context(), listParams(c), c.Query("group"))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.GET("/settings/groups", RequirePermission(security.PermSettingsRead), func(c *gin.Context) {
		groups, err := d.Admin.SettingGroups(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, groups)
	})

	admin.PUT("/settings/:key", RequirePermission(security.PermSettingsWrite), func(c *gin.Context) {
		var body struct {
			Value  string `json:"value"`
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send value."))
			return
		}
		err := d.Admin.UpdateSetting(c.Request.Context(), c.Param("key"), body.Value,
			body.Reason, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"key": c.Param("key")})
	})
}

type dietTypeBody struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
	HasSubtypes    bool   `json:"has_subtypes"`
	SortOrder      int    `json:"sort_order"`
	IsActive       *bool  `json:"is_active"`
}

func (b dietTypeBody) input(id uuid.UUID) app.DietTypeInput {
	return app.DietTypeInput{
		ID: id, Name: b.Name, Slug: b.Slug, Description: b.Description,
		SEOTitle: b.SEOTitle, SEODescription: b.SEODescription,
		HasSubtypes: b.HasSubtypes, SortOrder: b.SortOrder,
		IsActive: boolOr(b.IsActive, true),
	}
}

// boolOr defaults an omitted JSON boolean.
//
// A missing `is_active` means "the caller did not say", and for these tables
// the safe reading is active — a newly created diet type nobody can see is a
// confusing default.
func boolOr(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}
