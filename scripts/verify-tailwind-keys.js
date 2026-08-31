// Guard against a Tailwind key collision that paints text an invisible colour.
//
// THE INCIDENT. `theme.extend.colors` had a key `bar` (#468973, the masthead
// fill) and `theme.extend.fontSize` had a key `bar` (19px, the size that makes
// beige legal on that fill). Tailwind resolves `text-{key}` against BOTH
// scales, so `text-bar` emitted two rules:
//
//     .text-bar { font-size: 1.1875rem; line-height: 1.2 }
//     .text-bar { color: rgb(70 137 115) }
//
// Every element using it for size alone was therefore painted mid-green.
// On the mid-green bar that is 1.00:1 — the text is not low-contrast, it is
// ABSENT. The quantity in the meal-detail stepper was invisible for exactly
// this reason, and it took a computed-style probe to see it, because the
// source says `text-bar font-bold` and reads as a size.
//
// Nothing about this is visible in review: the config is correct in isolation,
// each scale is correct in isolation, and the collision only exists in the
// generated CSS. So it is asserted here rather than remembered.
//
// Run:  node scripts/verify-tailwind-keys.js
// Exits non-zero on any collision.

const path = require('path')

async function main() {
  const configPath = path.join(__dirname, '..', 'web', 'tailwind.config.js')
  const mod = await import('file://' + configPath)
  const extend = mod.default?.theme?.extend ?? {}

  // The scales `text-*` draws from. A key present in more than one of these
  // makes `text-<key>` ambiguous, and Tailwind emits every match.
  const SCALES = ['colors', 'fontSize']

  const seen = new Map() // key -> [scale, …]
  for (const scale of SCALES) {
    for (const key of Object.keys(extend[scale] ?? {})) {
      // Nested colour groups (nourish.deep) produce `text-nourish-deep`, not
      // `text-nourish`, unless they carry a DEFAULT — which does collide.
      const value = extend[scale][key]
      const isGroup = scale === 'colors' && value && typeof value === 'object'
      if (isGroup && !('DEFAULT' in value)) continue
      seen.set(key, [...(seen.get(key) ?? []), scale])
    }
  }

  const clashes = [...seen.entries()].filter(([, scales]) => scales.length > 1)

  if (clashes.length === 0) {
    console.log('tailwind keys: no text-* collisions across ' + SCALES.join(' / '))
    return
  }

  console.error('\nTailwind key collision — `text-<key>` would set BOTH:\n')
  for (const [key, scales] of clashes) {
    console.error(`  text-${key}  ->  ${scales.join(' AND ')}`)
    console.error(`     Whichever rule Tailwind emits last wins, and the loser`)
    console.error(`     is silent. Rename one of them.\n`)
  }
  process.exit(1)
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
