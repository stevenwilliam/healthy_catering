package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContentRepo stores editable public copy in three languages.
//
// See migration 0017 for why this is a table rather than more sys_parameters
// rows: it carries which locale is the source, whether a translation was
// hand-written, and whether it has gone stale.
type ContentRepo struct{ db *gorm.DB }

func NewContentRepo(db *gorm.DB) *ContentRepo { return &ContentRepo{db: db} }

// ContentRow is one key in one language.
type ContentRow struct {
	Key        string    `json:"key"`
	Locale     string    `json:"locale"`
	Value      string    `json:"value"`
	IsOverride bool      `json:"is_override"`
	IsHTML     bool      `json:"is_html"`
	SourceHash string    `json:"source_hash"`
	UpdatedAt  time.Time `json:"updated_at"`
	UpdatedBy  *string   `json:"updated_by,omitempty"`
}

func (ContentRow) TableName() string { return "public_content" }

// All returns every row, ordered so the admin screen can group by key.
func (r *ContentRepo) All(ctx context.Context) ([]ContentRow, error) {
	var out []ContentRow
	err := r.db.WithContext(ctx).
		Table("public_content").
		Select(`key, locale, value, is_override, is_html, source_hash, updated_at,
		        updated_by::text AS updated_by`).
		Order("key, locale").
		Scan(&out).Error
	return out, err
}

// HTMLKeys returns the set of keys whose value is markup.
func (r *ContentRepo) HTMLKeys(ctx context.Context) (map[string]bool, error) {
	var keys []string
	if err := r.db.WithContext(ctx).Raw(
		`SELECT key FROM public_content WHERE locale = 'id' AND is_html`).
		Scan(&keys).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = true
	}
	return out, nil
}

// ForLocale returns key -> value for one language, falling back to Indonesian
// per key when the translation is empty.
//
// The fallback is in SQL rather than in Go because the page renders on every
// request and this is one round trip either way: COALESCE on a LEFT JOIN
// against the source row.
func (r *ContentRepo) ForLocale(ctx context.Context, locale string) (map[string]string, error) {
	type kv struct{ Key, Value string }
	var rows []kv
	err := r.db.WithContext(ctx).Raw(`
		SELECT src.key,
		       COALESCE(NULLIF(tr.value, ''), src.value) AS value
		  FROM public_content src
		  LEFT JOIN public_content tr
		         ON tr.key = src.key AND tr.locale = ?
		 WHERE src.locale = 'id'`, locale).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Value
	}
	return out, nil
}

// IsHTML reports whether a key holds markup rather than a sentence. Read from
// the SOURCE row, so every locale of one key agrees — a key that were plain in
// English and HTML in Indonesian is how a raw tag reaches a page.
func (r *ContentRepo) IsHTML(ctx context.Context, key string) (bool, error) {
	var b bool
	err := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE(is_html, FALSE) FROM public_content
		  WHERE key = ? AND locale = 'id'`, key).Scan(&b).Error
	return b, err
}

// SourceValue reads the Indonesian text for a key.
func (r *ContentRepo) SourceValue(ctx context.Context, key string) (string, error) {
	var v string
	err := r.db.WithContext(ctx).Raw(
		`SELECT value FROM public_content WHERE key = ? AND locale = 'id'`, key).
		Scan(&v).Error
	return v, err
}

// Keys lists the content keys that exist, in a stable order.
func (r *ContentRepo) Keys(ctx context.Context) ([]string, error) {
	var out []string
	err := r.db.WithContext(ctx).Raw(
		`SELECT key FROM public_content WHERE locale = 'id' ORDER BY key`).
		Scan(&out).Error
	return out, err
}

// PutSource replaces the Indonesian text. It never touches the translations —
// deciding what happens to those is the service's job, because it depends on
// whether each one is an override.
func (r *ContentRepo) PutSource(ctx context.Context, key, value string, by uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO public_content (key, locale, value, is_override, source_hash, updated_by)
		VALUES (?, 'id', ?, FALSE, '', ?)
		ON CONFLICT (key, locale) DO UPDATE
		   SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by`,
		key, value, uuidOrNil(by)).Error
}

// PutTranslation writes a derived or overridden translation.
func (r *ContentRepo) PutTranslation(ctx context.Context, key, locale, value string,
	isOverride bool, sourceHash string, by uuid.UUID) error {
	// is_html is copied from the source row rather than passed in: it is a
	// property of the KEY, not of one translation, and a caller that forgot it
	// would turn a rich-text row into a plain one on the next save.
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO public_content (key, locale, value, is_override, source_hash, updated_by, is_html)
		VALUES (?, ?, ?, ?, ?, ?,
		        COALESCE((SELECT is_html FROM public_content WHERE key = ? AND locale = 'id'), FALSE))
		ON CONFLICT (key, locale) DO UPDATE
		   SET value = EXCLUDED.value,
		       is_override = EXCLUDED.is_override,
		       source_hash = EXCLUDED.source_hash,
		       updated_by = EXCLUDED.updated_by`,
		key, locale, value, isOverride, sourceHash, uuidOrNil(by), key).Error
}

// IsOverride reports whether a translation is hand-written.
func (r *ContentRepo) IsOverride(ctx context.Context, key, locale string) (bool, error) {
	var b bool
	err := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE(is_override, FALSE) FROM public_content
		  WHERE key = ? AND locale = ?`, key, locale).Scan(&b).Error
	return b, err
}

func uuidOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
