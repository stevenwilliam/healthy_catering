package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/config"
	"github.com/stevenwilliam/healthy_catering/internal/platform/ratelimit"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// bindError distinguishes an over-sized body from malformed JSON. Both are
// rejections, but "your request was too large" and "your JSON is wrong" send a
// developer to different places, and a body killed by the size cap would
// otherwise be reported as a field problem it never reached.
func bindError(err error, msg string) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return apierror.New(http.StatusRequestEntityTooLarge, apierror.CodeValidation,
			"That request is too large.")
	}
	return apierror.Validation(msg, nil)
}

// badInput turns a sanitize rejection into the standard validation error,
// naming the field so the form can point at it.
func badInput(err error) error {
	var se *sanitize.Error
	if errors.As(err, &se) {
		return apierror.Validation("Please check the highlighted field.",
			map[string]any{se.Field: se.Reason})
	}
	return apierror.Validation("Invalid input.", nil)
}

// Deps is everything the router needs, wired in cmd/api/main.go.
type Deps struct {
	Config         *config.Config
	Log            *slog.Logger
	Serviceability *app.Serviceability
	Auth           *app.Auth
	Admin          *app.Admin
	Catalogue      *app.Catalogue
	Pricing        *app.Pricing
	Ordering       *app.Ordering
	Finance        *app.Finance
	Packages       *app.Packages
	Reports        *app.Reports
	Params         *sysparam.Store
	Signer         *security.TokenSigner
	Limiter        *ratelimit.Limiter
	Health         func() error
	// OnVerificationToken hands a freshly minted verification token to the
	// caller to mail. The auth service does not know about the mailer, so a
	// test can assert on the token without a mail server.
	OnVerificationToken func(userID uuid.UUID, token string)
}

// New builds the router.
//
// The API is versioned from the first commit because phase 2 is a mobile client
// against this same contract (PROMPT §1) — a v1 added later is a migration for
// everyone who already shipped.
func New(d Deps) *gin.Engine {
	if d.Config.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	_ = r.SetTrustedProxies(d.Config.App.TrustedProxies)

	r.Use(RequestID(), Logger(d.Log), Recovery(d.Log),
		SecurityHeaders(d.Config.App.IsProduction()),
		CORS(d.Config.App.AllowedOrigins),
		// 1 MiB is generous for JSON; file uploads get their own limit on
		// their own route.
		MaxBody(1<<20))

	r.NoRoute(func(c *gin.Context) { Fail(c, apierror.NotFound("No such endpoint.")) })

	// Operational endpoints, outside the versioned API.
	r.GET("/healthz", func(c *gin.Context) {
		if d.Health != nil {
			if err := d.Health(); err != nil {
				Fail(c, apierror.Internal(err))
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// The server-rendered public surface: home, one page per diet type, and
	// the SEO plumbing (docs/02 D-2).
	registerPublicPages(r, d)

	v1 := r.Group("/api/v1")
	registerPublic(v1, d)
	registerAuth(v1, d)
	registerAdmin(v1, d)
	registerCatalogue(v1, d)
	registerPricing(v1, d)
	registerOrders(v1, d)
	registerFinance(v1, d)
	registerReports(v1, d)
	return r
}

func registerPublic(g *gin.RouterGroup, d Deps) {
	// "Do we deliver to you?" — the address form, checkout and the homepage
	// widget all ask this one endpoint (PROMPT §9.3).
	g.POST("/delivery-area/check", func(c *gin.Context) {
		var body struct {
			Lat      float64 `json:"lat"`
			Lng      float64 `json:"lng"`
			SlotID   string  `json:"slot_id"`
			Date     string  `json:"date"`
			Qty      int     `json:"qty"`
			District string  `json:"district"`
			City     string  `json:"city"`
			Source   string  `json:"source"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			Fail(c, bindError(err, "Send lat and lng as numbers."))
			return
		}
		if body.Lat == 0 && body.Lng == 0 {
			Fail(c, apierror.Validation("A map pin is required.", map[string]any{
				"lat": "required", "lng": "required",
			}))
			return
		}

		// Free text is normalized and bounded server-side, and the enum is
		// allow-listed. The browser checks these too, for feedback — but the
		// browser can be bypassed, so nothing here trusts it (CLAUDE.md §4).
		district, err := sanitize.Text("district", body.District, 120)
		if err != nil {
			Fail(c, badInput(err))
			return
		}
		city, err := sanitize.Text("city", body.City, 120)
		if err != nil {
			Fail(c, badInput(err))
			return
		}
		source := "WIDGET"
		if body.Source != "" {
			source, err = sanitize.Enum("source", body.Source, "WIDGET", "ADDRESS_FORM", "CHECKOUT")
			if err != nil {
				Fail(c, badInput(err))
				return
			}
		}
		if body.Qty < 0 || body.Qty > 999 {
			Fail(c, apierror.Validation("qty must be between 1 and 999.", nil))
			return
		}

		in := app.CheckInput{
			Lat: body.Lat, Lng: body.Lng, Qty: body.Qty,
			District: district, City: city, Source: source,
		}
		if body.SlotID != "" {
			id, err := uuid.Parse(body.SlotID)
			if err != nil {
				Fail(c, apierror.Validation("slot_id must be a UUID.", nil))
				return
			}
			in.SlotID = id
		}
		// The date is a business calendar date; parsing it here keeps
		// Asia/Jakarta conversion in one place.
		loc, _ := d.Config.Locale.TZ()
		if loc == nil {
			loc = time.UTC
		}
		if body.Date != "" {
			t, err := time.ParseInLocation("2006-01-02", body.Date, loc)
			if err != nil {
				Fail(c, apierror.Validation("date must be YYYY-MM-DD.", nil))
				return
			}
			in.Date = t
		} else {
			now := time.Now().In(loc)
			in.Date = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
		}

		res, err := d.Serviceability.Check(c.Request.Context(), in)
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, res)
	})

	// Customers see the alias only; the exact time is internal (PROMPT §8.1).
	g.GET("/delivery-slots", func(c *gin.Context) {
		slots, err := d.Serviceability.Slots(c.Request.Context())
		if err != nil {
			Fail(c, apierror.Internal(err))
			return
		}
		OK(c, http.StatusOK, slots)
	})
}

// parseQtyParam is a small helper kept here rather than duplicated per handler.
func parseQtyParam(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return def
	}
	return n
}
