/** Evermore Tailwind config.
 *
 * The palette is the design system's tokens, not a second opinion. Every
 * colour here has a measured contrast ratio in docs/10-design-system.md, and
 * the ones that FAIL AA are named so they cannot be reached for by accident.
 *
 * 2026-08-18: the page ground moved from Restore Beige to Nourish Green
 * #468973. Nothing reaches AA 4.5 on that green (white is the best case at
 * 4.13), so `canvas` is a GROUND — the app's content sits on `sheet` or on a
 * white card, never straight on it.
 */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        nourish: {
          deep: '#1C3D34',   // 11.32 on beige — text, fills, focus rings
          DEFAULT: '#468973',// the page ground; 2.88 vs deep ink, never ink-on
        },
        canvas: '#468973',   // ground only — display type and surfaces
        sheet: '#FFFAE0',    // where body copy lives: 11.32 for ink on it
        beige: {
          DEFAULT: '#FFFAE0',
          deep: '#CCBDAA',   // 1.75 on beige — divider between filled surfaces;
                             // 6.47 on the deep green, its footer role
        },
        brown: { deep: '#613F37', light: '#A36E50' },
        ocean: { deep: '#2E55A3', light: '#B6DAFA' },
        ember: { deep: '#E0782D', light: '#FFBC8F' }, // deep fails at 2.90
        berry: { deep: '#91253D', light: '#CC6883' },
        ink: { DEFAULT: '#1C3D34', muted: '#4A5D56' },
      },
      fontFamily: {
        display: ['Erode', 'Georgia', 'serif'],
        sans: ['InterVariable', 'Inter', 'system-ui', 'sans-serif'],
      },
      // Raised a step on 2026-08-18 (Steven: "more bold and bigger"). Kept in
      // step with the --text-* tokens in web/public/css/tokens.css.
      fontSize: {
        xs: ['0.8125rem', { lineHeight: '1.4' }],
        sm: ['0.9375rem', { lineHeight: '1.5' }],
        base: ['1.0625rem', { lineHeight: '1.65' }],
        lg: ['1.25rem', { lineHeight: '1.5' }],
        xl: ['1.5rem', { lineHeight: '1.3' }],
        '2xl': ['2rem', { lineHeight: '1.15' }],
        '3xl': ['2.75rem', { lineHeight: '1.05' }],
        '4xl': ['4rem', { lineHeight: '1.02' }],
      },
      boxShadow: {
        lift: '0 10px 30px rgba(11, 30, 25, 0.22)',   // a surface on the ground
      },
      minHeight: { touch: '44px' },  // docs/10 §4
      minWidth: { touch: '44px' },
    },
  },
  plugins: [],
}
