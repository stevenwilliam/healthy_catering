package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NavRepo is the configurable header menu.
type NavRepo struct{ db *gorm.DB }

func NewNavRepo(db *gorm.DB) *NavRepo { return &NavRepo{db: db} }

// NavItem is one entry in the header.
type NavItem struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	LabelKey  string `json:"label_key"`
	ActiveKey string `json:"active_key"`
	SortOrder int    `json:"sort_order"`
	IsVisible bool   `json:"is_visible"`
}

// Visible is what the public header renders.
func (r *NavRepo) Visible(ctx context.Context) ([]NavItem, error) {
	out := []NavItem{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT id::text, key, kind, path, label_key, active_key, sort_order, is_visible
		  FROM nav_item WHERE is_visible ORDER BY sort_order, key`).Scan(&out).Error
	return out, err
}

// All is the admin grid, hidden items included.
func (r *NavRepo) All(ctx context.Context) ([]NavItem, error) {
	out := []NavItem{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT id::text, key, kind, path, label_key, active_key, sort_order, is_visible
		  FROM nav_item ORDER BY sort_order, key`).Scan(&out).Error
	return out, err
}

// Update sets visibility and position. The label, path and kind are not
// editable here: they are wiring, not configuration.
func (r *NavRepo) Update(ctx context.Context, id uuid.UUID, visible bool, sort int, by uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE nav_item SET is_visible = ?, sort_order = ?, updated_by = ?
		 WHERE id = ?`, visible, sort, uuidOrNil(by), id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
