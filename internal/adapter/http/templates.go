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
{{$wa := waNumber .Company.whatsapp}}
{{if $wa}}
<!-- Floating WhatsApp contact. A plain anchor, not a script: the CSP has no
     'unsafe-inline', so an inline handler would silently not run. rel=noopener
     because target=_blank without it hands the opened tab a window.opener
     reference back into this page. -->
<a class="wa-float" href="https://wa.me/{{$wa}}" target="_blank" rel="noopener noreferrer"
   aria-label="Hubungi kami di WhatsApp">
  <svg viewBox="0 0 24 24" width="28" height="28" aria-hidden="true" focusable="false">
    <path fill="currentColor" d="M12.04 2C6.58 2 2.13 6.45 2.13 11.91c0 1.75.46 3.45 1.32 4.95L2 22l5.25-1.38a9.9 9.9 0 0 0 4.79 1.22h.01c5.46 0 9.91-4.45 9.91-9.91C21.96 6.45 17.5 2 12.04 2zm0 18.15h-.01a8.2 8.2 0 0 1-4.19-1.15l-.3-.18-3.12.82.83-3.04-.2-.31a8.19 8.19 0 0 1-1.26-4.38c0-4.54 3.7-8.23 8.25-8.23 2.2 0 4.27.86 5.83 2.42a8.18 8.18 0 0 1 2.41 5.82c0 4.54-3.7 8.23-8.24 8.23zm4.52-6.16c-.25-.12-1.47-.72-1.69-.81-.23-.08-.39-.12-.56.13-.16.24-.64.8-.78.97-.14.16-.29.18-.54.06-.25-.13-1.05-.39-1.99-1.23-.74-.66-1.24-1.47-1.38-1.72-.15-.25-.02-.38.11-.5.11-.11.25-.29.37-.43.13-.15.17-.25.25-.41.08-.17.04-.31-.02-.43-.06-.12-.56-1.35-.77-1.84-.2-.49-.4-.42-.56-.43h-.47c-.17 0-.43.06-.66.31-.23.25-.86.85-.86 2.07 0 1.22.89 2.4 1.01 2.56.12.17 1.75 2.67 4.23 3.74.59.26 1.05.41 1.41.52.59.19 1.13.16 1.56.1.48-.07 1.47-.6 1.68-1.18.21-.58.21-1.07.14-1.18-.06-.11-.22-.17-.47-.29z"/>
  </svg>
</a>
{{end}}
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
