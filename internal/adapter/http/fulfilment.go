package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// registerFulfilment mounts the kitchen and courier endpoints.
//
// These are the screens a kitchen lead and a driver actually use, so they are
// deliberately small: a list they can filter, and one button each.
func registerFulfilment(g *gin.RouterGroup, d Deps) {
	authed := g.Group("", RequireAuth(d.Auth, d.Signer))

	authed.GET("/admin/deliveries", RequirePermission(security.PermDeliveryRead), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		page, err := d.Fulfilment.List(c.Request.Context(), ident, listParams(c),
			c.Query("from"), c.Query("to"), c.Query("status"))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	// Kitchen marks prepared; courier marks out-for-delivery and delivered.
	// Both hold delivery.fulfil, and the DOMAIN machine decides which moves are
	// legal from where — so a courier cannot mark something delivered that the
	// kitchen never cooked.
	authed.POST("/admin/deliveries/:id/status",
		RequirePermission(security.PermDeliveryFulfil),
		func(c *gin.Context) {
			id, ok := pathUUID(c, "id")
			if !ok {
				return
			}
			var body struct {
				Status string `json:"status"`
				Reason string `json:"reason"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				Fail(c, bindError(err, "Send status: PREPARING, OUT_FOR_DELIVERY, DELIVERED or FAILED."))
				return
			}
			ident, _ := Authenticated(c)
			if err := d.Fulfilment.Advance(c.Request.Context(), ident, id,
				body.Status, body.Reason, actorFrom(c)); err != nil {
				Fail(c, err)
				return
			}
			OK(c, http.StatusOK, gin.H{"id": id, "status": body.Status})
		})

	authed.POST("/admin/deliveries/:id/reassign",
		RequirePermission(security.PermDeliveryReassign),
		func(c *gin.Context) {
			id, ok := pathUUID(c, "id")
			if !ok {
				return
			}
			var body struct {
				KitchenID uuid.UUID `json:"kitchen_id"`
				Reason    string    `json:"reason"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				Fail(c, bindError(err, "Send kitchen_id and a reason."))
				return
			}
			if err := d.Fulfilment.Reassign(c.Request.Context(), id,
				body.KitchenID, body.Reason, actorFrom(c)); err != nil {
				Fail(c, err)
				return
			}
			OK(c, http.StatusOK, gin.H{"id": id, "assignment_mode": "MANUAL"})
		})

	// A customer skips their own delivery; staff can skip on their behalf.
	authed.POST("/deliveries/:id/skip",
		RequirePermission(security.PermDeliveryScheduleOwn, security.PermDeliveryReassign),
		func(c *gin.Context) {
			id, ok := pathUUID(c, "id")
			if !ok {
				return
			}
			var body struct {
				Reason string `json:"reason"`
			}
			_ = c.ShouldBindJSON(&body)

			ident, _ := Authenticated(c)
			out, err := d.Fulfilment.Skip(c.Request.Context(), ident, id, body.Reason, actorFrom(c))
			if err != nil {
				Fail(c, err)
				return
			}
			OK(c, http.StatusOK, out)
		})

	authed.GET("/my/deliveries", RequirePermission(security.PermDeliveryViewOwn), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		page, err := d.Ordering.MyDeliveries(c.Request.Context(), ident, listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})
}
