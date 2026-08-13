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

// registerPricing mounts the four price forms and the quote endpoint.
//
// The table is a PATH segment, not a body field: four separate admin forms is
// an explicit requirement (PROMPT §5.2), and four separate URLs is what makes
// them separate screens rather than one screen with a dropdown that can be set
// wrong.
func registerPricing(g *gin.RouterGroup, d Deps) {
	admin := g.Group("/admin", RequireAuth(d.Auth, d.Signer))

	admin.GET("/prices/:table", RequirePermission(security.PermPriceRead), func(c *gin.Context) {
		page, err := d.Pricing.ListPrices(c.Request.Context(), c.Param("table"), listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.GET("/price-tiers", RequirePermission(security.PermPriceRead), func(c *gin.Context) {
		tiers, err := d.Pricing.Tiers(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, tiers)
	})

	admin.POST("/prices/:table", RequirePermission(security.PermPriceWrite), func(c *gin.Context) {
		var body priceBody
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send price_idr and valid_from."))
			return
		}
		res, err := d.Pricing.SavePrice(c.Request.Context(),
			body.input(uuid.Nil, c.Param("table")), actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusCreated, res)
	})

	admin.PUT("/prices/:table/:id", RequirePermission(security.PermPriceWrite), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		var body priceBody
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send the price fields."))
			return
		}
		res, err := d.Pricing.SavePrice(c.Request.Context(),
			body.input(id, c.Param("table")), actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, res)
	})

	// ── Quote ───────────────────────────────────────────────────────────────
	// Authenticated: the price depends on the caller's customer type, so an
	// anonymous quote would have to guess a scope, and guessing a price is the
	// one thing §5.1 forbids.
	authed := g.Group("", RequireAuth(d.Auth, d.Signer))

	authed.GET("/quote", func(c *gin.Context) {
		ident, _ := Authenticated(c)
		q, err := d.Ordering.QuoteFor(c.Request.Context(), ident, quoteQuery(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, q)
	})
}

// quoteQuery reads a quote request off the query string.
func quoteQuery(c *gin.Context) app.QuoteQuery {
	q := app.QuoteQuery{
		Date: c.Query("date"),
		Qty:  parseQtyParam(c, "qty", 1),
	}
	if v := c.Query("diet_type_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q.DietTypeID = &id
		}
	}
	if v := c.Query("package_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q.PackageID = &id
		}
	}
	return q
}

type priceBody struct {
	CustomerTypeID *uuid.UUID `json:"customer_type_id"`
	DietTypeID     *uuid.UUID `json:"diet_type_id"`
	TierID         *uuid.UUID `json:"tier_id"`
	PackageID      *uuid.UUID `json:"package_id"`
	PriceIDR       int64      `json:"price_idr"`
	PromoLabel     string     `json:"promo_label"`
	Note           string     `json:"note"`
	ValidFrom      string     `json:"valid_from"`
	ValidTo        *string    `json:"valid_to"`
	IsActive       *bool      `json:"is_active"`
}

func (b priceBody) input(id uuid.UUID, table string) app.PriceInput {
	return app.PriceInput{
		ID: id, Table: table, CustomerTypeID: b.CustomerTypeID,
		DietTypeID: b.DietTypeID, TierID: b.TierID, PackageID: b.PackageID,
		PriceIDR: b.PriceIDR, PromoLabel: b.PromoLabel, Note: b.Note,
		ValidFrom: b.ValidFrom, ValidTo: b.ValidTo, IsActive: boolOr(b.IsActive, true),
	}
}

// mustDate parses a business date, defaulting to tomorrow — the day a customer
// ordering now is most likely to want.
func mustDateOrTomorrow(v string, loc *time.Location) (time.Time, error) {
	if v == "" {
		now := time.Now().In(loc)
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc), nil
	}
	t, err := time.ParseInLocation("2006-01-02", v, loc)
	if err != nil {
		return time.Time{}, apierror.Validation("That date is not valid.",
			map[string]any{"date": "YYYY-MM-DD"})
	}
	return t, nil
}
