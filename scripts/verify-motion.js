// Motion guard for the public home page.
//
// It asserts the rule that is not taste (docs/10 §2.9, design.md §4):
// MOTION IS NEVER THE REASON CONTENT IS VISIBLE — under prefers-reduced-motion
// every animated element must still be present, opaque and non-zero height,
// with no animation attached.
//
// It also asserts the four things that silently do nothing when they are one
// word wrong, each of which shipped here before this script existed:
//
//   * the hero parallax must CHANGE with scroll   (overflow:hidden on the frame
//     makes it a scroll container and freezes the view() timeline; use clip)
//   * the card hover lift must survive the reveal (the `transform` shorthand
//     lets the animation's fill state erase it; translate/scale compose)
//   * the section rule must have real dimensions
//   * every price row must carry its reveal
//
// Needs a browser and the running service, so it cannot live in `go test`.
//
//   export NODE_PATH=/home/dev/.npm/_npx/e41f203b7505f1fb/node_modules
//   node scripts/verify-motion.js [url]
//
// Use the 1.62.1 npx cache; 1.49.1 hangs at launch. docs/RUN-WHEN-BACK.md §A3.
// Exits non-zero on any failure.

const { chromium } = require('playwright');
const URL = process.argv[2] || 'http://127.0.0.1:8090/';
(async () => {
  const b = await chromium.launch();
  const fails = [];
  const ctx = await b.newContext({ viewport: { width: 390, height: 844 } });
  const p = await ctx.newPage();
  await p.goto(URL, { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(1400);

  // 1. Parallax: the hero image's translate must actually change with scroll.
  const pan = await p.evaluate(async () => {
    const img = document.querySelector('.hero-art img');
    const at = [];
    for (const y of [0, 150, 300, 450]) {
      window.scrollTo(0, y);
      await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)));
      at.push({ y, translate: getComputedStyle(img).translate, scale: getComputedStyle(img).scale });
    }
    window.scrollTo(0, 0);
    return at;
  });
  console.log('hero parallax:'); pan.forEach(r => console.log('   scrollY ' + String(r.y).padStart(4) + '  translate=' + r.translate + '  scale=' + r.scale));
  if (new Set(pan.map(r => r.translate)).size < 2) fails.push('hero parallax: translate never changes with scroll');

  // 2. Hover lift must survive the scroll animation's fill state.
  const card = p.locator('.diets .card-diet').first();
  await card.scrollIntoViewIfNeeded(); await p.waitForTimeout(400);
  const before = await card.evaluate(el => getComputedStyle(el).scale);
  await card.hover(); await p.waitForTimeout(350);
  const after = await card.evaluate(el => getComputedStyle(el).scale);
  console.log('\ncard scale  rest=' + before + '  hover=' + after);
  if (before === after) fails.push('hover lift does nothing — the reveal animation is overriding it');

  // 3. The section rule must be a real, visible element.
  const rule = await p.evaluate(() => {
    const h = document.querySelector('.diets .section-head h2');
    const cs = getComputedStyle(h, '::after');
    return { width: cs.width, height: cs.height, bg: cs.backgroundColor };
  });
  console.log('section rule ::after  ' + JSON.stringify(rule));
  if (parseFloat(rule.height) === 0) fails.push('section rule has no height');

  // 4. Price rows animate.
  const rows = await p.evaluate(() => [...document.querySelectorAll('.home-prices .pricetable tbody tr')]
      .map(tr => getComputedStyle(tr).animationName));
  console.log('price row animations: ' + JSON.stringify(rows));
  if (!rows.length) fails.push('no price rows found');
  else if (rows.some(n => n === 'none')) fails.push('some price rows carry no animation');
  await ctx.close();

  // 5. THE INVARIANT: reduced motion shows everything, animates nothing.
  const rctx = await b.newContext({ viewport: { width: 390, height: 844 }, reducedMotion: 'reduce' });
  const rp = await rctx.newPage();
  await rp.goto(URL, { waitUntil: 'domcontentloaded' });
  await rp.waitForTimeout(1200);
  const red = await rp.evaluate(() => ['.hero-copy .eyebrow','.hero-copy h1','.hero-copy .lede','.hero-copy .cta',
      '.hero-art','.hero-art img','.home-prices .pricetable tbody tr','.check .panel','.diets .card-diet']
      .map(sel => { const e = document.querySelector(sel); if (!e) return { sel, missing: true };
        const cs = getComputedStyle(e); const r = e.getBoundingClientRect();
        return { sel, opacity: +cs.opacity, anim: cs.animationName, h: Math.round(r.height), translate: cs.translate }; }));
  console.log('\nreduced motion:');
  red.forEach(r => { console.log('   ' + r.sel.padEnd(34) + ' opacity=' + r.opacity + ' anim=' + r.anim + ' h=' + r.h);
    if (r.missing) fails.push(r.sel + ' missing');
    else { if (r.opacity !== 1) fails.push(r.sel + ' opacity ' + r.opacity + ' under reduced motion');
           if (r.anim !== 'none') fails.push(r.sel + ' still animating under reduced motion');
           if (r.h === 0) fails.push(r.sel + ' zero height under reduced motion'); } });
  await rctx.close(); await b.close();
  console.log(fails.length ? '\nFAILURES:\n  ' + fails.join('\n  ') : '\nAll assertions passed.');
  process.exit(fails.length ? 1 : 0);
})();
