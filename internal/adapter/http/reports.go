package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// registerReports mounts the operational reports.
//
// Every one takes ?format=csv, because "all reports: date-range filter,
// CSV/Excel export, printable A4 layout" is the requirement (PROMPT §12) and a
// report nobody can export is a report nobody uses.
func registerReports(g *gin.RouterGroup, d Deps) {
	rep := g.Group("/admin/reports", RequireAuth(d.Auth, d.Signer))

	scopeOf := func(c *gin.Context) app.Scope {
		s := app.Scope{From: c.Query("from"), To: c.Query("to"), Q: c.Query("q")}
		if v := c.Query("kitchen_id"); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				s.KitchenID = &id
			}
		}
		if v := c.Query("slot_id"); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				s.SlotID = &id
			}
		}
		return s
	}

	rep.GET("/production", RequirePermission(security.PermReportRead), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		rows, err := d.Reports.ProductionSheet(c.Request.Context(), ident, scopeOf(c))
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			csvOut(c, "production-sheet",
				[]string{"date", "slot", "kitchen", "diet type", "dish", "role", "portions"},
				func() [][]string {
					out := make([][]string, 0, len(rows))
					for _, r := range rows {
						out = append(out, []string{r.ServiceDate, r.Slot, r.Kitchen,
							r.DietType, r.FoodName, r.ItemRole, strconv.Itoa(r.Portions)})
					}
					return out
				}())
			return
		}
		OK(c, http.StatusOK, rows)
	})

	rep.GET("/packing-labels", RequirePermission(security.PermReportRead), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		rows, err := d.Reports.PackingLabels(c.Request.Context(), ident, scopeOf(c))
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			csvOut(c, "packing-labels",
				[]string{"delivery", "date", "slot", "kitchen", "customer", "phone",
					"address", "district", "diet type", "qty", "dishes", "allergens", "note"},
				func() [][]string {
					out := make([][]string, 0, len(rows))
					for _, r := range rows {
						out = append(out, []string{r.DeliveryCode, r.ServiceDate, r.Slot,
							r.Kitchen, r.CustomerName, r.Phone, r.AddressLine, r.District,
							r.DietType, strconv.Itoa(r.Qty), r.Foods, r.Allergens, r.DriverNote})
					}
					return out
				}())
			return
		}
		OK(c, http.StatusOK, rows)
	})

	rep.GET("/manifest", RequirePermission(security.PermDeliveryRead), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		rows, err := d.Reports.CourierManifest(c.Request.Context(), ident, scopeOf(c))
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			csvOut(c, "courier-manifest",
				[]string{"stop", "delivery", "customer", "phone", "address", "district",
					"lat", "lng", "maps", "distance m", "meals", "note", "status"},
				func() [][]string {
					out := make([][]string, 0, len(rows))
					for _, r := range rows {
						out = append(out, []string{strconv.Itoa(r.Seq), r.DeliveryCode,
							r.CustomerName, r.Phone, r.AddressLine, r.District,
							fmt.Sprintf("%.6f", r.Latitude), fmt.Sprintf("%.6f", r.Longitude),
							r.MapsURL, strconv.Itoa(r.DistanceM), strconv.Itoa(r.Meals),
							r.DriverNote, r.Status})
					}
					return out
				}())
			return
		}
		OK(c, http.StatusOK, rows)
	})

	rep.GET("/coverage", RequirePermission(security.PermReportRead), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		rows, err := d.Reports.Coverage(c.Request.Context(), ident, scopeOf(c))
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			csvOut(c, "coverage",
				[]string{"district", "city", "attempts", "notify requests",
					"avg distance to nearest (km)", "nearest kitchen"},
				func() [][]string {
					out := make([][]string, 0, len(rows))
					for _, r := range rows {
						out = append(out, []string{
							r.District, r.City,
							strconv.Itoa(r.Attempts), strconv.Itoa(r.NotifyRequests),
							strconv.FormatFloat(r.AvgDistanceKM, 'f', 1, 64),
							r.NearestKitchen,
						})
					}
					return out
				}())
			return
		}
		OK(c, http.StatusOK, rows)
	})

	rep.GET("/sales", RequirePermission(security.PermReportFinancial), func(c *gin.Context) {
		ident, _ := Authenticated(c)
		rows, err := d.Reports.Sales(c.Request.Context(), ident, scopeOf(c), c.Query("group_by"))
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			csvOut(c, "sales",
				[]string{"period", "customer type", "diet type", "order type", "orders",
					"meals", "gross", "tax base", "tax", "promo discount"},
				func() [][]string {
					out := make([][]string, 0, len(rows))
					for _, r := range rows {
						out = append(out, []string{r.Period, r.CustomerType, r.DietType,
							r.OrderType, strconv.Itoa(r.Orders), strconv.Itoa(r.Meals),
							strconv.FormatInt(r.GrossIDR, 10),
							strconv.FormatInt(r.TaxBaseIDR, 10),
							strconv.FormatInt(r.TaxIDR, 10),
							strconv.FormatInt(r.DiscountGivenIDR, 10)})
					}
					return out
				}())
			return
		}
		OK(c, http.StatusOK, rows)
	})

	rep.GET("/credits", RequirePermission(security.PermReportRead), func(c *gin.Context) {
		page, err := d.Reports.Credits(c.Request.Context(), listParams(c))
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			csvOut(c, "customer-credits",
				[]string{"datetime today", "customer", "email", "package",
					"date purchase", "date expired", "purchased credit", "remaining credit", "status"},
				func() [][]string {
					out := make([][]string, 0, len(page.Items))
					for _, r := range page.Items {
						out = append(out, []string{r.DatetimeToday, r.CustomerName,
							r.CustomerEmail, r.PackageName, r.DatePurchase, r.DateExpired,
							strconv.Itoa(r.PurchasedCredit), strconv.Itoa(r.RemainingCredit), r.Status})
					}
					return out
				}())
			return
		}
		OK(c, http.StatusOK, page)
	})

	rep.GET("/unpaid", RequirePermission(security.PermReportRead), func(c *gin.Context) {
		rows, err := d.Reports.UnpaidAndExpiring(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		if c.Query("format") == "csv" {
			csvOut(c, "unpaid-and-expiring",
				[]string{"kind", "reference", "customer", "amount", "deadline",
					"minutes left"},
				func() [][]string {
					out := make([][]string, 0, len(rows))
					for _, r := range rows {
						out = append(out, []string{
							r.Kind, r.Reference, r.CustomerName, r.Amount,
							r.Deadline, strconv.Itoa(r.MinutesLeft),
						})
					}
					return out
				}())
			return
		}
		OK(c, http.StatusOK, rows)
	})

	rep.GET("/retention", RequirePermission(security.PermReportFinancial), func(c *gin.Context) {
		rows, err := d.Reports.Retention(c.Request.Context())
		if err != nil {
			Fail(c, err)
			return
		}
		OK(c, http.StatusOK, rows)
	})
}

// csvOut streams a report as CSV with the formula guard applied.
func csvOut(c *gin.Context, name string, headers []string, rows [][]string) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	// The filename is ours, never echoed from user input — a header built from
	// a query parameter is a response-splitting bug.
	c.Header("Content-Disposition", `attachment; filename="`+name+`.csv"`)
	if err := app.WriteCSV(c.Writer, headers, rows); err != nil {
		Fail(c, apierror.Internal(err))
	}
}
