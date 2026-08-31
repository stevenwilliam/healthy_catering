import { useState } from 'react'

/** The photo band every artboard draws as "foto meal".
 *
 * The box is reserved whether or not a photo exists, and the fallback fills
 * exactly the same space. That is the point: photos arrive one meal at a time,
 * and a grid that reflows as they land is a grid that moves under the reader's
 * thumb between deciding and tapping.
 *
 * The fallback is the diet type's own tint from the public site's meal-art
 * palette, so a card looks deliberate rather than broken while the kitchen is
 * still photographing. Those tints are decorative fills carrying no text, so
 * the palette's text-contrast rules do not apply to them (docs/10 §2.4).
 *
 * A photo that 404s falls back too — storage keys outlive the objects they
 * name, and a broken-image glyph on a menu card looks like a broken shop.
 *
 * Two shapes reach this component and both are real. Seeded meals carry a
 * SERVED PATH ("/images/menu/ayam-panggang-brokoli.jpg"), which the Go static
 * mount answers directly. Uploaded meals carry an object-storage KEY, which
 * has to go through /api/v1/media to be presigned. Telling them apart on the
 * leading slash is what lets both exist while the library is migrated.
 */

/** src for either shape: a served path, or a storage key needing a presign. */
export function photoURL(key: string): string {
  return key.startsWith('/') ? key : `/api/v1/media/${encodeURIComponent(key)}`
}

const TINT: Record<string, string> = {
  healthy: '#468973',
  balanced: '#468973',
  'weight-gain': '#E0782D',
  'weight-loss': '#2E55A3',
  'high-protein': '#613F37',
  'muscle-gain': '#613F37',
  'special-diet': '#91253D',
  keto: '#A36E50',
}

function tintFor(diet: string): string {
  const key = diet.toLowerCase().replace(/\s+/g, '-')
  return TINT[key] ?? '#468973'
}

export function MealPhoto({
  photoKey, diet, alt, className = 'h-40',
}: {
  /** Object-storage key. Empty or missing is the normal case today. */
  photoKey?: string
  diet: string
  alt: string
  /** Height utility — the artboards use 92px in a list, 152px on detail. */
  className?: string
}) {
  const [broken, setBroken] = useState(false)
  const show = photoKey && !broken

  return (
    <div
      className={`relative w-full overflow-hidden ${className}`}
      style={show ? undefined : { background: tintFor(diet) }}
    >
      {show && (
        <img
          src={photoURL(photoKey)}
          alt={alt}
          loading="lazy"
          decoding="async"
          className="h-full w-full object-cover"
          onError={() => setBroken(true)}
        />
      )}
    </div>
  )
}
