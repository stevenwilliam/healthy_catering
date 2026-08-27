// Guard for the produce background on the public pages.
//
// It asserts the rule that is not taste (docs/10 §2.8, design.md §5): the
// background may cost contrast, but never below AA for the token that binds.
// Public-page text sits directly on the ground, and the binding ink is
// --beige-deep #CCBDAA, NOT beige — the tile this replaced left beige at a
// perfectly good 7.29 while putting the muted token at 4.16, below AA, and
// nobody noticed because only beige was ever checked.
//
// It computes the ARITHMETIC BOUND rather than sampling a screenshot. A
// screenshot is one frame at one drift offset: an earlier pass measured 4.89
// and looked safe purely because no two strokes happened to cross in it. Over
// a 150-second drift across a whole viewport, the crossing happens. So this
// reads each layer's true peak alpha out of its own artwork, combines it with
// the opacity the stylesheet actually applies, and composites that.
//
//   export NODE_PATH=/home/dev/.npm/_npx/e41f203b7505f1fb/node_modules
//   node scripts/verify-background.js [url]
//
// Exits non-zero on any failure. docs/RUN-WHEN-BACK.md §A3 for the browser.

const { chromium } = require('playwright');
const URL = process.argv[2] || 'http://127.0.0.1:8090/';

const GROUND = [0x1c, 0x3d, 0x34];
const INKS = { beige: [0xff, 0xfa, 0xe0], white: [0xff, 0xff, 0xff], 'beige-deep': [0xcc, 0xbd, 0xaa] };
const AA = 4.5;

const chan = (v) => { v /= 255; return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; };
const lum = ([r, g, b]) => 0.2126 * chan(r) + 0.7152 * chan(g) + 0.0722 * chan(b);
const ratio = (a, b) => { const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x); return (hi + 0.05) / (lo + 0.05); };
const over = (fg, a, bg) => fg.map((c, i) => Math.round(c * a + bg[i] * (1 - a)));
const hex = (c) => '#' + c.map((v) => v.toString(16).padStart(2, '0').toUpperCase()).join('');

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 900, height: 700 } });
  await page.goto(URL, { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForTimeout(1200);
  const fails = [];

  const layers = await page.evaluate(async () => {
    const read = (pe) => {
      const cs = getComputedStyle(document.body, pe);
      const m = cs.backgroundImage.match(/url\(["']?([^"')]+)["']?\)/);
      return { url: m && m[1], opacity: parseFloat(cs.opacity), z: cs.zIndex, anim: cs.animationName };
    };
    // The artwork's own peak alpha, read from the file the page is really
    // using. Same-origin SVG, so the canvas is not tainted.
    const peakAlpha = (url) => new Promise((res, rej) => {
      const img = new Image();
      img.onload = () => {
        const c = document.createElement('canvas');
        c.width = img.naturalWidth; c.height = img.naturalHeight;
        const g = c.getContext('2d', { willReadFrequently: true });
        g.drawImage(img, 0, 0);
        const d = g.getImageData(0, 0, c.width, c.height).data;
        let max = 0;
        for (let i = 3; i < d.length; i += 4) if (d[i] > max) max = d[i];
        res(max / 255);
      };
      img.onerror = () => rej(new Error('could not load ' + url));
      img.src = url;
    });
    const out = {};
    for (const pe of ['::before', '::after']) {
      const l = read(pe);
      l.artworkPeak = l.url ? await peakAlpha(l.url) : null;
      out[pe] = l;
    }
    out.bodyImage = getComputedStyle(document.body).backgroundImage;
    return out;
  });

  let combined = 0;
  for (const pe of ['::before', '::after']) {
    const l = layers[pe];
    if (!l.url) { fails.push(`body${pe} has no background image — the layer is gone`); continue; }
    if (l.z !== '-1') fails.push(`body${pe} is at z-index ${l.z}, not -1 — it would paint over content`);
    const eff = l.artworkPeak * l.opacity;
    combined = 1 - (1 - combined) * (1 - eff);
    console.log(`body${pe.padEnd(9)} ${l.url.replace(/^.*\//, '').padEnd(20)} opacity ${l.opacity}  artwork peak ${l.artworkPeak.toFixed(3)}  effective ${eff.toFixed(4)}`);
  }
  if (layers.bodyImage !== 'none') fails.push('body still carries its own background image: ' + layers.bodyImage);

  const composite = over(INKS.beige, combined, GROUND);
  console.log(`\ncombined peak ${combined.toFixed(4)}  ->  ${hex(composite)} over ${hex(GROUND)}\n`);
  for (const [name, ink] of Object.entries(INKS)) {
    const r = ratio(ink, composite);
    const ok = r >= AA;
    console.log(`  ${name.padEnd(11)} ${r.toFixed(2)}  ${ok ? 'AA ok' : '*** BELOW AA ***'}`);
    if (!ok) fails.push(`${name} is ${r.toFixed(2)} against the background peak — AA needs ${AA}`);
  }

  // And the motion must stop when asked.
  const ctx = await browser.newContext({ reducedMotion: 'reduce' });
  const rp = await ctx.newPage();
  await rp.goto(URL, { waitUntil: 'domcontentloaded', timeout: 20000 });
  await rp.waitForTimeout(900);
  const red = await rp.evaluate(() => ['::before', '::after']
    .map((pe) => ({ pe, anim: getComputedStyle(document.body, pe).animationName,
                    img: getComputedStyle(document.body, pe).backgroundImage !== 'none' })));
  red.forEach((r) => {
    if (r.anim !== 'none') fails.push(`body${r.pe} still drifting under reduced motion`);
    if (!r.img) fails.push(`body${r.pe} lost its artwork under reduced motion`);
  });
  console.log('\nreduced motion: ' + red.map((r) => `body${r.pe} anim=${r.anim}`).join(', '));

  await ctx.close(); await browser.close();
  console.log(fails.length ? '\nFAILURES:\n  ' + fails.join('\n  ') : '\nAll assertions passed.');
  process.exit(fails.length ? 1 : 0);
})();
