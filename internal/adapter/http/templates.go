package http

// publicTemplates are the server-rendered public pages.
//
// Deliberately small and dependency-free: every colour is a token from
// docs/10-design-system.md with a measured contrast ratio, the fonts are
// self-hosted, and there is no inline script — the CSP has no 'unsafe-inline'
// so an inline <script> would simply not run.
//
// Two grounds: the PAGE is deep #1C3D34 (beige on it is 11.32, so body copy
// can sit straight on it) and the BARS — masthead and footer — are mid #468973,
// where beige is only 3.93 and every string must be large. See the header
// comment in web/public/css/public.css.
//
// NO LITERAL COPY. Every visible string goes through `t`, and every internal
// link through `path`, so the page renders in the reader's language and stays
// there. Adding a fourth language is then a catalogue edit in messages.go,
// not a template rewrite.
const publicTemplates = `
{{define "head"}}
<!doctype html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<meta name="description" content="{{.Description}}">
<link rel="canonical" href="{{.Canonical}}">
<meta name="theme-color" content="#1C3D34">

<!-- Every language of this page, so a search engine treats the three as one
     page in three languages rather than as duplicates competing with each
     other. x-default points at Indonesian, the default locale. -->
{{range .Languages}}<link rel="alternate" hreflang="{{.Info.Tag}}" href="{{.Abs}}">
{{end}}<link rel="alternate" hreflang="x-default" href="{{.BaseURL}}/">

<!-- Open Graph, STATIC in the served HTML. Preview bots do not run
     JavaScript, so these have to be here rather than set by a script
     (99 §13). This is what makes a WhatsApp share show a card. -->
<meta property="og:type" content="website">
<meta property="og:site_name" content="Evermore">
<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Description}}">
<meta property="og:url" content="{{.Canonical}}">
<meta property="og:image" content="{{.OGImage}}">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:locale" content="{{.Lang}}">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{{.Title}}">
<meta name="twitter:description" content="{{.Description}}">
<meta name="twitter:image" content="{{.OGImage}}">

<!-- Favicon. The mark is the wordmark's leading 'e' on the masthead fill,
     derived by scripts/mklogo.py — the full lockup is 104:11 and turns to mush
     at 16px. .ico first for the browsers that still ask for it by convention. -->
<link rel="icon" href="/images/favicon.ico" sizes="any">
<link rel="icon" type="image/png" sizes="32x32" href="/images/favicon-32.png">
<link rel="apple-touch-icon" href="/images/apple-touch-icon.png">
<link rel="stylesheet" href="/fonts/fonts.css?v={{.AssetV}}">
<link rel="stylesheet" href="/css/tokens.css?v={{.AssetV}}">
<link rel="stylesheet" href="/css/public.css?v={{.AssetV}}">
<link rel="preload" href="/fonts/erode/Erode-Bold.woff2" as="font" type="font/woff2" crossorigin>
<link rel="preload" href="/fonts/inter/InterVariable.woff2" as="font" type="font/woff2" crossorigin>
{{if .JSONLD}}<script type="application/ld+json">{{.JSONLD}}</script>{{end}}
</head>
<body>
{{if .Ribbon}}
<!-- Corner ribbon. Real text in the DOM rather than a background image, so a
     screen reader announces the offer and a translation can change its length
     without redrawing anything. pointer-events are off in CSS: it sits over
     the top-right corner, and a decorative banner must not swallow clicks
     meant for what is underneath it. -->
<div class="ribbon"><span>{{c .Copy .L "ribbon.text"}}</span></div>
{{end}}
<header class="masthead">
  <!-- The supplied wordmark, reversed out for the dark fill. width/height are
       the intrinsic pixels so the masthead does not reflow as it loads; the
       stylesheet constrains the displayed height. -->
  <a class="wordmark" href="{{path .L "/"}}" aria-label="{{t .L "nav.home_aria"}}">
    <!-- The mark is DECORATION here: alt="" deliberately. The link already
         carries an aria-label, and the wordmark beside it carries alt.
         Without the empty alt a screen reader announces the brand three
         times for one link. -->
    <img class="wordmark-mark" src="/images/evermore-mark-128.png?v={{.AssetV}}"
         width="128" height="128" alt="">
    <img class="wordmark-type" src="/images/evermore-wordmark-light.png?v={{.AssetV}}"
         width="560" height="60" alt="Evermore">
  </a>
  <!-- The menu is rendered TWICE and one is always display:none.
       Below 48rem it is a burger; above, a plain row. A single copy toggled by
       CSS is not possible without JavaScript: <details> hides its own content
       when closed, and no CSS property reliably un-hides it, so the desktop
       row could not be forced open. Duplicating the markup is the honest
       trade — the hidden copy is display:none, which takes it out of the
       accessibility tree too, so a screen reader is offered exactly one menu.
       Both come from the same .Nav loop, so they cannot drift. -->
  <nav class="nav-wide" aria-label="{{t .L "nav.main"}}">
    <!-- Data-driven (migration 0026): which items appear and in what order is
         configuration, not markup. CATEGORY is the diet-type dropdown; the
         rest are plain links. Labels come from the message catalogue by
         label_key, because a label typed into an admin box would exist in one
         language only. -->
    {{range .Nav}}
    {{if eq .Kind "CATEGORY"}}
    <details class="navdrop"{{if eq $.Active .ActiveKey}} data-current="true"{{end}}>
      <summary>{{t $.L .LabelKey}}
        <svg class="langpick-caret" viewBox="0 0 10 6" aria-hidden="true" focusable="false">
          <path d="M1 1l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.6"
                stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </summary>
      <ul class="navdrop-menu">
        {{range $.DietTypes}}
        <li><a href="{{path $.L (printf "/menu/%s" .Slug)}}">{{.Name}}</a></li>
        {{end}}
      </ul>
    </details>
    {{else}}
    <a href="{{if .IsLocalised}}{{path $.L .Path}}{{else}}{{.Path}}{{end}}"{{if eq $.Active .ActiveKey}} aria-current="page"{{end}}>
      {{if eq .Icon "cart"}}
      <!-- The icon is decoration beside a real label, so it is aria-hidden:
           a screen reader reads "Pesan", not "cart Pesan". -->
      <svg class="nav-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path d="M2 3h3l2.6 11.2a2 2 0 0 0 2 1.6h7.7a2 2 0 0 0 2-1.5L21 7H6.2"
              fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round"/>
        <circle cx="10" cy="20" r="1.6"/><circle cx="18" cy="20" r="1.6"/>
      </svg>
      {{end}}{{t $.L .LabelKey}}</a>
    {{end}}
    {{end}}
  </nav>

  <!-- M1's button pair. Two actions, not two more nav links: signing in and
       seeing the menu are what a visitor does, and the artboard gives them the
       weight of buttons. The cta-bar class is sized for the masthead fill,
       where beige is 3.93 and everything must be large text (docs/10 §2.7).
       NOTE: no backticks anywhere in this constant — it is a Go raw string
       literal, and one backtick in a comment ends it. -->
  <div class="masthead-actions">
    <a class="cta-bar cta-bar-ghost" href="/app/login">{{t .L "nav.sign_in"}}</a>
    <a class="cta-bar" href="{{path .L "/menu/healthy"}}">{{t .L "nav.see_menu"}}</a>
  </div>

  <details class="burger">
    <summary aria-label="{{t .L "nav.main"}}">
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path d="M3 6h18M3 12h18M3 18h18" fill="none" stroke="currentColor"
              stroke-width="2" stroke-linecap="round"/>
      </svg>
    </summary>
    <nav class="nav-drawer" aria-label="{{t .L "nav.main"}}">
    <!-- Data-driven (migration 0026): which items appear and in what order is
         configuration, not markup. CATEGORY is the diet-type dropdown; the
         rest are plain links. Labels come from the message catalogue by
         label_key, because a label typed into an admin box would exist in one
         language only. -->
    {{range .Nav}}
    {{if eq .Kind "CATEGORY"}}
    <details class="navdrop"{{if eq $.Active .ActiveKey}} data-current="true"{{end}}>
      <summary>{{t $.L .LabelKey}}
        <svg class="langpick-caret" viewBox="0 0 10 6" aria-hidden="true" focusable="false">
          <path d="M1 1l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.6"
                stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </summary>
      <ul class="navdrop-menu">
        {{range $.DietTypes}}
        <li><a href="{{path $.L (printf "/menu/%s" .Slug)}}">{{.Name}}</a></li>
        {{end}}
      </ul>
    </details>
    {{else}}
    <a href="{{if .IsLocalised}}{{path $.L .Path}}{{else}}{{.Path}}{{end}}"{{if eq $.Active .ActiveKey}} aria-current="page"{{end}}>
      {{if eq .Icon "cart"}}
      <!-- The icon is decoration beside a real label, so it is aria-hidden:
           a screen reader reads "Pesan", not "cart Pesan". -->
      <svg class="nav-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path d="M2 3h3l2.6 11.2a2 2 0 0 0 2 1.6h7.7a2 2 0 0 0 2-1.5L21 7H6.2"
              fill="none" stroke="currentColor" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round"/>
        <circle cx="10" cy="20" r="1.6"/><circle cx="18" cy="20" r="1.6"/>
      </svg>
      {{end}}{{t $.L .LabelKey}}</a>
    {{end}}
    {{end}}
    </nav>
  </details>
  {{template "langpicker" .}}
</header>
{{end}}

<!-- Language selector.
     <details> rather than a scripted menu: the CSP has no 'unsafe-inline', the
     public pages ship no JavaScript at all, and a disclosure widget is
     keyboard-operable and screen-reader-announced for free. Closed it is the
     current language's FLAG only (Steven, 2026-08-18) — the name took too much
     of the bar. The control keeps its accessible name through aria-label, so
     what a screen reader announces is unchanged; it is the sighted label that
     is gone.
     Each option in the open menu still carries its endonym: someone who cannot
     read Indonesian must still be able to find "English" and "中文" in a list,
     and a column of flags alone would not give them that.
     Flags are aria-hidden decoration. A flag is a country, not a language, so
     the name is what actually labels each option. -->
{{define "langpicker"}}
{{$current := .L}}
<details class="langpick">
  <summary aria-label="{{t .L "lang.choose"}}">
    {{range .Languages}}{{if .Active}}{{.Flag}}{{end}}{{end}}
    <svg class="langpick-caret" viewBox="0 0 10 6" aria-hidden="true" focusable="false">
      <path d="M1 1l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.6"
            stroke-linecap="round" stroke-linejoin="round"/>
    </svg>
  </summary>
  <ul class="langpick-menu">
    {{range .Languages}}
    <li>
      <a href="{{.URL}}" hreflang="{{.Info.Tag}}" lang="{{.Info.Tag}}"
         {{if .Active}}aria-current="true"{{end}}>
        {{.Flag}}<span>{{.Info.Endonym}}</span>
        {{if .Active}}<svg class="langpick-tick" viewBox="0 0 12 12" aria-hidden="true" focusable="false">
          <path d="M2 6.5l2.8 2.8L10 3.5" fill="none" stroke="currentColor" stroke-width="1.8"
                stroke-linecap="round" stroke-linejoin="round"/>
        </svg>{{end}}
      </a>
    </li>
    {{end}}
  </ul>
</details>
{{end}}

{{define "foot"}}
<!-- Floating WhatsApp contact. In "foot" rather than on the home page alone,
     so it is on every public page (Steven, 2026-08-18). A plain anchor, not a
     script: the CSP has no 'unsafe-inline', so an inline handler would
     silently not run. rel=noopener because target=_blank without it hands the
     opened tab a window.opener reference back into this page. -->
{{$wa := waNumber .Company.whatsapp}}
{{if $wa}}
<a class="wa-float" href="https://wa.me/{{$wa}}" target="_blank" rel="noopener noreferrer"
   aria-label="{{t .L "wa.aria"}}">
  <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <path fill="currentColor" d="M12.04 2C6.58 2 2.13 6.45 2.13 11.91c0 1.75.46 3.45 1.32 4.95L2 22l5.25-1.38a9.9 9.9 0 0 0 4.79 1.22h.01c5.46 0 9.91-4.45 9.91-9.91C21.96 6.45 17.5 2 12.04 2zm0 18.15h-.01a8.2 8.2 0 0 1-4.19-1.15l-.3-.18-3.12.82.83-3.04-.2-.31a8.19 8.19 0 0 1-1.26-4.38c0-4.54 3.7-8.23 8.25-8.23 2.2 0 4.27.86 5.83 2.42a8.18 8.18 0 0 1 2.41 5.82c0 4.54-3.7 8.23-8.24 8.23zm4.52-6.16c-.25-.12-1.47-.72-1.69-.81-.23-.08-.39-.12-.56.13-.16.24-.64.8-.78.97-.14.16-.29.18-.54.06-.25-.13-1.05-.39-1.99-1.23-.74-.66-1.24-1.47-1.38-1.72-.15-.25-.02-.38.11-.5.11-.11.25-.29.37-.43.13-.15.17-.25.25-.41.08-.17.04-.31-.02-.43-.06-.12-.56-1.35-.77-1.84-.2-.49-.4-.42-.56-.43h-.47c-.17 0-.43.06-.66.31-.23.25-.86.85-.86 2.07 0 1.22.89 2.4 1.01 2.56.12.17 1.75 2.67 4.23 3.74.59.26 1.05.41 1.41.52.59.19 1.13.16 1.56.1.48-.07 1.47-.6 1.68-1.18.21-.58.21-1.07.14-1.18-.06-.11-.22-.17-.47-.29z"/>
  </svg>
</a>
{{end}}
<footer class="foot">
  <p>{{.Company.name}}{{if .Company.email}} · {{.Company.email}}{{end}}{{if .Company.phone}} · {{.Company.phone}}{{end}}</p>
  <span class="sep" aria-hidden="true">·</span>
  <p class="small">&copy; {{.Year}} Evermore</p>
</footer>
</body>
</html>
{{end}}

{{define "home"}}
{{template "head" .}}
<main>
  <section class="hero hero-split">
    <div class="hero-copy">
      <!-- M1's service-area badge. Ember on the ground is 7.27, and it says
           WHERE we deliver, which is the first question a visitor has. -->
      <p class="area-badge">{{c .Copy .L "home.area_badge"}}</p>
      <p class="eyebrow">{{c .Copy .L "home.eyebrow"}}</p>
      <h1>{{c .Copy .L "home.h1"}}</h1>
      <p class="lede">{{c .Copy .L "home.lede"}}</p>
      <p class="hero-cta">
        <a class="cta" href="{{path .L "/menu/healthy"}}">{{c .Copy .L "home.cta"}}</a>
        <a class="cta cta-ghost" href="{{path .L "/contact"}}">{{t .L "home.cta_area"}}</a>
      </p>
      <!-- The two figures the artboard states. Both are COUNTED — the diet
           types from the table, the kitchens from the active rows — so neither
           can go stale in copy the way a written "3 dapur" would. -->
      <dl class="hero-stats">
        <div>
          <dt>{{len .DietTypes}}</dt>
          <dd>{{t .L "home.stat_diets"}}</dd>
        </div>
        {{if .Kitchens}}
        <div>
          <dt>{{.Kitchens}}</dt>
          <dd>{{t .L "home.stat_kitchens"}}</dd>
        </div>
        {{end}}
      </dl>
    </div>
    {{if .HeroImage}}
    <!-- The picture is a sys_parameters row, so it can be swapped without a
         deploy. width/height are MEASURED from the file (heroSize) rather than
         hard-coded: they were 800x800 against a 800x533 photograph, so the
         browser reserved a square and everything below the hero jumped when it
         loaded. Eager and fetch-priority high: it is the largest element on
         the first screen, so it is the LCP, and lazy-loading it would delay
         the very metric it defines. -->
    <div class="hero-art">
      <img src="{{.HeroImage}}" alt="{{t .L "home.hero_alt"}}"
           {{if and .HeroW .HeroH}}width="{{.HeroW}}" height="{{.HeroH}}"{{end}}
           fetchpriority="high" decoding="async">
      <!-- M1's "Antar hari ini" card. It sits on the deep ground rather than
           on the photograph: text over an unpredictable image has no contrast
           anyone can calculate, and this card carries the cut-off — the one
           fact on the page a customer is held to. -->
      <div class="deliver-card">
        <span class="kicker">{{t .L "home.deliver_today"}}</span>
        <strong>{{t .L "home.deliver_time"}}</strong>
        <span class="fine">{{t .L "home.deliver_cutoff"}}</span>
      </div>
    </div>
    {{end}}
  </section>

  {{if .Meals}}
  <!-- M1's menu band: this week's published meals on the mid-green bar fill,
       with the package card closing the row. The band is a SURFACE, not a bar,
       so the 19px bar rule does not apply to it — but beige on #468973 is only
       3.93, so every card inside it stands on the deep ground again rather
       than sitting straight on the green (docs/10 §2.7). -->
  <section class="menu-band">
    <div class="section-head band-head">
      <div>
        <h2>{{t .L "home.menu_h2"}}</h2>
        <p>{{t .L "home.menu_sub"}}</p>
      </div>
      <!-- The diet chips M1 puts on the right of the band head. The
           "Antar hari ini" card is NOT repeated here: it is already on the
           hero artwork, and two of them on one screen reads as a bug. -->
      <div class="band-diets">
        {{range .DietTypes}}
        <a href="{{path $.L (printf "/menu/%s" .Slug)}}">{{.Name}}</a>
        {{end}}
      </div>
    </div>
    <div class="band-grid">
      {{range .Meals}}
      <article class="card meal band-meal">
        <p class="when">{{.Slot}} · {{.DietType}}</p>
        <h3>{{if .Name}}{{.Name}}{{else}}{{index .Items 0}}{{end}}</h3>
        <ul class="items">
          {{range .Items}}<li>{{.}}</li>{{end}}
        </ul>
        <p class="badges">
          <span class="badge">{{.Kcal}} kkal</span>
          <span class="badge">{{.ProteinG}} protein</span>
          {{if not .Complete}}<span class="badge est">≈</span>{{end}}
        </p>
      </article>
      {{end}}
      <!-- The package card. Same row, different job: it is the one card in the
           band that is an offer rather than a dish. -->
      <article class="card pkg-card">
        <p class="kicker kicker-info">{{t .L "home.pkg_kicker"}}</p>
        <h3>{{t .L "price.packages_h2"}}</h3>
        <p>{{t .L "home.pkg_body"}}</p>
        <a class="cta cta-ghost" href="{{path .L "/price-list"}}">{{t .L "home.pkg_cta"}}</a>
      </article>
    </div>
  </section>
  {{end}}

  {{if and .Prices .Prices.Packages}}
  <!-- Price table on the home page (Steven, 2026-08-19), the same partial the
       price-list page renders — one table, two placements, so the two can
       never disagree about what a package costs. -->
  <section class="home-prices">
    <div class="section-head"><h2>{{t .L "price.packages_h2"}}</h2></div>
    {{template "packagetable" .}}
  </section>
  {{end}}

  <!-- Why Evermore, below the prices. Same editable content as the Benefits
       page; rich text, so chtml rather than c. -->
  <section class="check">
    <div class="panel">
      <h2>{{c .Copy .L "benefit.title"}}</h2>
      <div class="richtext">{{chtml .Copy .L "benefit.body"}}</div>
    </div>
  </section>

  <section class="diets">
    <div class="section-head"><h2>{{t .L "home.diets_h2"}}</h2></div>
    <div class="grid">
      {{range .DietTypes}}
      <article class="card card-diet">
        <!-- Contextual corner mark. Decoration only: aria-hidden, and it says
             nothing the heading does not. currentColor so it tints from the
             card's ink token rather than carrying a colour of its own. -->
        <svg class="card-art" viewBox="0 0 64 64" aria-hidden="true" focusable="false">
          {{dietArt .Slug}}
        </svg>
        <h3><a href="{{path $.L (printf "/menu/%s" .Slug)}}">{{.Name}}</a></h3>
        <p>{{.Description}}</p>
      </article>
      {{end}}
    </div>
  </section>

  <!-- Body copy, so it sits on a beige panel: deep ink there is 11.32:1,
       against 2.88 straight on the green ground. -->
  <section class="check">
    <div class="panel">
      <h2>{{t .L "home.check_h2"}}</h2>
      <p>{{t .L "home.check_body"}}</p>
    </div>
  </section>

  {{if .Certs}}
  <!-- Certification badges, as images. Each path is a sys_parameter, so the
       certifying body's own logo file replaces the default seal without a
       deploy — those marks are trademarks and arrive from the issuer.
       The alt text is the standard's name: the badge IS the information, so it
       is not decorative and must not be alt="". The caption below each is
       editable content, which is where the certificate number goes. -->
  <section class="certs">
    <h2>{{c .Copy .L "cert.heading"}}</h2>
    <ul class="cert-badges">
      {{range .Certs}}
      <li>
        <img class="cert-badge" src="{{.Image}}" alt="{{.Label}}" loading="lazy"
             {{if and .W .H}}width="{{.W}}" height="{{.H}}"{{end}}>
        <span class="cert-note">{{c $.Copy $.L .Key}}</span>
      </li>
      {{end}}
    </ul>
  </section>
  {{end}}
</main>
{{template "foot" .}}
{{end}}

{{define "menu"}}
{{template "head" .}}
<main>
  <section class="hero compact">
    <p class="eyebrow">{{t .L "menu.eyebrow"}}</p>
    <h1>{{.Diet.Name}}</h1>
    <p class="lede">{{.Diet.Description}}</p>
  </section>

  <section class="meals">
    <div class="section-head"><h2>{{t .L "menu.h2"}}</h2></div>
    {{if .Meals}}
    <div class="grid">
      {{range .Meals}}
      <article class="card meal">
        <!-- The meal's picture. A real photograph once one is uploaded
             (scheduled_meal.hero_photo_key); until then an illustrated band in
             the diet type's colour, drawn from the SAME glyph source as the
             corner marks on the home cards so the two cannot drift. The
             fallback is decoration and is aria-hidden; a real photo carries
             the dish name as its alt. -->
        <div class="meal-art meal-art-{{$.Diet.Slug}}">
          {{if .PhotoKey}}
          <img src="{{.PhotoKey}}" alt="{{.Name}}" loading="lazy" decoding="async">
          {{else}}
          <svg viewBox="0 0 64 64" aria-hidden="true" focusable="false">{{dietArt $.Diet.Slug}}</svg>
          {{end}}
        </div>
        <p class="when">{{.Date}} · {{.Slot}}</p>
        <h3>{{if .Name}}{{.Name}}{{else}}{{index .Items 0}}{{end}}</h3>
        <ul class="items">{{range .Items}}<li>{{.}}</li>{{end}}</ul>
        <p class="badges">
          <span class="badge">{{.Kcal}} {{t $.L "menu.kcal"}}</span>
          <span class="badge">{{.ProteinG}} {{t $.L "menu.protein"}}</span>
          {{if not .Complete}}<span class="badge est">{{t $.L "menu.estimated"}}</span>{{end}}
        </p>
      </article>
      {{end}}
    </div>
    {{else}}
    <div class="panel">
      <p class="empty">{{t .L "menu.empty"}}</p>
    </div>
    {{end}}
  </section>
</main>
{{template "foot" .}}
{{end}}

{{define "notfound"}}
{{template "head" .}}
<main>
  <section class="hero">
    <h1>{{t .L "notfound.h1"}}</h1>
    <p class="lede">{{t .L "notfound.body"}}</p>
    <a class="cta" href="{{path .L "/"}}">{{t .L "notfound.cta"}}</a>
  </section>
</main>
{{template "foot" .}}
{{end}}

{{define "packagetable"}}
    <!-- A table, not cards (Steven, 2026-08-19). Prices are read by comparing
         DOWN a column — the whole point is to see which package costs what per
         day — and cards force the eye to jump between boxes to do that. -->
    <div class="table-scroll">
      <table class="pricetable">
        <caption class="sr-only">{{t .L "price.packages_h2"}}</caption>
        <thead>
          <tr>
            <th scope="col">{{t .L "price.col_package"}}</th>
            <th scope="col" class="num">{{t .L "price.col_days"}}</th>
            <th scope="col" class="num">{{t .L "price.col_amount"}}</th>
            <th scope="col" class="num">{{t .L "price.col_per_day"}}</th>
          </tr>
        </thead>
        <tbody>
          {{range .Prices.Packages}}
          <tr>
            <th scope="row">
              {{.Name}}
              {{if .Description}}<span class="row-note">{{.Description}}</span>{{end}}
            </th>
            <td class="num">{{.MealCredits}}</td>
            <td class="num">
              {{if .PriceIDR}}{{idr .PriceIDR}}{{else}}{{t $.L "price.on_request"}}{{end}}
            </td>
            <!-- Per-day is the number a customer is actually comparing, and
                 working it out in the head across three rows is exactly the
                 friction a price table exists to remove. Integer division on
                 whole rupiah — no floats anywhere near money (CLAUDE.md §4). -->
            <td class="num">
              {{if and .PriceIDR .MealCredits}}{{idr (perDay .PriceIDR .MealCredits)}}{{else}}—{{end}}
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </div>
{{end}}

{{define "pricelist"}}
{{template "head" .}}
<main>
  <section class="hero compact">
    <p class="eyebrow">{{t .L "nav.pricelist"}}</p>
    <h1>{{t .L "price.h1"}}</h1>
    <p class="lede">{{if .ShowMealPrices}}{{t .L "price.lede"}}{{else}}{{t .L "price.lede_quote"}}{{end}}</p>
  </section>

  {{if and .Prices .Prices.Packages}}
  <section class="check">
    <div class="section-head"><h2>{{t .L "price.packages_h2"}}</h2></div>
    {{template "packagetable" .}}
  </section>
  {{end}}
  {{if .ShowMealPrices}}
  <section class="panel">
    <h2>{{t .L "price.meals_h2"}}</h2>
    <p>{{t .L "price.meals_note"}}</p>
    {{if and .Prices .Prices.Prices}}
    <div class="table-scroll">
      <table class="pricetable">
        <caption class="sr-only">{{t .L "price.meals_h2"}}</caption>
        <thead>
          <tr>
            <th scope="col">{{t .L "price.col_category"}}</th>
            <th scope="col">{{t .L "price.col_tier"}}</th>
            <th scope="col" class="num">{{t .L "price.col_price"}}</th>
          </tr>
        </thead>
        <tbody>
          {{range .Prices.Prices}}
          <tr>
            <th scope="row">{{.DietName}}</th>
            <td>{{.TierLabel}}</td>
            <td class="num">{{idr .UnitPriceIDR}}</td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </div>
    {{else}}
    <p class="empty">{{t .L "price.empty"}}</p>
    {{end}}

    {{if and .Prices .Prices.Tiers}}
    <h3>{{t .L "price.tiers_h3"}}</h3>
    <ul class="items">
      {{range .Prices.Tiers}}<li>{{.Label}}</li>{{end}}
    </ul>
    {{end}}
  </section>
  {{else}}
  <!-- Per-portion prices are switched off (public.show_meal_prices). The page
       still has to answer "what does it cost" with something, so it points at
       the quote route rather than silently dropping the section and leaving a
       price page with no prices on it. -->
  <section class="panel">
    <h2>{{t .L "price.quote_h2"}}</h2>
    <p>{{t .L "price.quote_body"}}</p>
    <p><a class="cta cta-on-sheet" href="{{path .L "/contact"}}">{{t .L "price.quote_cta"}}</a></p>
  </section>
  {{end}}

  <!-- Benefit. After the price list (Steven, 2026-08-18), editable in the back
       office, and the body is rich text — hence chtml rather than c. -->
  <section class="check">
    <div class="panel">
      <h2>{{c .Copy .L "benefit.title"}}</h2>
      <div class="richtext">{{chtml .Copy .L "benefit.body"}}</div>
    </div>
  </section>

</main>
{{template "foot" .}}
{{end}}

{{define "contact"}}
{{template "head" .}}
<main>
  <section class="hero compact">
    <p class="eyebrow">{{t .L "nav.contact"}}</p>
    <h1>{{t .L "contact.h1"}}</h1>
    <p class="lede">{{c .Copy .L "contact.lede"}}</p>
  </section>
  <section class="panel">
    <h2>{{t .L "contact.reach_h2"}}</h2>
    <ul class="contact-list">
      <li><strong>{{.Company.name}}</strong></li>
      {{if .Company.email}}
      <li>{{t .L "contact.email"}}:
        <a href="mailto:{{.Company.email}}">{{.Company.email}}</a></li>
      {{end}}
      {{if .Company.phone}}
      <li>{{t .L "contact.phone"}}:
        <a href="tel:{{.Company.phone}}">{{.Company.phone}}</a></li>
      {{end}}
      {{$wa := waNumber .Company.whatsapp}}
      {{if $wa}}
      <li>WhatsApp:
        <a href="https://wa.me/{{$wa}}" target="_blank" rel="noopener noreferrer">{{.Company.phone}}</a></li>
      {{end}}
    </ul>
    <p>{{t .L "contact.hours"}}</p>
  </section>
</main>
{{template "foot" .}}
{{end}}

{{define "about"}}
{{template "head" .}}
<main>
  <section class="hero compact">
    <p class="eyebrow">{{t .L "nav.about"}}</p>
    <h1>{{t .L "about.h1"}}</h1>
    <p class="lede">{{c .Copy .L "about.lede"}}</p>
  </section>
  <section class="panel">
    <p>{{c .Copy .L "about.body"}}</p>
  </section>
</main>
{{template "foot" .}}
{{end}}

{{define "career"}}
{{template "head" .}}
<main>
  <section class="hero compact">
    <p class="eyebrow">{{t .L "nav.career"}}</p>
    <h1>{{t .L "career.h1"}}</h1>
    <p class="lede">{{c .Copy .L "career.lede"}}</p>
  </section>

  <!-- What is open right now, first. Someone on this page wants to know
       whether there is a job before they read anything else. Rows come from
       job_opening, edited in the back office. -->
  <section class="panel">
    <h2>{{t .L "career.openings_h2"}}</h2>
    {{if .Openings}}
    <ul class="openings">
      {{range .Openings}}
      <li>
        <strong>{{.Title}}</strong>
        {{if .Summary}}<span class="opening-summary">{{.Summary}}</span>{{end}}
      </li>
      {{end}}
    </ul>
    {{else}}
    <p class="empty">{{t .L "career.no_openings"}}</p>
    {{end}}
    <p>{{c .Copy .L "career.body"}}</p>
  </section>

  <section class="check">
    <div class="panel">
      <h2>{{t .L "career.form_h2"}}</h2>

      {{if .Applied}}
      <p class="notice-ok" role="status">{{t .L "career.thanks"}}</p>
      {{else}}

      {{if .FormErrors._}}
      <p class="error" role="alert">
        {{if eq .FormErrors._ "toomany"}}{{t .L "career.err_toomany"}}
        {{else if eq .FormErrors._ "toolarge"}}{{t .L "career.err_toolarge"}}
        {{else if eq .FormErrors._ "unsupported"}}{{t .L "career.err_unsupported"}}
        {{else}}{{t .L "career.err_failed"}}{{end}}
      </p>
      {{end}}

      <!-- No enctype and no file input, on purpose: this form is
           urlencoded-only and the server refuses anything else. A CV is
           emailed after we reply (see the note under the button). -->
      <form method="post" action="{{path .L "/career"}}" novalidate class="jobform">
        <div class="field-row">
          <label class="label" for="full_name">{{t .L "career.f_name"}}</label>
          <input class="field" id="full_name" name="full_name" maxlength="120" required
                 value="{{index .Form "full_name"}}"
                 {{if .FormErrors.full_name}}aria-invalid="true" aria-describedby="e-name"{{end}}>
          {{if .FormErrors.full_name}}<p class="error" id="e-name">{{t .L "career.e_required"}}</p>{{end}}
        </div>

        <div class="field-row">
          <label class="label" for="email">{{t .L "career.f_email"}}</label>
          <input class="field" id="email" name="email" type="email" maxlength="254" required
                 value="{{index .Form "email"}}"
                 {{if .FormErrors.email}}aria-invalid="true" aria-describedby="e-email"{{end}}>
          {{if .FormErrors.email}}<p class="error" id="e-email">{{t .L "career.e_email"}}</p>{{end}}
        </div>

        <div class="field-row">
          <label class="label" for="phone">{{t .L "career.f_phone"}}</label>
          <input class="field" id="phone" name="phone" inputmode="tel" maxlength="30"
                 value="{{index .Form "phone"}}"
                 {{if .FormErrors.phone}}aria-invalid="true" aria-describedby="e-phone"{{end}}>
          {{if .FormErrors.phone}}<p class="error" id="e-phone">{{t .L "career.e_phone"}}</p>{{end}}
        </div>

        <div class="field-row">
          <label class="label" for="position">{{t .L "career.f_position"}}</label>
          <select class="field" id="position" name="position" required
                  {{if .FormErrors.position}}aria-invalid="true" aria-describedby="e-pos"{{end}}>
            <option value="">{{t .L "career.f_position_choose"}}</option>
            {{$sel := index .Form "position"}}
            {{range .Openings}}
            <option value="{{.Slug}}"{{if eq $sel .Slug}} selected{{end}}>{{.Title}}</option>
            {{end}}
          </select>
          {{if .FormErrors.position}}<p class="error" id="e-pos">{{t .L "career.e_position"}}</p>{{end}}
        </div>

        <div class="field-row">
          <label class="label" for="message">{{t .L "career.f_message"}}</label>
          <textarea class="field" id="message" name="message" rows="6" maxlength="4000" required
                    {{if .FormErrors.message}}aria-invalid="true" aria-describedby="e-msg"{{end}}>{{index .Form "message"}}</textarea>
          {{if .FormErrors.message}}<p class="error" id="e-msg">{{t .L "career.e_required"}}</p>{{end}}
        </div>

        <button class="cta cta-on-sheet" type="submit">{{t .L "career.submit"}}</button>
        <p class="form-note">{{t .L "career.no_file_note"}}</p>
      </form>
      {{end}}
    </div>
  </section>
</main>
{{template "foot" .}}
{{end}}

{{define "benefits"}}
{{template "head" .}}
<main>
  <section class="hero compact">
    <p class="eyebrow">{{t .L "nav.benefits"}}</p>
    <h1>{{c .Copy .L "benefit.title"}}</h1>
  </section>
  <section class="panel">
    <div class="richtext">{{chtml .Copy .L "benefit.body"}}</div>
  </section>
</main>
{{template "foot" .}}
{{end}}
`
