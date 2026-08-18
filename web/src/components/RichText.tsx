import { useEffect, useRef } from 'react'
import { useT } from '../lib/i18n'

/** A small WYSIWYG editor for the rich-text content blocks.
 *
 * Hand-rolled rather than a library. The alternative was a 100–300 kB editor
 * dependency for four formatting buttons on one admin screen, on an app whose
 * whole bundle is 75 kB gzipped. If the requirement ever grows past this —
 * tables, images, embeds — that trade flips and a real editor is the right
 * answer.
 *
 * document.execCommand is deprecated and has no replacement. Every browser
 * still implements it, every rich-text library still falls back to it, and the
 * standards position is that contenteditable editing commands are simply not
 * being re-specified. Written down so the deprecation warning is a known cost
 * rather than a surprise.
 *
 * SECURITY: what this produces is never trusted. The server sanitises it
 * against an allowlist on write and again on render
 * (internal/platform/richtext), so the worst an editor — or anyone who reaches
 * this endpoint — can store is markup the allowlist permits.
 */
export default function RichText({
  value,
  onChange,
  id,
}: {
  value: string
  onChange: (html: string) => void
  id?: string
}) {
  const t = useT()
  const box = useRef<HTMLDivElement>(null)

  // Written into the DOM only when the incoming value differs from what is
  // already there. Assigning innerHTML on every render would move the caret to
  // the start on every keystroke, which makes the field unusable.
  useEffect(() => {
    const el = box.current
    if (el && el.innerHTML !== value) el.innerHTML = value
  }, [value])

  function exec(command: string, arg?: string) {
    box.current?.focus()
    document.execCommand(command, false, arg)
    if (box.current) onChange(box.current.innerHTML)
  }

  function addLink() {
    const url = window.prompt(t('rich.link_prompt'), 'https://')
    if (!url) return
    // Only http(s) and mailto reach the server's allowlist anyway; refusing
    // here as well means the editor never shows a link that will silently
    // vanish on save.
    if (!/^(https?:|mailto:)/i.test(url)) {
      window.alert(t('rich.link_invalid'))
      return
    }
    exec('createLink', url)
  }

  const buttons: { label: string; title: string; run: () => void }[] = [
    { label: 'B', title: t('rich.bold'), run: () => exec('bold') },
    { label: 'I', title: t('rich.italic'), run: () => exec('italic') },
    { label: '• —', title: t('rich.bullets'), run: () => exec('insertUnorderedList') },
    { label: '1.', title: t('rich.numbers'), run: () => exec('insertOrderedList') },
    { label: '🔗', title: t('rich.link'), run: addLink },
    { label: '⌫', title: t('rich.clear'), run: () => exec('removeFormat') },
  ]

  return (
    <div className="rich">
      <div className="rich-toolbar" role="toolbar" aria-label={t('rich.toolbar')}>
        {buttons.map((b) => (
          <button
            key={b.title}
            type="button"
            className="rich-btn"
            // The visible glyph is a symbol, so the accessible name has to
            // come from somewhere: title for a mouse, aria-label for a reader.
            title={b.title}
            aria-label={b.title}
            onMouseDown={(e) => e.preventDefault()} // keep the selection
            onClick={b.run}
          >
            {b.label}
          </button>
        ))}
      </div>
      <div
        id={id}
        ref={box}
        className="rich-area"
        contentEditable
        suppressContentEditableWarning
        role="textbox"
        aria-multiline="true"
        onInput={() => onChange(box.current?.innerHTML ?? '')}
        onBlur={() => onChange(box.current?.innerHTML ?? '')}
      />
    </div>
  )
}
