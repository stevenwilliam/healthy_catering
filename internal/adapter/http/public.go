package http

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/platform/i18n"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
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
	Lang        string // BCP-47, for <html lang> — "id-ID", "en", "zh-Hans"
	JSONLD      template.JS

	// L is the locale every template lookup goes through.
	L i18n.Locale
	// Languages is this same page in each language: what the selector links
	// to, and what the hreflang alternates advertise. Built per request
	// because the answer depends on the path being viewed.
	Languages []langLink

	BaseURL   string
	Year      int
	Company   map[string]string
	DietTypes []dietLink
	Meals     []mealCard
	Diet      *dietLink
	MapsKey   string
	// HeroImage is the big picture beside the home headline. A sys_parameters
	// row, so swapping it never needs a deploy; empty hides the picture and
	// lets the headline run full width.
	HeroImage string
}

// langLink is one entry in the language selector.
type langLink struct {
	Info   i18n.Info
	URL    string        // path for the link, e.g. "/zh/menu/keto"
	Abs    string        // absolute, for hreflang
	Flag   template.HTML // inline SVG; decorative, the name carries the meaning
	Active bool
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
		// waNumber turns whatever is stored in sys_parameters into the digits
		// wa.me expects. It returns "" when the number is not usable, and the
		// template hides the button on empty — a floating action that opens a
		// broken chat is worse than no button.
		"waNumber": waNumber,
		// t is the only way copy reaches a public page. Templates never carry
		// a literal string, so adding a language is a catalogue edit rather
		// than a template rewrite (CLAUDE.md §10).
		"t": publicMessages.T,
		// path rewrites a locale-free path into the current locale, so a link
		// written once in the template stays inside the language the reader
		// chose. Without it every href would silently drop them back to
		// Indonesian.
		"path": i18n.Path,
	}).Parse(publicTemplates))
	r.SetHTMLTemplate(tpl)

	// Static assets: fonts, tokens, images. Self-hosted, never a CDN (99 §7).
	r.Static("/fonts", "./web/public/fonts")
	r.Static("/css", "./web/public/css")
	// /images was never mounted, so the og:image every page has been
	// advertising — /images/og-default.png — 404'd, and a shared link showed a
	// card with a broken image. The brand assets under it are generated from
	// docs/design_guideline/logo.png by scripts/mklogo.py.
	r.Static("/images", "./web/public/images")

	// The SPA, served by the same binary — one deploy unit, no Node in
	// production (docs/02 D-2).
	//
	// ONE handler rather than a static mount plus a catch-all: gin's router
	// refuses `/app/assets/*filepath` alongside `/app/*path` and panics at
	// startup. A real file is served if it exists, and everything else falls
	// through to index.html so a deep link like /app/orders/<id> survives a
	// hard refresh.
	const spaRoot = "./web/dist"
	spa := func(c *gin.Context) {
		// noindex on the transactional surface, belt and braces with robots.txt.
		c.Header("X-Robots-Tag", "noindex, nofollow")

		rel := strings.TrimPrefix(c.Param("path"), "/")
		// Never join a client-supplied path without cleaning it: "../" is how a
		// static handler serves /etc/passwd.
		if rel != "" && !strings.Contains(rel, "..") {
			full := filepath.Join(spaRoot, filepath.Clean("/"+rel))
			if st, err := os.Stat(full); err == nil && !st.IsDir() {
				// Hashed asset filenames change every build, so they are safe
				// to cache hard.
				if strings.HasPrefix(rel, "assets/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				}
				c.File(full)
				return
			}
		}
		c.Header("Cache-Control", "no-cache")
		c.File(filepath.Join(spaRoot, "index.html"))
	}
	r.GET("/app", spa)
	r.GET("/app/*path", spa)

	base := func() string { return strings.TrimRight(d.Config.App.BaseURL, "/") }

	// render fills in everything the shared "head" and "foot" templates read.
	// Every page has to go through it, including the 404: "foot" carries the
	// floating WhatsApp button, which needs .Company, so a page rendered
	// around it would come out without the button (and with an empty footer).
	render := func(c *gin.Context, status int, name string, data PageData) {
		data.BaseURL = base()
		data.Year = time.Now().Year()
		data.Lang = i18n.Meta(data.L).Tag

		// The same page in every language: the selector's links and the
		// hreflang alternates are the same list. Derived from the path being
		// viewed, so switching language on /zh/menu/keto lands on
		// /en/menu/keto rather than dumping the reader back on the home page —
		// which is the single most annoying thing a language switcher does.
		_, rest := i18n.FromPath(c.Request.URL.Path)
		data.Languages = make([]langLink, 0, len(i18n.Supported))
		for _, info := range i18n.All() {
			u := i18n.Path(info.Locale, rest)
			data.Languages = append(data.Languages, langLink{
				Info:   info,
				URL:    u,
				Abs:    base() + u,
				Flag:   flagFor(info.Locale),
				Active: info.Locale == data.L,
			})
		}

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
		data.HeroImage = d.Params.String(ctx, sysparam.KeyPublicHeroImage,
			"/images/hero-meditation.svg")
		c.HTML(status, name, data)
	}

	page := func(c *gin.Context, name string, data PageData) {
		render(c, http.StatusOK, name, data)
	}

	// ── Home ────────────────────────────────────────────────────────────────
	home := func(loc i18n.Locale) gin.HandlerFunc {
		return func(c *gin.Context) {
			diets, _ := publicDiets(c, d)
			page(c, "home", PageData{
				L:           loc,
				Title:       publicMessages.T(loc, "home.title"),
				Description: publicMessages.T(loc, "home.description"),
				Canonical:   base() + i18n.Path(loc, "/"),
				DietTypes:   diets,
				JSONLD: template.JS(`{"@context":"https://schema.org","@type":"Restaurant",` +
					`"name":"Evermore","servesCuisine":"Healthy","priceRange":"$$",` +
					`"address":{"@type":"PostalAddress","addressLocality":"Jakarta","addressCountry":"ID"},` +
					`"url":"` + base() + `"}`),
			})
		}
	}

	// ── One page per diet type — the SEO surface ────────────────────────────
	menu := func(lang i18n.Locale) gin.HandlerFunc {
		return func(c *gin.Context) {
			dt, err := d.Catalogue.DietTypeBySlug(c.Request.Context(), c.Param("slug"))
			if err != nil {
				diets, _ := publicDiets(c, d)
				render(c, http.StatusNotFound, "notfound", PageData{
					L:           lang,
					Title:       publicMessages.T(lang, "notfound.title"),
					Description: publicMessages.T(lang, "notfound.body"),
					Canonical:   base() + i18n.Path(lang, "/"),
					DietTypes:   diets,
				})
				return
			}

			tz := tzOf(d)
			now := time.Now().In(tz)
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

			// The diet-type name and description are database rows in one
			// language, so they read the same whichever locale is selected —
			// only the frame around them translates. Same for the SEO
			// overrides: a per-locale seo_title would need a column per
			// locale (docs/03 Q-24).
			title := publicMessages.Tf(lang, "menu.title", dt.Name)
			if dt.SEOTitle != nil && *dt.SEOTitle != "" {
				title = *dt.SEOTitle
			}
			desc := dt.Description
			if dt.SEODescription != nil && *dt.SEODescription != "" {
				desc = *dt.SEODescription
			}

			path := i18n.Path(lang, "/menu/"+dt.Slug)
			diets, _ := publicDiets(c, d)
			page(c, "menu", PageData{
				L:           lang,
				Title:       title,
				Description: desc,
				Canonical:   base() + path,
				Diet:        &dietLink{Name: dt.Name, Slug: dt.Slug, Description: dt.Description},
				Meals:       cards, DietTypes: diets,
				JSONLD: template.JS(`{"@context":"https://schema.org","@type":"Menu",` +
					`"name":"` + template.JSEscapeString(dt.Name) + `",` +
					`"inLanguage":"` + i18n.Meta(lang).Tag + `","url":"` + base() + path + `"}`),
			})
		}
	}

	// One set of routes per language. The default locale keeps the bare paths
	// it has always had, so no existing link, bookmark or indexed URL breaks;
	// the other two live under /en and /zh. Path-prefixed rather than a cookie
	// on one URL, because a cookie serves different content at the same
	// address — which breaks sharing a link and gives search engines one URL
	// with three bodies.
	for _, info := range i18n.All() {
		prefix := ""
		if info.Prefix != "" {
			prefix = "/" + info.Prefix
		}
		r.GET(prefix+"/", home(info.Locale))
		r.GET(prefix+"/menu/:slug", menu(info.Locale))
	}

	// ── Public company contact ──────────────────────────────────────────────
	//
	// The SPA needs the WhatsApp number for its floating contact button, and
	// CLAUDE.md §7 is explicit that a value like this is a sys_parameters row
	// rather than a constant — so it cannot be baked into the bundle at build
	// time. Nothing here is new exposure: these four values are already
	// rendered into the footer of every public page.
	r.GET("/api/v1/public/company", func(c *gin.Context) {
		ctx := c.Request.Context()
		c.Header("Cache-Control", "public, max-age=300")
		c.JSON(http.StatusOK, gin.H{
			"name":     d.Params.String(ctx, sysparam.KeyCompanyLegalName, "Evermore"),
			"email":    d.Params.String(ctx, sysparam.KeyCompanyEmail, ""),
			"phone":    d.Params.String(ctx, sysparam.KeyCompanyPhone, ""),
			"whatsapp": waNumber(d.Params.String(ctx, sysparam.KeyCompanyWhatsApp, "")),
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
			// The translated marketing pages are the same surface as the
			// Indonesian one and are meant to be indexed too.
			"Allow: /en",
			"Allow: /zh",
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

		// Every page in every language, and each entry declares the whole set
		// as xhtml:link alternates. A sitemap that lists only the Indonesian
		// URLs leaves the other two to be discovered by luck; listing them
		// without hreflang gets them read as duplicate content.
		paths := []string{"/"}
		for _, dt := range diets {
			paths = append(paths, "/menu/"+dt.Slug)
		}

		var b strings.Builder
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
		b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"` +
			` xmlns:xhtml="http://www.w3.org/1999/xhtml">` + "\n")
		for _, p := range paths {
			priority, changefreq := "0.8", "<changefreq>daily</changefreq>"
			if p == "/" {
				priority, changefreq = "1.0", ""
			}
			for _, info := range i18n.All() {
				b.WriteString("  <url><loc>" + base() + i18n.Path(info.Locale, p) + "</loc>")
				b.WriteString(changefreq + "<priority>" + priority + "</priority>")
				for _, alt := range i18n.All() {
					b.WriteString(`<xhtml:link rel="alternate" hreflang="` + alt.Tag +
						`" href="` + base() + i18n.Path(alt.Locale, p) + `"/>`)
				}
				b.WriteString(`<xhtml:link rel="alternate" hreflang="x-default" href="` +
					base() + i18n.Path(i18n.Default, p) + `"/>`)
				b.WriteString("</url>\n")
			}
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

// waNumber normalises an Indonesian number to the bare international digits
// wa.me needs: 08176315568 -> 628176315568.
//
// It reuses sanitize.Phone rather than trimming a leading zero by hand, so the
// link and the stored contact agree about what a valid number is.
func waNumber(in string) string {
	if strings.TrimSpace(in) == "" {
		return ""
	}
	normalised, err := sanitize.Phone("whatsapp", in)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(normalised, "+")
}
