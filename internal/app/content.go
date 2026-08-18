package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/adapter/translate"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/i18n"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
)

// Content is editable public copy in three languages.
//
// The model Steven asked for, stated once here because the rules are easy to
// get subtly wrong:
//
//   - Indonesian is the SOURCE. It is the only language a person is required
//     to write.
//   - English and Chinese are DERIVED from it by the translator.
//   - Either may be OVERRIDDEN by hand, and an override is permanent: the
//     translator must never overwrite a human's words. That is the whole point
//     of "can be overridden when needed" — an override that a later edit
//     silently reverts is not an override.
//   - When the Indonesian changes, a derived translation is refreshed and an
//     overridden one is left alone but marked STALE, so an editor can see that
//     their English no longer matches the Indonesian and decide.
//
// Machine translation failing is never allowed to fail the save. The source is
// what a human typed; losing it because a third party timed out would be the
// worse outcome by far.
type Content struct {
	repo  *postgres.ContentRepo
	tr    translate.Translator
	audit *postgres.AuditRepo
	log   *slog.Logger
}

// ContentDeps wires the service.
type ContentDeps struct {
	Repo       *postgres.ContentRepo
	Translator translate.Translator
	Audit      *postgres.AuditRepo
	Log        *slog.Logger
}

func NewContent(d ContentDeps) *Content {
	tr := d.Translator
	if tr == nil {
		tr = translate.Noop{}
	}
	return &Content{repo: d.Repo, tr: tr, audit: d.Audit, log: d.Log}
}

// ── Reading ─────────────────────────────────────────────────────────────────

// ForLocale returns key -> copy for a language, already falling back to the
// Indonesian source where a translation is missing.
func (s *Content) ForLocale(ctx context.Context, locale i18n.Locale) (map[string]string, error) {
	return s.repo.ForLocale(ctx, string(locale))
}

// ── The admin view ──────────────────────────────────────────────────────────

// ContentEntry is one key across all three languages, as the back office shows
// it.
type ContentEntry struct {
	Key    string                        `json:"key"`
	Source string                        `json:"source"`
	Values map[string]ContentTranslation `json:"values"`
}

// ContentTranslation is one language of one key.
type ContentTranslation struct {
	Value string `json:"value"`
	// IsOverride: written by a person, and the translator will not touch it.
	IsOverride bool `json:"is_override"`
	// Stale: the Indonesian has changed since this was produced. Only
	// meaningful for overrides — a derived translation is refreshed on the
	// spot, so it is never stale.
	Stale bool `json:"stale"`
	// Empty: nothing here yet. The public page falls back to Indonesian, and
	// the editor needs to know that is what a reader is getting.
	Empty bool `json:"empty"`
}

// TranslatorStatus tells the admin screen whether auto-translation is on.
type TranslatorStatus struct {
	Available bool   `json:"available"`
	Provider  string `json:"provider"`
}

func (s *Content) TranslatorStatus() TranslatorStatus {
	return TranslatorStatus{Available: s.tr.Available(), Provider: s.tr.Name()}
}

// List returns every key with all three languages and their state.
func (s *Content) List(ctx context.Context) ([]ContentEntry, error) {
	rows, err := s.repo.All(ctx)
	if err != nil {
		return nil, err
	}

	sources := map[string]string{}
	for _, r := range rows {
		if r.Locale == string(i18n.ID) {
			sources[r.Key] = r.Value
		}
	}

	byKey := map[string]*ContentEntry{}
	order := make([]string, 0, len(sources))
	for _, r := range rows {
		e, ok := byKey[r.Key]
		if !ok {
			e = &ContentEntry{
				Key:    r.Key,
				Source: sources[r.Key],
				Values: map[string]ContentTranslation{},
			}
			byKey[r.Key] = e
			order = append(order, r.Key)
		}
		if r.Locale == string(i18n.ID) {
			continue
		}
		e.Values[r.Locale] = ContentTranslation{
			Value:      r.Value,
			IsOverride: r.IsOverride,
			Stale:      r.IsOverride && r.SourceHash != hashOf(sources[r.Key]),
			Empty:      strings.TrimSpace(r.Value) == "",
		}
	}

	out := make([]ContentEntry, 0, len(order))
	for _, k := range order {
		e := byKey[k]
		// A locale with no row at all still has to appear, or the screen has
		// no field to type the missing translation into.
		for _, l := range i18n.Supported {
			if l == i18n.ID {
				continue
			}
			if _, ok := e.Values[string(l)]; !ok {
				e.Values[string(l)] = ContentTranslation{Empty: true}
			}
		}
		out = append(out, *e)
	}
	return out, nil
}

// ── Writing ─────────────────────────────────────────────────────────────────

// SetSource replaces the Indonesian text and refreshes every derived
// translation.
//
// Returns the per-locale outcome so the caller can tell the editor what
// actually happened — "translated", "left as your override", or "no translator
// configured" are three different things and a silent save hides which.
func (s *Content) SetSource(ctx context.Context, key, value string, by Actor) (map[string]string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, apierror.BadRequest(apierror.CodeValidation, "A content key is required.")
	}
	clean, err := sanitize.Text("value", value, 2000)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(clean) == "" {
		return nil, apierror.BadRequest(apierror.CodeValidation,
			"The Indonesian text cannot be empty — it is what the other languages are made from.")
	}

	before, _ := s.repo.SourceValue(ctx, key)
	if err := s.repo.PutSource(ctx, key, clean, by.UserID); err != nil {
		return nil, err
	}
	s.audited(ctx, "content.source", key, before, clean, by)

	return s.retranslate(ctx, key, clean, by), nil
}

// retranslate refreshes the derived languages for one key.
//
// Never returns an error: the source is already saved, and a translator
// failure must not be reported as a failed save. Every outcome is reported per
// locale instead, and logged.
func (s *Content) retranslate(ctx context.Context, key, source string, by Actor) map[string]string {
	result := map[string]string{}
	hash := hashOf(source)

	for _, l := range i18n.Supported {
		if l == i18n.ID {
			continue
		}
		locale := string(l)

		override, err := s.repo.IsOverride(ctx, key, locale)
		if err != nil {
			result[locale] = "error"
			continue
		}
		if override {
			// Left alone on purpose. Its source_hash is not updated either,
			// which is exactly what makes it show as stale.
			result[locale] = "kept-override"
			continue
		}
		if !s.tr.Available() {
			result[locale] = "no-translator"
			continue
		}

		out, err := s.tr.Translate(ctx, source, string(i18n.ID), locale)
		if err != nil {
			// Leave whatever was there rather than blanking the page.
			if s.log != nil {
				s.log.Warn("content translation failed",
					"key", key, "locale", locale, "provider", s.tr.Name(), "error", err)
			}
			result[locale] = "failed"
			continue
		}
		if err := s.repo.PutTranslation(ctx, key, locale, out, false, hash, by.UserID); err != nil {
			result[locale] = "error"
			continue
		}
		result[locale] = "translated"
	}
	return result
}

// SetOverride writes a translation by hand and pins it.
func (s *Content) SetOverride(ctx context.Context, key, locale, value string, by Actor) error {
	l, ok := i18n.Parse(locale)
	if !ok || l == i18n.ID {
		return apierror.BadRequest(apierror.CodeValidation, "Override a translated language, not the source.")
	}
	clean, err := sanitize.Text("value", value, 2000)
	if err != nil {
		return err
	}
	source, err := s.repo.SourceValue(ctx, key)
	if err != nil {
		return err
	}
	if source == "" {
		return apierror.NotFound("No such content key.")
	}
	// Pinned to the CURRENT source, so a fresh override does not immediately
	// show as stale.
	if err := s.repo.PutTranslation(ctx, key, string(l), clean, true, hashOf(source), by.UserID); err != nil {
		return err
	}
	s.audited(ctx, "content.override", key+"."+string(l), "", clean, by)
	return nil
}

// ClearOverride releases a translation back to the machine and re-translates
// it immediately, so the editor sees the result rather than an empty box.
func (s *Content) ClearOverride(ctx context.Context, key, locale string, by Actor) (string, error) {
	l, ok := i18n.Parse(locale)
	if !ok || l == i18n.ID {
		return "", apierror.BadRequest(apierror.CodeValidation, "Only a translated language can be released.")
	}
	source, err := s.repo.SourceValue(ctx, key)
	if err != nil {
		return "", err
	}
	if source == "" {
		return "", apierror.NotFound("No such content key.")
	}
	if err := s.repo.PutTranslation(ctx, key, string(l), "", false, "", by.UserID); err != nil {
		return "", err
	}
	s.audited(ctx, "content.release", key+"."+string(l), "", "", by)

	out := s.retranslate(ctx, key, source, by)
	return out[string(l)], nil
}

// RetranslateAll refreshes every non-overridden translation — the button for
// after a translator is configured for the first time.
func (s *Content) RetranslateAll(ctx context.Context, by Actor) (map[string]map[string]string, error) {
	if !s.tr.Available() {
		return nil, apierror.BadRequest(apierror.CodeValidation, "No translation provider is configured.")
	}
	keys, err := s.repo.Keys(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]string{}
	for _, k := range keys {
		src, err := s.repo.SourceValue(ctx, k)
		if err != nil || strings.TrimSpace(src) == "" {
			continue
		}
		out[k] = s.retranslate(ctx, k, src, by)
	}
	return out, nil
}

// audited records the change. public_content is keyed by (key, locale) rather
// than by a UUID, so the key travels in the payload — Entry.EntityID is a
// *uuid.UUID and there is no id to put in it.
func (s *Content) audited(ctx context.Context, action, target, before, after string, by Actor) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &by.UserID, ActorEmail: by.Email,
		Action: action, EntityType: "public_content",
		Before: map[string]string{"key": target, "value": before},
		After:  map[string]string{"key": target, "value": after},
		IP:     by.IP, UserAgent: by.UA,
	})
}

func hashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
