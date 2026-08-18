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
<link rel="stylesheet" href="/fonts/fonts.css">
<link rel="stylesheet" href="/css/tokens.css">
<link rel="stylesheet" href="/css/public.css">
<link rel="preload" href="/fonts/erode/Erode-Bold.woff2" as="font" type="font/woff2" crossorigin>
<link rel="preload" href="/fonts/inter/InterVariable.woff2" as="font" type="font/woff2" crossorigin>
{{if .JSONLD}}<script type="application/ld+json">{{.JSONLD}}</script>{{end}}
</head>
<body>
<header class="masthead">
  <!-- The supplied wordmark, reversed out for the dark fill. width/height are
       the intrinsic pixels so the masthead does not reflow as it loads; the
       stylesheet constrains the displayed height. -->
  <a class="wordmark" href="{{path .L "/"}}" aria-label="{{t .L "nav.home_aria"}}">
    <img src="/images/evermore-wordmark-light.png" width="560" height="60"
         alt="Evermore">
  </a>
  <nav aria-label="{{t .L "nav.main"}}">
    {{range .DietTypes}}<a href="{{path $.L (printf "/menu/%s" .Slug)}}">{{.Name}}</a>{{end}}
  </nav>
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
      <p class="eyebrow">{{c .Copy .L "home.eyebrow"}}</p>
      <h1>{{c .Copy .L "home.h1"}}</h1>
      <p class="lede">{{c .Copy .L "home.lede"}}</p>
      <a class="cta" href="{{path .L "/menu/healthy"}}">{{c .Copy .L "home.cta"}}</a>
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
    </div>
    {{end}}
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
`
