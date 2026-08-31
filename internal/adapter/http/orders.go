package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// registerOrders mounts checkout and the customer's own order views.
func registerOrders(g *gin.RouterGroup, d Deps) {
	authed := g.Group("", RequireAuth(d.Auth, d.Signer))

	// Ordering requires a verified email (docs/03 Q-15) AND the permission.
	// Rate limited: order creation is a money path and a brute-forceable one
	// (99 §7).
	authed.POST("/orders",
		RequireVerifiedEmail(),
		RequirePermission(security.PermOrderCreate),
		RateLimit(d.Limiter, "order", 20),
		func(c *gin.Context) {
			ident, _ := Authenticated(c)

			var body struct {
				Lines []struct {
					ScheduledMealID uuid.UUID `json:"scheduled_meal_id"`
					Qty             int       `json:"qty"`
					AddressID       uuid.UUID `json:"address_id"`
				} `json:"lines"`
				// The courier note typed at checkout (artboard 04). Sanitized
				// and length-capped like every other free-text field — it is
				// printed on a packing label and read off a phone by a
				// courier, so it is an output context too (CLAUDE.md §4).
				DriverNote string `json:"driver_note"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				Fail(c, bindError(err, "Send lines with scheduled_meal_id, qty and address_id."))
				return
			}

			lines := make([]app.CartLine, 0, len(body.Lines))
			for _, l := range body.Lines {
				lines = append(lines, app.CartLine{
					ScheduledMealID: l.ScheduledMealID, Qty: l.Qty, AddressID: l.AddressID,
				})
			}

			// Idempotency-Key is how a double-tapped checkout button creates
			// one order rather than two (PROMPT §14).
			key := c.GetHeader("Idempotency-Key")
			if len(key) > 200 {
				Fail(c, apierror.Validation("Idempotency-Key is too long.", nil))
				return
			}

			// Sanitized on the way IN, and REJECTED rather than repaired
			// (CLAUDE.md §4): this string is printed on a packing label, read
			// off a phone by a courier and rendered in the back office, so it
			// is untrusted customer text in three output contexts.
			note, err := sanitize.Text("driver_note", body.DriverNote, 280)
			if err != nil {
				Fail(c, apierror.Validation("That note is not valid.",
					map[string]any{"driver_note": err.Error()}))
				return
			}

			out, err := d.Ordering.PlaceOrder(c.Request.Context(), ident, app.PlaceOrderInput{
				Lines: lines, IdempotencyKey: key,
				IP: c.ClientIP(), UA: c.Request.UserAgent(),
				DriverNote: note,
			})
			if err != nil {
				Fail(c, err)
				return
			}
			OK(c, http.StatusCreated, out)
		})

	authed.GET("/orders", RequirePermission(security.PermOrderViewOwn), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		page, err := d.Ordering.ListMyOrders(c.Request.Context(), ident, listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	authed.GET("/orders/:id", RequirePermission(security.PermOrderViewOwn), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		ident, _ := Authenticated(c)
		o, err := d.Ordering.GetMyOrder(c.Request.Context(), ident, id)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, o)
	})

	// ── Addresses ───────────────────────────────────────────────────────────
	authed.GET("/addresses", RequirePermission(security.PermAddressManage), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		list, err := d.Ordering.ListAddresses(c.Request.Context(), ident)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, list)
	})

	authed.POST("/addresses", RequirePermission(security.PermAddressManage), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		var body struct {
			Label          string  `json:"label"`
			RecipientName  string  `json:"recipient_name"`
			RecipientPhone string  `json:"recipient_phone"`
			AddressLine    string  `json:"address_line"`
			District       string  `json:"district"`
			City           string  `json:"city"`
			Province       string  `json:"province"`
			PostalCode     string  `json:"postal_code"`
			Latitude       float64 `json:"latitude"`
			Longitude      float64 `json:"longitude"`
			GooglePlaceID  string  `json:"google_place_id"`
			DriverNote     string  `json:"driver_note"`
			IsDefault      bool    `json:"is_default"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send the address with a map pin."))
			return
		}
		out, err := d.Ordering.SaveAddress(c.Request.Context(), ident, app.AddressInput{
			Label: body.Label, RecipientName: body.RecipientName,
			RecipientPhone: body.RecipientPhone, AddressLine: body.AddressLine,
			District: body.District, City: body.City, Province: body.Province,
			PostalCode: body.PostalCode, Latitude: body.Latitude, Longitude: body.Longitude,
			GooglePlaceID: body.GooglePlaceID, DriverNote: body.DriverNote,
			IsDefault: body.IsDefault,
		})
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusCreated, out)
	})
}
