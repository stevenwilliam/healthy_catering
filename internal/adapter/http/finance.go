package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// registerFinance mounts the verification queue and the package flow.
func registerFinance(g *gin.RouterGroup, d Deps) {
	admin := g.Group("/admin", RequireAuth(d.Auth, d.Signer))

	admin.GET("/payments", RequirePermission(security.PermPaymentVerify), func(c *gin.Context) {
		page, err := d.Finance.Queue(c.Request.Context(), listParams(c), c.Query("status"))
		if err != nil {
			Fail(c, err)
			return
		}
		// Every data grid exports (99 §8), honouring the search and status
		// filter that produced it.
		if c.Query("format") == "csv" {
			rows := make([][]string, 0, len(page.Items))
			for _, q := range page.Items {
				code := ""
				if q.UniqueCode != nil {
					code = strconv.Itoa(*q.UniqueCode)
				}
				submitted := ""
				if q.SubmittedAt != nil {
					submitted = *q.SubmittedAt
				}
				rows = append(rows, []string{
					q.OrderCode, q.CustomerName, q.CustomerEmail, q.Expected,
					code, q.BankName, q.Status, submitted,
					strconv.Itoa(q.WaitingMinutes), strconv.Itoa(q.ProofCount),
				})
			}
			csvOut(c, "payment-queue",
				[]string{"order", "customer", "email", "expected amount",
					"unique code", "bank", "status", "submitted at",
					"waiting minutes", "proofs"}, rows)
			return
		}
		OK(c, http.StatusOK, page)
	})

	admin.POST("/payments/:id/verify", RequirePermission(security.PermPaymentVerify), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		var body struct {
			PaidAmountIDR *int64 `json:"paid_amount_idr"`
		}
		_ = c.ShouldBindJSON(&body)

		out, err := d.Finance.Verify(c.Request.Context(), id, body.PaidAmountIDR, actorFrom(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, out)
	})

	admin.POST("/payments/:id/reject", RequirePermission(security.PermPaymentVerify), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send a reason."))
			return
		}
		if err := d.Finance.Reject(c.Request.Context(), id, body.Reason, actorFrom(c)); err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"status": "REJECTED"})
	})

	// ── Customer-facing ─────────────────────────────────────────────────────
	authed := g.Group("", RequireAuth(d.Auth, d.Signer))

	// Bank details for the transfer instructions.
	authed.GET("/bank-accounts", func(c *gin.Context) {
		list, err := d.Finance.BankAccounts(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, list)
	})

	authed.POST("/orders/:id/payment-proof",
		RequirePermission(security.PermPaymentProofUpload),
		func(c *gin.Context) {
			id, ok := pathUUID(c, "id")
			if !ok {
				return
			}
			ident, _ := Authenticated(c)

			// The proof itself is uploaded to the private bucket by the storage
			// adapter; this records it. Until MinIO is wired (M8 storage), the
			// object key is supplied by the client and validated as a key
			// rather than a path.
			var body struct {
				ObjectKey   string `json:"object_key"`
				ContentType string `json:"content_type"`
				Bytes       int64  `json:"bytes"`
				Checksum    string `json:"checksum"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				Fail(c, bindError(err, "Send object_key, content_type and bytes."))
				return
			}
			if err := d.Ordering.SubmitPaymentProof(c.Request.Context(), ident, id,
				app.ProofInput{
					ObjectKey: body.ObjectKey, ContentType: body.ContentType,
					Bytes: body.Bytes, Checksum: body.Checksum,
				}); err != nil {
				Fail(c, err)
				return
			}
			OK(c, http.StatusOK, gin.H{"status": "PAYMENT_SUBMITTED"})
		})

	// ── Packages ────────────────────────────────────────────────────────────
	g.GET("/packages", func(c *gin.Context) {
		page, err := d.Packages.List(c.Request.Context(), listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	authed.POST("/packages/:id/buy",
		RequireVerifiedEmail(),
		RequirePermission(security.PermOrderCreate),
		RateLimit(d.Limiter, "order", 20),
		func(c *gin.Context) {
			id, ok := pathUUID(c, "id")
			if !ok {
				return
			}
			ident, _ := Authenticated(c)
			out, err := d.Packages.Buy(c.Request.Context(), ident, id,
				c.GetHeader("Idempotency-Key"), c.ClientIP(), c.Request.UserAgent())
			if err != nil {
				Fail(c, err)
				return
			}
			OK(c, http.StatusCreated, out)
		})

	authed.GET("/my/packages", RequirePermission(security.PermPackageViewOwn), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		page, err := d.Packages.MyPackages(c.Request.Context(), ident, listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, page)
	})

	authed.GET("/my/packages/:id/ledger", RequirePermission(security.PermPackageViewOwn), func(c *gin.Context) {
		id, ok := pathUUID(c, "id")
		if !ok {
			return
		}
		ident, _ := Authenticated(c)
		entries, err := d.Packages.Ledger(c.Request.Context(), ident, id)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, entries)
	})

	authed.POST("/my/packages/:id/book",
		RequirePermission(security.PermDeliveryScheduleOwn),
		func(c *gin.Context) {
			id, ok := pathUUID(c, "id")
			if !ok {
				return
			}
			var body struct {
				ScheduledMealID uuid.UUID `json:"scheduled_meal_id"`
				AddressID       uuid.UUID `json:"address_id"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				Fail(c, bindError(err, "Send scheduled_meal_id and address_id."))
				return
			}
			ident, _ := Authenticated(c)
			out, err := d.Packages.Book(c.Request.Context(), ident, app.BookInput{
				PackageID: id, ScheduledMealID: body.ScheduledMealID, AddressID: body.AddressID,
			})
			if err != nil {
				Fail(c, err)
				return
			}
			OK(c, http.StatusCreated, out)
		})

	// ── The expiry sweep, triggerable for operations and tests ──────────────
	admin.POST("/jobs/expire-unpaid", RequirePermission(security.PermSettingsWrite), func(c *gin.Context) {
		n, err := d.Finance.ExpireUnpaid(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, gin.H{"expired": n})
	})

	_ = apierror.CodeInternal
}
