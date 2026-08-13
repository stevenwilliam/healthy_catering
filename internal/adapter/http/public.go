package http

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// PageData is what every server-rendered page needs.
//
// The SEO fields are not optional decoration: link-preview bots do not run
// JavaScript, so a client-rendered page pasted into WhatsApp shows a blank
// card. For a business whose customers share links in chat, this is the
// highest-value SEO item there is (99 §13) — which is exactly why the public
// surface is rendered here rather than in the SPA.
type PageData struct {
	Title       string
	Description string
	Canonical   string
	OGImage     string
	Lang        string
	JSONLD      template.JS

	BaseURL   string
	Year      int
	Company   map[string]string
	DietTypes []dietLink
	Meals     []mealCard
	Diet      *dietLink
	MapsKey   string
}

type dietLink struct {
	Name        string
	Slug        string
	Description string
}

type mealCard struct {
	Name     string
	DietType string
	Slot     string
	Date     string
	Kcal     int
	ProteinG string
	Items    []string
	Complete bool
}

// registerPublic mounts the server-rendered marketing and menu pages.
//
// Go templates rather than a second Node runtime (docs/02 D-2): the public
// surface is eight routes, all of which need correct OG tags and fresh daily
// content, and rendering them from the same binary means one deploy unit.
func registerPublicPages(r *gin.Engine, d Deps) {
	tpl := template.Must(template.New("").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(publicTemplates))
	r.SetHTMLTemplate(tpl)

	// Static assets: fonts, tokens, images. Self-hosted, never a CDN (99 §7).
	r.Static("/fonts", "./web/public/fonts")
	r.Static("/css", "./web/public/css")

	base := func() string { return strings.TrimRight(d.Config.App.BaseURL, "/") }

	page := func(c *gin.Context, name string, data PageData) {
		data.BaseURL = base()
		data.Year = time.Now().Year()
		data.Lang = "id"
		if data.OGImage == "" {
			data.OGImage = base() + "/images/og-default.png"
		}
		ctx := c.Request.Context()
		data.Company = map[string]string{
			"name":     d.Params.String(ctx, sysparam.KeyCompanyLegalName, "Evermore"),
			"email":    d.Params.String(ctx, sysparam.KeyCompanyEmail, ""),
			"phone":    d.Params.String(ctx, sysparam.KeyCompanyPhone, ""),
			"whatsapp": d.Params.String(ctx, sysparam.KeyCompanyWhatsApp, ""),
		}
		data.MapsKey = d.Config.Maps.BrowserKey
		c.HTML(http.StatusOK, name, data)
	}

	// ── Home ────────────────────────────────────────────────────────────────
	r.GET("/", func(c *gin.Context) {
		diets, _ := publicDiets(c, d)
		page(c, "home", PageData{
			Title: "Evermore — katering sehat harian di Jakarta",
			Description: "Makanan sehat harian diantar ke rumah atau kantor Anda di Jakarta. " +
				"Pilih menu sesuai kebutuhan: Healthy, Weight Loss, High Protein dan lainnya.",
			Canonical: base() + "/",
			DietTypes: diets,
			JSONLD: template.JS(`{"@context":"https://schema.org","@type":"Restaurant",` +
				`"name":"Evermore","servesCuisine":"Healthy","priceRange":"$$",` +
				`"address":{"@type":"PostalAddress","addressLocality":"Jakarta","addressCountry":"ID"},` +
				`"url":"` + base() + `"}`),
		})
	})

	// ── One page per diet type — the SEO surface ────────────────────────────
	r.GET("/menu/:slug", func(c *gin.Context) {
		dt, err := d.Catalogue.DietTypeBySlug(c.Request.Context(), c.Param("slug"))
		if err != nil {
			c.HTML(http.StatusNotFound, "notfound", PageData{
				Title: "Halaman tidak ditemukan — Evermore", Lang: "id", Year: time.Now().Year(),
			})
			return
		}

		loc := tzOf(d)
		now := time.Now().In(loc)
		meals, _ := d.Catalogue.Calendar(c.Request.Context(), app.CalendarQuery{
			From:       now.Format("2006-01-02"),
			To:         now.AddDate(0, 0, 7).Format("2006-01-02"),
			DietTypeID: &dt.ID, PublicOnly: true,
		})

		cards := make([]mealCard, 0, len(meals))
		for _, m := range meals {
			items := make([]string, 0, len(m.Items))
			for _, it := range m.Items {
				items = append(items, it.FoodName)
			}
			name := ""
			if m.Name != nil {
				name = *m.Name
			}
			cards = append(cards, mealCard{
				Name: name, DietType: m.DietTypeName, Slot: m.SlotAlias,
				Date: m.ServiceDate, Kcal: m.Nutrition.CaloriesKcal,
				ProteinG: gramsOf(m.Nutrition.ProteinMg), Items: items,
				Complete: m.Nutrition.Complete,
			})
		}

		title := dt.Name + " — menu minggu ini | Evermore"
		if dt.SEOTitle != nil && *dt.SEOTitle != "" {
			title = *dt.SEOTitle
		}
		desc := dt.Description
		if dt.SEODescription != nil && *dt.SEODescription != "" {
			desc = *dt.SEODescription
		}

		diets, _ := publicDiets(c, d)
		page(c, "menu", PageData{
			Title: title, Description: desc,
			Canonical: base() + "/menu/" + dt.Slug,
			Diet:      &dietLink{Name: dt.Name, Slug: dt.Slug, Description: dt.Description},
			Meals:     cards, DietTypes: diets,
			JSONLD: template.JS(`{"@context":"https://schema.org","@type":"Menu",` +
				`"name":"` + template.JSEscapeString(dt.Name) + `",` +
				`"inLanguage":"id-ID","url":"` + base() + "/menu/" + dt.Slug + `"}`),
		})
	})

	// ── SEO plumbing ────────────────────────────────────────────────────────
	r.GET("/robots.txt", func(c *gin.Context) {
		// The transactional surface is disallowed: crawlers do reach a cart or
		// an order page, and a transactional page in an index is a support
		// problem rather than a ranking one (99 §13).
		c.String(http.StatusOK, strings.Join([]string{
			"User-agent: *",
			"Allow: /$",
			"Allow: /menu",
			"Disallow: /admin",
			"Disallow: /app",
			"Disallow: /cart",
			"Disallow: /checkout",
			"Disallow: /orders",
			"Disallow: /my",
			"Disallow: /api/",
			"",
			"Sitemap: " + base() + "/sitemap.xml",
			"",
		}, "\n"))
	})

	r.GET("/sitemap.xml", func(c *gin.Context) {
		diets, _ := publicDiets(c, d)
		var b strings.Builder
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
		b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
		b.WriteString("  <url><loc>" + base() + "/</loc><priority>1.0</priority></url>\n")
		for _, dt := range diets {
			b.WriteString("  <url><loc>" + base() + "/menu/" + dt.Slug +
				"</loc><changefreq>daily</changefreq><priority>0.8</priority></url>\n")
		}
		b.WriteString("</urlset>\n")
		c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(b.String()))
	})
}

func publicDiets(c *gin.Context, d Deps) ([]dietLink, error) {
	active := true
	page, err := d.Admin.ListDietTypes(c.Request.Context(),
		listParamsWithActive(&active))
	if err != nil {
		return nil, err
	}
	out := make([]dietLink, 0, len(page.Items))
	for _, dt := range page.Items {
		out = append(out, dietLink{Name: dt.Name, Slug: dt.Slug, Description: dt.Description})
	}
	return out, nil
}

func gramsOf(mg int) string {
	whole := mg / 1000
	frac := (mg % 1000) / 100
	if frac == 0 {
		return itoa(whole)
	}
	return itoa(whole) + "." + itoa(frac)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
