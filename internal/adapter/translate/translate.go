// Package translate turns Indonesian source copy into the other UI languages.
//
// Steven, 2026-08-18: the back office writes Indonesian and the backend
// produces English and Chinese, with a human able to override either.
//
// Machine translation is a THIRD-PARTY CALL — it sends the text off the
// machine, it costs money per character, and it fails. So it sits behind an
// interface with a no-op default: with no provider configured the service
// still works, the source text still saves, and the translations are simply
// marked as needing a human. A marketing page that cannot be edited because a
// translation API is down would be a worse product than one that needs a
// person to type the English.
package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Translator is the port. app.Content depends on this, not on a vendor.
type Translator interface {
	// Available reports whether translation can actually happen. Callers use
	// it to tell "the machine has not translated this yet" apart from "there
	// is no machine", which are different messages to show an editor.
	Available() bool
	// Translate returns text rendered in `to`. `from` and `to` are the short
	// locale codes the rest of the system uses: id, en, zh.
	Translate(ctx context.Context, text, from, to string) (string, error)
	// Name identifies the provider in logs and in the admin screen.
	Name() string
}

// ── No provider ─────────────────────────────────────────────────────────────

// Noop is what runs when nothing is configured.
type Noop struct{}

func (Noop) Available() bool { return false }
func (Noop) Name() string    { return "none" }
func (Noop) Translate(context.Context, string, string, string) (string, error) {
	return "", ErrUnavailable
}

// ErrUnavailable means no translator is configured. Callers treat it as "leave
// the translation empty and flag it", never as a failure of the save.
var ErrUnavailable = fmt.Errorf("translate: no provider configured")

// ── Google Cloud Translation v2 ─────────────────────────────────────────────

// Google calls the v2 REST endpoint, which takes a plain API key.
//
// v2 rather than v3 deliberately: v3 needs a service account, OAuth and a
// project number, which is a lot of setup for four strings on a marketing
// page. v2 is one key in the environment.
type Google struct {
	key    string
	client *http.Client
}

// GoogleConfig is what the wiring passes in.
type GoogleConfig struct {
	APIKey  string
	Timeout time.Duration
}

// NewGoogle returns nil when no key is set, so the caller falls back to Noop
// rather than constructing something that can only fail.
func NewGoogle(cfg GoogleConfig) *Google {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Google{key: cfg.APIKey, client: &http.Client{Timeout: cfg.Timeout}}
}

func (g *Google) Available() bool { return g != nil && g.key != "" }
func (g *Google) Name() string    { return "google" }

// googleTarget maps our locale codes to the ones the API expects. "zh" alone
// is ambiguous to Google as well as to a reader, so it is pinned to
// Simplified — the same decision docs/11 records for the UI.
func googleTarget(locale string) string {
	switch locale {
	case "zh":
		return "zh-CN"
	default:
		return locale
	}
}

func (g *Google) Translate(ctx context.Context, text, from, to string) (string, error) {
	if !g.Available() {
		return "", ErrUnavailable
	}
	if strings.TrimSpace(text) == "" {
		return "", nil
	}

	form := url.Values{
		"q":      {text},
		"source": {googleTarget(from)},
		"target": {googleTarget(to)},
		// text, not html: the copy is plain strings and asking for HTML would
		// get entities escaped into the value we then store and re-escape.
		"format": {"text"},
	}

	endpoint := "https://translation.googleapis.com/language/translate/v2?key=" + url.QueryEscape(g.key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("translate: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Cap the read: a wrong endpoint returning a large body should not become
	// a memory problem.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("translate: reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		// The key is in the URL, so never echo the request. The API's own
		// message is safe and is what an admin needs to fix a bad key.
		return "", fmt.Errorf("translate: provider returned %d: %s",
			resp.StatusCode, firstLine(body))
	}

	var out struct {
		Data struct {
			Translations []struct {
				TranslatedText string `json:"translatedText"`
			} `json:"translations"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&out); err != nil {
		return "", fmt.Errorf("translate: decoding response: %w", err)
	}
	if len(out.Data.Translations) == 0 {
		return "", fmt.Errorf("translate: provider returned no translation")
	}
	return out.Data.Translations[0].TranslatedText, nil
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	return s
}

// New picks a provider from configuration. Unknown or empty means Noop.
func New(provider, apiKey string) Translator {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		if g := NewGoogle(GoogleConfig{APIKey: apiKey}); g != nil {
			return g
		}
	}
	return Noop{}
}
