package http

import (
	"html/template"
	"image"
	_ "image/jpeg" // registers the JPEG decoder for image.DecodeConfig
	_ "image/png"  // and the PNG one
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/app"
	"github.com/stevenwilliam/healthy_catering/internal/domain/money"
	"github.com/stevenwilliam/healthy_catering/internal/platform/i18n"
	"github.com/stevenwilliam/healthy_catering/internal/platform/ratelimit"
	"github.com/stevenwilliam/healthy_catering/internal/platform/richtext"
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
	// HeroW/HeroH are its intrinsic pixels, so the browser can reserve the
	// right box before the file arrives. Zero when they could not be read, in
	// which case the template omits the attributes rather than guessing.
	HeroW, HeroH int

	// Openings and the career form's state. Applied is set after a successful
	// POST so the page can confirm rather than silently re-render.
	Openings   []postgres.JobOpening
	Applied    bool
	FormErrors map[string]string
	Form       map[string]string

	// Certifications gates the badge row on the home page. See migration 0024:
	// these are regulated claims, not decoration.
	Certifications bool

	// Ribbon is the corner banner. Off is a supported state, and the wording
	// comes from public_content like the rest of the public copy.
	Ribbon bool

	// Active names the current section, so the masthead can mark which page
	// you are on. Without it every nav item looks identical on every page,
	// which is the one thing a visitor most wants the header to tell them.
	Active string

	// Prices is populated on the price-list route only.
	Prices *app.PublicPriceList
	// ShowMealPrices gates the per-portion table. Off by default since
	// 2026-08-18 (Steven); packages are shown either way.
	ShowMealPrices bool

	// Copy is editable public wording from public_content, keyed the same way
	// as the static catalogue. The template reads it through `c`, which falls
	// back to the catalogue when a key is absent — so a database that has
	// never been edited renders exactly what it always did.
	Copy map[string]string
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
	// PhotoKey is the meal's own picture when one has been uploaded. Empty is
	// the normal case today — object storage is not wired (M9) — and the card
	// then falls back to the diet type's illustrated band.
	PhotoKey string
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
		// c is t for strings the back office can edit: public_content first,
		// the compiled catalogue second. Two layers rather than one because
		// the catalogue is the guarantee that a page always renders — a
		// database row that is missing, empty or not yet translated can never
		// leave a heading blank.
		"c": func(copy map[string]string, l i18n.Locale, key string) string {
			if v, ok := copy[key]; ok && strings.TrimSpace(v) != "" {
				return v
			}
			return publicMessages.T(l, key)
		},
		// dietArt is the decorative corner mark on a diet-type card, chosen by
		// slug. Returns a fallback rather than nothing for an unknown slug.
		"dietArt": dietArt,
		// idr formats whole rupiah the Indonesian way — Rp 500.000 — through the
		// domain formatter, so a price on a marketing page and a price on an
		// invoice cannot be grouped differently.
		"idr": func(v int64) string { return money.Format(money.IDR(v)) },
		// chtml is `c` for the rich-text keys: it renders stored markup
		// UNESCAPED, which is the point of a WYSIWYG field and also the only
		// way script could reach these pages. It goes through the same
		// allowlist the write path used, so a value written by an older build
		// — or straight into the database by hand — still cannot carry a tag
		// this policy does not permit. Sanitised in, sanitised out
		// (CLAUDE.md §4).
		"chtml": func(copy map[string]string, l i18n.Locale, key string) template.HTML {
			v, ok := copy[key]
			if !ok || strings.TrimSpace(v) == "" {
				v = publicMessages.T(l, key)
			}
			return richtext.Render(v)
		},
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
			defaultHeroImage)
		// Editable copy. A failure here is not a failure of the page: the
		// template falls back to the compiled catalogue, so a database blip
		// costs the edits, not the site.
		if d.Content != nil {
			if copy, err := d.Content.ForLocale(ctx, data.L); err == nil {
				data.Copy = copy
			} else if d.Log != nil {
				d.Log.Warn("public content unavailable, falling back to the catalogue",
					"locale", data.L, "error", err)
			}
		}
		data.HeroW, data.HeroH = heroSize(data.HeroImage)
		data.Ribbon = d.Params.Bool(ctx, sysparam.KeyPublicRibbonEnabled, true)
		data.Certifications = d.Params.Bool(ctx, sysparam.KeyPublicCertsEnabled, true)
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
				Active:      "home",
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
				photo := ""
				if m.HeroPhotoKey != nil {
					photo = *m.HeroPhotoKey
				}
				cards = append(cards, mealCard{
					Name: name, DietType: m.DietTypeName, Slot: m.SlotAlias,
					Date: m.ServiceDate, Kcal: m.Nutrition.CaloriesKcal,
					ProteinG: gramsOf(m.Nutrition.ProteinMg), Items: items,
					Complete: m.Nutrition.Complete, PhotoKey: photo,
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
				Active:      "category",
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

	// ── Static-copy pages ───────────────────────────────────────────────────
	//
	// One handler shape for all of them: the template name is the page, and
	// every string on it comes from the catalogue or from public_content, so
	// adding a page is a template plus its keys rather than new plumbing.
	simplePage := func(lang i18n.Locale, tpl, path, titleKey, descKey string) gin.HandlerFunc {
		return func(c *gin.Context) {
			diets, _ := publicDiets(c, d)
			page(c, tpl, PageData{
				L:           lang,
				Active:      tpl,
				Title:       publicMessages.T(lang, titleKey),
				Description: publicMessages.T(lang, descKey),
				Canonical:   base() + i18n.Path(lang, path),
				DietTypes:   diets,
			})
		}
	}

	priceList := func(lang i18n.Locale) gin.HandlerFunc {
		return func(c *gin.Context) {
			diets, _ := publicDiets(c, d)
			data := PageData{
				L:           lang,
				Active:      "pricelist",
				Title:       publicMessages.T(lang, "price.title"),
				Description: publicMessages.T(lang, "price.description"),
				Canonical:   base() + i18n.Path(lang, "/price-list"),
				DietTypes:   diets,
			}
			data.ShowMealPrices = d.Params.Bool(c.Request.Context(),
				sysparam.KeyPublicShowMealPrices, false)

			// A pricing failure must not 500 the page: the template renders an
			// empty-state instead, which is a better answer to a visitor than
			// an error screen.
			if list, err := d.Pricing.PublicList(c.Request.Context()); err == nil {
				// The per-portion rows are not fetched into the page at all
				// when they are hidden, rather than fetched and skipped in the
				// template. A price that never reaches the response cannot be
				// read out of the HTML by someone who thinks to look.
				if !data.ShowMealPrices {
					list.Prices = nil
					list.Tiers = nil
				}
				data.Prices = &list
			} else if d.Log != nil {
				d.Log.Warn("price list unavailable", "error", err)
			}
			page(c, "pricelist", data)
		}
	}

	// ── Career ──────────────────────────────────────────────────────────────
	career := func(lang i18n.Locale) gin.HandlerFunc {
		return func(c *gin.Context) {
			diets, _ := publicDiets(c, d)
			data := PageData{
				L:           lang,
				Active:      "career",
				Title:       publicMessages.T(lang, "career.title"),
				Description: publicMessages.T(lang, "career.description"),
				Canonical:   base() + i18n.Path(lang, "/career"),
				DietTypes:   diets,
				Form:        map[string]string{},
				FormErrors:  map[string]string{},
			}
			if d.Career != nil {
				data.Openings, _ = d.Career.Openings(c.Request.Context())
			}
			page(c, "career", data)
		}
	}

	// The form POST.
	//
	// NO FILES, structurally rather than by validation: the body is capped,
	// multipart is refused outright, and only ParseForm is ever called — the
	// multipart parser is never reached, so there is nothing to write to disk
	// even if someone crafts a request for it.
	applyCareer := func(lang i18n.Locale) gin.HandlerFunc {
		return func(c *gin.Context) {
			diets, _ := publicDiets(c, d)
			data := PageData{
				L:           lang,
				Active:      "career",
				Title:       publicMessages.T(lang, "career.title"),
				Description: publicMessages.T(lang, "career.description"),
				Canonical:   base() + i18n.Path(lang, "/career"),
				DietTypes:   diets,
				Form:        map[string]string{},
				FormErrors:  map[string]string{},
			}
			if d.Career != nil {
				data.Openings, _ = d.Career.Openings(c.Request.Context())
			}

			if ct := c.ContentType(); ct != "application/x-www-form-urlencoded" {
				data.FormErrors["_"] = "unsupported"
				render(c, http.StatusUnsupportedMediaType, "career", data)
				return
			}
			// 32 KB is generous for five text fields and small enough that a
			// large body is refused before it is read into memory.
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
			if err := c.Request.ParseForm(); err != nil {
				data.FormErrors["_"] = "toolarge"
				render(c, http.StatusRequestEntityTooLarge, "career", data)
				return
			}

			// Rate limited by IP: an unauthenticated public form is a spam
			// target, and there is no CAPTCHA configured yet (docs/12).
			if d.Limiter != nil {
				if res := d.Limiter.Allow("career:"+c.ClientIP(),
					ratelimit.Rule{Burst: 5, Window: time.Hour}); !res.Allowed {
					data.FormErrors["_"] = "toomany"
					render(c, http.StatusTooManyRequests, "career", data)
					return
				}
			}

			form := c.Request.PostForm
			in := app.ApplicationInput{
				FullName:  form.Get("full_name"),
				Email:     form.Get("email"),
				Phone:     form.Get("phone"),
				Position:  form.Get("position"),
				Message:   form.Get("message"),
				Locale:    lang,
				IP:        c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
			}
			// Echoed back so a rejected form does not make the applicant retype
			// everything — the single most common reason a form is abandoned.
			data.Form = map[string]string{
				"full_name": in.FullName, "email": in.Email, "phone": in.Phone,
				"position": in.Position, "message": in.Message,
			}

			fieldErrs, err := d.Career.Apply(c.Request.Context(), in)
			if err != nil {
				data.FormErrors["_"] = "failed"
				render(c, http.StatusInternalServerError, "career", data)
				return
			}
			if len(fieldErrs) > 0 {
				data.FormErrors = fieldErrs
				render(c, http.StatusUnprocessableEntity, "career", data)
				return
			}
			data.Applied = true
			data.Form = map[string]string{}
			page(c, "career", data)
		}
	}

	// ── Benefits ────────────────────────────────────────────────────────────
	// Its own page as well as the block on the price list, because it is now a
	// header destination (Steven, 2026-08-18).
	benefits := func(lang i18n.Locale) gin.HandlerFunc {
		return simplePage(lang, "benefits", "/benefits",
			"benefits.title", "benefits.description")
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
		r.GET(prefix+"/price-list", priceList(info.Locale))
		r.GET(prefix+"/contact", simplePage(info.Locale, "contact", "/contact",
			"contact.title", "contact.description"))
		r.GET(prefix+"/about", simplePage(info.Locale, "about", "/about",
			"about.title", "about.description"))
		r.GET(prefix+"/career", career(info.Locale))
		r.POST(prefix+"/career", applyCareer(info.Locale))
		r.GET(prefix+"/benefits", benefits(info.Locale))
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
			"Allow: /price-list",
			"Allow: /contact",
			"Allow: /about",
			"Allow: /career",
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
		paths := []string{"/", "/price-list", "/contact", "/about", "/career"}
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

// defaultHeroImage is used only when the sys_parameters row is missing —
// normally migration 0015/0016 has set it.
const defaultHeroImage = "/images/hero-home.jpg"

// heroSizes memoises intrinsic image dimensions by path.
//
// Note the consequence: replacing the FILE at an unchanged path needs a
// restart to be re-measured. Pointing the parameter at a new path does not,
// because the path is the cache key — and that is the normal way the picture
// gets swapped.
var heroSizes sync.Map // string -> [2]int

// heroSize reads an image's intrinsic width and height.
//
// Without it the template had to hard-code a size, and it hard-coded 800x800
// while the supplied photograph is 800x533. The browser then reserved a square
// box, the image loaded into a 3:2 one, and everything below the hero jumped —
// a layout shift on the largest element of the first screen, which is the
// worst place to have one. image.DecodeConfig reads only the header, not the
// pixels.
//
// Returns 0,0 for anything it cannot measure — a remote URL, or an SVG, which
// the standard library does not decode. The template then omits the attributes
// and an SVG's own viewBox carries the ratio instead.
func heroSize(webPath string) (int, int) {
	if v, ok := heroSizes.Load(webPath); ok {
		d := v.([2]int)
		return d[0], d[1]
	}
	var w, h int
	if rel := strings.TrimPrefix(webPath, "/images/"); rel != webPath && !strings.Contains(rel, "..") {
		if f, err := os.Open(filepath.Join("./web/public/images", filepath.Clean("/"+rel))); err == nil {
			if cfg, _, err := image.DecodeConfig(f); err == nil {
				w, h = cfg.Width, cfg.Height
			}
			_ = f.Close()
		}
	}
	heroSizes.Store(webPath, [2]int{w, h})
	return w, h
}
