package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// SettingsRepo lists sys_parameters for the admin screen.
//
// Reads for the application go through platform/sysparam, which caches and
// types them. This is the CRUD surface: searchable, grouped, and masking
// secrets (CLAUDE.md §7).
type SettingsRepo struct{ db *gorm.DB }

func NewSettingsRepo(db *gorm.DB) *SettingsRepo { return &SettingsRepo{db: db} }

// Setting is one row as the admin screen shows it.
type Setting struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	Group       string `json:"group"`
	Label       string `json:"label"`
	Description string `json:"description"`
	IsSecret    bool   `json:"is_secret"`
	IsSystem    bool   `json:"is_system"`
	SortOrder   int    `json:"sort_order"`
	UpdatedAt   string `json:"updated_at"`
	UpdatedBy   string `json:"updated_by,omitempty"`
}

// List returns a searchable page of settings.
//
// The search covers key, label, description AND group, because an admin looking
// for the cut-off will type "cutoff", "cut-off" or "ordering" with equal
// likelihood and should find it either way.
func (r *SettingsRepo) List(ctx context.Context, p ListParams, group string) (Page[Setting], error) {
	p = p.Normalise("param_group", "param_group", "key", "label", "sort_order")
	pattern := SearchPattern(p.Q)

	base := r.db.WithContext(ctx).Table("sys_parameters sp")
	if p.Q != "" {
		base = base.Where(`lower(sp.key) LIKE ? OR lower(sp.label) LIKE ?
		                   OR lower(sp.description) LIKE ? OR lower(sp.param_group) LIKE ?`,
			pattern, pattern, pattern, pattern)
	}
	if group != "" {
		base = base.Where("sp.param_group = ?", group)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[Setting]{}, fmt.Errorf("postgres: count settings: %w", err)
	}

	var items []Setting
	err := base.Session(&gorm.Session{}).
		Select(`sp.key, sp.value, sp.value_type, sp.param_group AS group, sp.label,
		        sp.description, sp.is_secret, sp.is_system, sp.sort_order,
		        sp.updated_at::text AS updated_at,
		        COALESCE(u.email::text, '') AS updated_by`).
		Joins("LEFT JOIN app_user u ON u.id = sp.updated_by").
		Order("sp.param_group ASC, sp.sort_order ASC").
		Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[Setting]{}, fmt.Errorf("postgres: list settings: %w", err)
	}

	// Masked at the edge of the repository, so no handler can forget it and no
	// log line downstream can print it (99 §7).
	for i := range items {
		if items[i].IsSecret && items[i].Value != "" {
			items[i].Value = "••••••••"
		}
	}
	return NewPage(items, total, p), nil
}

// Groups returns the distinct parameter groups, for the settings screen's tabs.
func (r *SettingsRepo) Groups(ctx context.Context) ([]string, error) {
	var groups []string
	err := r.db.WithContext(ctx).Raw(
		`SELECT DISTINCT param_group FROM sys_parameters ORDER BY param_group`).Scan(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: setting groups: %w", err)
	}
	return groups, nil
}

// Raw returns the unmasked stored value, for the audit "before" state. Never
// serve this to a client.
func (r *SettingsRepo) Raw(ctx context.Context, key string) (Setting, error) {
	var s Setting
	err := r.db.WithContext(ctx).Raw(`
		SELECT key, value, value_type, param_group AS group, label, description,
		       is_secret, is_system, sort_order, updated_at::text AS updated_at
		  FROM sys_parameters WHERE key = ?`, key).Scan(&s).Error
	if err != nil {
		return Setting{}, fmt.Errorf("postgres: get setting: %w", err)
	}
	if s.Key == "" {
		return Setting{}, ErrNotFound
	}
	return s, nil
}
