package http

// publicTemplates are the server-rendered public pages.
//
// Deliberately small and dependency-free: every colour is a token from
// docs/10-design-system.md with a measured contrast ratio, the fonts are
// self-hosted, and there is no inline script — the CSP has no 'unsafe-inline'
// so an inline <script> would simply not run.
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

<!-- Open Graph, STATIC in the served HTML. Preview bots do not run
     JavaScript, so these have to be here rather than set by a script
     (99 §13). This is what makes a WhatsApp share show a card. -->
<meta property="og:type" content="website">
<meta property="og:site_name" content="Evermore">
<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Description}}">
<meta property="og:url" content="{{.Canonical}}">
<meta property="og:image" content="{{.OGImage}}">
<meta property="og:locale" content="id_ID">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{{.Title}}">
<meta name="twitter:description" content="{{.Description}}">
<meta name="twitter:image" content="{{.OGImage}}">

<link rel="stylesheet" href="/fonts/fonts.css">
<link rel="stylesheet" href="/css/tokens.css">
<link rel="stylesheet" href="/css/public.css">
<link rel="preload" href="/fonts/erode/Erode-Semibold.woff2" as="font" type="font/woff2" crossorigin>
<link rel="preload" href="/fonts/inter/InterVariable.woff2" as="font" type="font/woff2" crossorigin>
{{if .JSONLD}}<script type="application/ld+json">{{.JSONLD}}</script>{{end}}
</head>
<body>
<header class="masthead">
  <a class="wordmark" href="/">evermore</a>
  <nav aria-label="Menu utama">
    {{range .DietTypes}}<a href="/menu/{{.Slug}}">{{.Name}}</a>{{end}}
  </nav>
</header>
{{end}}

{{define "foot"}}
<footer class="foot">
  <p>{{.Company.name}}{{if .Company.email}} · {{.Company.email}}{{end}}{{if .Company.phone}} · {{.Company.phone}}{{end}}</p>
  <p class="small">&copy; {{.Year}} Evermore · Jakarta</p>
</footer>
</body>
</html>
{{end}}

{{define "home"}}
{{template "head" .}}
<main>
  <section class="hero">
    <h1>Makan sehat, setiap hari, tanpa repot.</h1>
    <p class="lede">{{.Description}}</p>
    <a class="cta" href="/menu/healthy">Lihat menu minggu ini</a>
  </section>

  <section class="diets">
    <h2>Pilih sesuai kebutuhan Anda</h2>
    <div class="grid">
      {{range .DietTypes}}
      <article class="card">
        <h3><a href="/menu/{{.Slug}}">{{.Name}}</a></h3>
        <p>{{.Description}}</p>
      </article>
      {{end}}
    </div>
  </section>

  <section class="check">
    <h2>Kami antar ke tempat Anda?</h2>
    <p>Masukkan titik lokasi Anda saat mendaftar — kami langsung memberi tahu
       dapur mana yang melayani, sebelum Anda memesan.</p>
  </section>
</main>
{{template "foot" .}}
{{end}}

{{define "menu"}}
{{template "head" .}}
<main>
  <section class="hero compact">
    <h1>{{.Diet.Name}}</h1>
    <p class="lede">{{.Diet.Description}}</p>
  </section>

  <section class="meals">
    <h2>Menu tujuh hari ke depan</h2>
    {{if .Meals}}
    <div class="grid">
      {{range .Meals}}
      <article class="card meal">
        <p class="when">{{.Date}} · {{.Slot}}</p>
        <h3>{{if .Name}}{{.Name}}{{else}}{{index .Items 0}}{{end}}</h3>
        <ul class="items">{{range .Items}}<li>{{.}}</li>{{end}}</ul>
        <p class="badges">
          <span class="badge">{{.Kcal}} kkal</span>
          <span class="badge">{{.ProteinG}} g protein</span>
          {{if not .Complete}}<span class="badge est">perkiraan</span>{{end}}
        </p>
      </article>
      {{end}}
    </div>
    {{else}}
    <p class="empty">Menu untuk minggu ini sedang disiapkan. Silakan cek kembali besok.</p>
    {{end}}
  </section>
</main>
{{template "foot" .}}
{{end}}

{{define "notfound"}}
<!doctype html>
<html lang="{{.Lang}}">
<head><meta charset="utf-8"><title>{{.Title}}</title>
<link rel="stylesheet" href="/css/tokens.css"><link rel="stylesheet" href="/css/public.css"></head>
<body><main class="hero"><h1>Halaman tidak ditemukan</h1>
<p><a href="/">Kembali ke beranda</a></p></main></body></html>
{{end}}
`
