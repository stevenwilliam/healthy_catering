package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
)

// seed-menu-images gives every sample menu a placeholder photograph.
//
//	./bin/api seed-menu-images
//
// Steven asked for AI-generated dish photography; that needs an image model and
// a key this box does not have (RUN-WHEN-BACK §A2). This is the stand-in he
// asked for instead: a random photograph per menu, so the cards are not blank
// while the real ones are commissioned.
//
// Three things about how it does it, none of them incidental:
//
//   - The files are DOWNLOADED ONCE AND STORED LOCALLY, never hot-linked. A
//     third-party image URL on a public page means that third party sees every
//     visitor's IP and which page they are on — the same objection that keeps
//     the fonts self-hosted (CLAUDE.md §7, 99 §7) — and the CSP would refuse
//     the request in any case.
//   - The seed is the MENU NAME, so the same dish always gets the same picture
//     across days, and a re-run is stable rather than reshuffling the site.
//   - It skips any file already on disk, so it needs the network exactly once.
//
// The pictures are Lorem Picsum, which serves Unsplash photographs under the
// Unsplash licence: free to use, including commercially. They are still
// PLACEHOLDERS — they are not pictures of this food, and they should not
// survive to launch.
func runSeedMenuImages(ctx context.Context, gdb *gorm.DB, log *slog.Logger) error {
	const (
		dir       = "./web/public/images/menu"
		webPrefix = "/images/menu/"
		// 16:9 to match the card's band, at twice its rendered width.
		source = "https://picsum.photos/seed/%s/800/450"
	)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("seed-images: creating %s: %w", dir, err)
	}

	var names []string
	if err := gdb.WithContext(ctx).Raw(`
		SELECT DISTINCT name FROM scheduled_meal
		 WHERE name IS NOT NULL AND name <> '' ORDER BY name`).
		Scan(&names).Error; err != nil {
		return fmt.Errorf("seed-images: reading menus: %w", err)
	}
	if len(names) == 0 {
		return fmt.Errorf("seed-images: no named menus — run `seed-menu` first")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	fetched, reused := 0, 0

	for _, name := range names {
		slug := slugify(name)
		if slug == "" {
			continue
		}
		path := filepath.Join(dir, slug+".jpg")

		if _, err := os.Stat(path); err == nil {
			reused++
		} else {
			if err := fetchImage(ctx, client, fmt.Sprintf(source, slug), path); err != nil {
				// One picture failing is not a reason to abandon the rest, and
				// the menu is perfectly usable without it — the card falls
				// back to its illustrated band.
				log.Warn("seed-images: could not fetch a placeholder",
					"menu", name, "error", err)
				continue
			}
			fetched++
		}

		if err := gdb.WithContext(ctx).Exec(`
			UPDATE scheduled_meal SET hero_photo_key = ?
			 WHERE name = ? AND (hero_photo_key IS NULL OR hero_photo_key = '')`,
			webPrefix+slug+".jpg", name).Error; err != nil {
			return fmt.Errorf("seed-images: attaching %s: %w", slug, err)
		}
	}

	log.Info("seed-menu-images complete",
		"menus", len(names), "downloaded", fetched, "already_on_disk", reused,
		"note", "placeholders under the Unsplash licence — replace before launch")
	return nil
}

// fetchImage downloads one picture and writes it only if it really is one.
func fetchImage(ctx context.Context, client *http.Client, url, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %d", url, resp.StatusCode)
	}
	// Trust the bytes, not the URL: an error page served as HTML must not be
	// written out as a .jpg for the site to then serve as an image.
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		return fmt.Errorf("%s returned %q, not an image", url, ct)
	}

	// 8 MB ceiling. These are ~60 KB; anything near the cap means something
	// other than what was asked for.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if len(body) < 1024 {
		return fmt.Errorf("%s returned %d bytes, too small to be a photograph", url, len(body))
	}

	// Write to a temporary file and rename, so an interrupted download cannot
	// leave a truncated image on disk for the site to serve.
	tmp := path + ".part"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// slugify makes a filename-safe, stable key out of a menu name.
func slugify(s string) string {
	var b strings.Builder
	lastDash := true // leading dashes suppressed
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) && r < unicode.MaxASCII, unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
