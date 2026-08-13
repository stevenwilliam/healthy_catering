/** Evermore Tailwind config.
 *
 * The palette is the design system's tokens, not a second opinion. Every
 * colour here has a measured contrast ratio in docs/10-design-system.md, and
 * the ones that FAIL AA are named so they cannot be reached for by accident.
 */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        nourish: {
          deep: '#1C3D34',   // 11.32 on beige — text, fills, focus rings
          DEFAULT: '#468973',// 3.93 — DECORATION ONLY, never text or a fill
        },
        beige: {
          DEFAULT: '#FFFAE0',
          deep: '#CCBDAA',   // 1.75 on beige — divider between filled surfaces only
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
      minHeight: { touch: '44px' },  // docs/10 §4
      minWidth: { touch: '44px' },
    },
  },
  plugins: [],
}
