// Motion guard for the public home page.
//
// It asserts the one rule that is not taste (docs/10 §2.9, design.md §4):
// MOTION IS NEVER THE REASON CONTENT IS VISIBLE. For every animated element it
// checks that the settled opacity is 1 under normal motion AND under
// prefers-reduced-motion, that reduced motion carries no animation at all, and
// that at least one animation is actually applied — so the day the stylesheet
// stops matching, this says so instead of the page quietly going blank.
//
// It needs a browser and the running service, so it cannot live in `go test`.
//
//   export NODE_PATH=/home/dev/.npm/_npx/e41f203b7505f1fb/node_modules
//   node scripts/verify-motion.js [url]
//
// Use the 1.62.1 npx cache; the 1.49.1 one hangs at launch. See
// docs/RUN-WHEN-BACK.md §A3. Exits non-zero on any failure.

const { chromium } = require('playwright');
const URL = process.argv[2] || 'http://127.0.0.1:8090/';

async function probe(browser, reduced) {
  const ctx = await browser.newContext({
    viewport: { width: 390, height: 844 },
    reducedMotion: reduced ? 'reduce' : 'no-preference',
  });
  const page = await ctx.newPage();
  await page.goto(URL, { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForTimeout(1600); // well past the longest 180ms delay + 400ms
  const out = await page.evaluate(() => {
    const info = (sel) => {
      const e = document.querySelector(sel);
      if (!e) return { sel, missing: true };
      const cs = getComputedStyle(e);
      const b = e.getBoundingClientRect();
      return {
        sel,
        animationName: cs.animationName,
        animationDelay: cs.animationDelay,
        opacity: +cs.opacity,
        transform: cs.transform,
        visibleHeight: +b.height.toFixed(1),
      };
    };
    return {
      docScrollW: document.documentElement.scrollWidth,
      viewport: window.innerWidth,
      supportsViewTimeline: CSS.supports('animation-timeline: view()'),
      items: ['.hero-copy .eyebrow', '.hero-copy h1', '.hero-copy .lede', '.hero-copy .cta',
              '.hero-art img', '.home-prices', '.check', '.diets .card-diet'].map(info),
    };
  });
  await ctx.close();
  return out;
}

(async () => {
  const browser = await chromium.launch();
  const normal = await probe(browser, false);
  const reduced = await probe(browser, true);
  await browser.close();

  console.log('supports view():', normal.supportsViewTimeline);
  console.log('scrollWidth normal/reduced:', normal.docScrollW, '/', reduced.docScrollW, '(viewport', normal.viewport + ')');
  const pad = (v, n) => String(v).padEnd(n);
  console.log('\n' + pad('element', 24) + '| ' + pad('animation (normal)', 22) + '| ' + pad('delay', 8) + '| settled opacity N/R');
  let bad = [];
  normal.items.forEach((n, i) => {
    const r = reduced.items[i];
    console.log(pad(n.sel.replace('.hero-copy ', '').replace('.diets ', ''), 24) + '| ' +
      pad(n.animationName, 22) + '| ' + pad(n.animationDelay, 8) + '| ' + n.opacity + ' / ' + r.opacity);
    if (n.opacity !== 1) bad.push(`${n.sel}: opacity ${n.opacity} after settle (normal motion)`);
    if (r.opacity !== 1) bad.push(`${r.sel}: opacity ${r.opacity} under REDUCED motion — content hidden`);
    if (r.animationName !== 'none') bad.push(`${r.sel}: animation "${r.animationName}" still set under reduced motion`);
    if (r.visibleHeight === 0) bad.push(`${r.sel}: zero height under reduced motion`);
  });
  const applied = normal.items.filter(x => x.animationName && x.animationName !== 'none').length;
  console.log('\nanimations applied under normal motion:', applied, '/', normal.items.length);
  if (applied === 0) bad.push('NO animation applied at all — the stylesheet change did nothing');
  if (normal.docScrollW > normal.viewport) console.log('NOTE pre-existing overflow:', normal.docScrollW, '>', normal.viewport);
  console.log(bad.length ? '\nFAILURES:\n  ' + bad.join('\n  ') : '\nAll assertions passed.');
  process.exit(bad.length ? 1 : 0);
})();
