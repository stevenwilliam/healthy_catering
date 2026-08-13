package http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/storage"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// registerUploads mounts the real multipart upload endpoints.
//
// The client sends a file, not a key: the object key is GENERATED server-side
// from the customer and order, so an upload can never escape its prefix or
// overwrite somebody else's proof.
func registerUploads(g *gin.RouterGroup, d Deps) {
	if d.Storage == nil {
		// Without object storage configured the routes are simply absent, which
		// is a clearer failure than a 500 from a nil client.
		return
	}
	authed := g.Group("", RequireAuth(d.Auth, d.Signer))

	authed.POST("/orders/:id/proof",
		RequirePermission(security.PermPaymentProofUpload),
		func(c *gin.Context) {
			id, ok := pathUUID(c, "id")
			if !ok {
				return
			}
			ident, _ := Authenticated(c)
			if ident.CustomerID == nil {
				Fail(c, apierror.Forbidden(apierror.CodeForbidden,
					"Only customers upload payment proof."))
				return
			}

			// The multipart body gets its own limit, larger than the JSON cap
			// but still bounded.
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, storage.MaxProofBytes+(1<<20))

			file, err := c.FormFile("file")
			if err != nil {
				// MaxBytesReader kills the body mid-parse, so an over-sized
				// upload arrives here looking like a MISSING one. Telling a
				// customer to attach the file they just attached is the kind
				// of message that generates a support call.
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) || strings.Contains(err.Error(), "too large") {
					Fail(c, apierror.New(http.StatusRequestEntityTooLarge,
						apierror.CodeValidation, "That file is too large — 5 MB maximum."))
					return
				}
				Fail(c, apierror.Validation("Attach the transfer proof as `file`.",
					map[string]any{"file": "required"}))
				return
			}
			if file.Size > storage.MaxProofBytes {
				Fail(c, apierror.Validation("The file must be 5 MB or smaller.",
					map[string]any{"file": "5 MB maximum"}))
				return
			}

			src, err := file.Open()
			if err != nil {
				Fail(c, apierror.Internal(err))
				return
			}
			defer src.Close()

			up, err := d.Storage.PutProof(c.Request.Context(), *ident.CustomerID, id, src, file.Size)
			if err != nil {
				Fail(c, uploadError(err))
				return
			}

			if err := d.Ordering.RecordProof(c.Request.Context(), ident, id,
				up.Key, up.ContentType, up.Bytes); err != nil {
				Fail(c, err)
				return
			}
			OK(c, http.StatusCreated, gin.H{
				"status":       "PAYMENT_SUBMITTED",
				"content_type": up.ContentType,
				"bytes":        up.Bytes,
			})
		})

	// Finance views a proof through a SHORT-LIVED presigned URL. The bucket is
	// private, so this is the only way to see one, and a link pasted into a
	// chat stops working (99 §7).
	authed.GET("/admin/payments/:id/proof",
		RequirePermission(security.PermPaymentVerify),
		func(c *gin.Context) {
			id, ok := pathUUID(c, "id")
			if !ok {
				return
			}
			keys, err := d.Finance.ProofKeys(c.Request.Context(), id)
			if err != nil {
				Fail(c, err)
				return
			}
			urls := make([]string, 0, len(keys))
			for _, k := range keys {
				u, err := d.Storage.PresignedURL(c.Request.Context(), k, 10*time.Minute)
				if err != nil {
					Fail(c, apierror.Internal(err))
					return
				}
				urls = append(urls, u)
			}
			OK(c, http.StatusOK, gin.H{"urls": urls, "expires_in_seconds": 600})
		})
}

func isErr(err, target error) bool { return errors.Is(err, target) }

// uploadError turns a storage rejection into a customer-facing message.
func uploadError(err error) error {
	switch {
	case isErr(err, storage.ErrUnsupported):
		// The message names what IS accepted rather than what was wrong: the
		// customer photographed a receipt and needs to know what to send.
		return apierror.Validation(
			"That file is not a JPEG, PNG, WebP or PDF. We check the file itself, not its name.",
			map[string]any{"file": "photo or PDF of your transfer"})
	case isErr(err, storage.ErrTooLarge):
		return apierror.Validation("The file must be 5 MB or smaller.",
			map[string]any{"file": "5 MB maximum"})
	case isErr(err, storage.ErrEmpty):
		return apierror.Validation("That file is empty.", map[string]any{"file": "required"})
	default:
		return apierror.Internal(err)
	}
}
