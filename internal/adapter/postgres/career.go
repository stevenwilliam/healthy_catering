package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CareerRepo is the open positions and the applications sent for them.
type CareerRepo struct{ db *gorm.DB }

func NewCareerRepo(db *gorm.DB) *CareerRepo { return &CareerRepo{db: db} }

// JobOpening is a vacancy as the public page and the admin grid both show it.
type JobOpening struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	Summary   string `json:"summary"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

// ActiveOpenings is what the career page lists, and what the form's position
// field is populated from — the same rows, so a visitor cannot apply for a
// role that is not open.
func (r *CareerRepo) ActiveOpenings(ctx context.Context) ([]JobOpening, error) {
	out := []JobOpening{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT id::text, title, slug, summary, sort_order, is_active
		  FROM job_opening WHERE is_active
		 ORDER BY sort_order, title`).Scan(&out).Error
	return out, err
}

// AllOpenings is the admin grid, inactive rows included.
func (r *CareerRepo) AllOpenings(ctx context.Context) ([]JobOpening, error) {
	out := []JobOpening{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT id::text, title, slug, summary, sort_order, is_active
		  FROM job_opening ORDER BY is_active DESC, sort_order, title`).Scan(&out).Error
	return out, err
}

// JobApplication is one submission.
type JobApplication struct {
	ID          string    `json:"id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Position    string    `json:"position"`
	Message     string    `json:"message"`
	Locale      string    `json:"locale"`
	Status      string    `json:"status"`
	SubmittedIP string    `json:"submitted_ip"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateApplication stores a submission. Everything reaching here has already
// been sanitised and validated by the service.
func (r *CareerRepo) CreateApplication(ctx context.Context, a JobApplication) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO job_application
		  (id, full_name, email, phone, position, message, locale, submitted_ip, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, a.FullName, a.Email, a.Phone, a.Position, a.Message, a.Locale,
		a.SubmittedIP, a.UserAgent).Error
	return id, err
}

// ListApplications is the admin grid: searchable, newest first.
func (r *CareerRepo) ListApplications(ctx context.Context, p ListParams, status string) (Page[JobApplication], error) {
	p = p.Normalise("created_at DESC", "created_at", "full_name", "position", "status")
	pattern := SearchPattern(p.Q)

	base := r.db.WithContext(ctx).Table("job_application ja")
	if p.Q != "" {
		base = base.Where(`lower(ja.full_name) LIKE ? OR lower(ja.email::text) LIKE ?
		                   OR lower(ja.position) LIKE ? OR lower(ja.message) LIKE ?`,
			pattern, pattern, pattern, pattern)
	}
	if status != "" {
		base = base.Where("ja.status = ?", status)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[JobApplication]{}, err
	}

	items := []JobApplication{}
	err := base.Session(&gorm.Session{}).
		Select(`ja.id::text, ja.full_name, ja.email::text AS email, ja.phone,
		        ja.position, ja.message, ja.locale, ja.status,
		        ja.submitted_ip, ja.user_agent, ja.created_at`).
		Order(p.OrderBy).Limit(p.PageSize).Offset(p.Offset()).
		Scan(&items).Error
	if err != nil {
		return Page[JobApplication]{}, err
	}
	return Page[JobApplication]{
		Items: items, Total: total, Page: p.Page, PageSize: p.PageSize,
	}, nil
}

// SetApplicationStatus moves one through the pipeline.
func (r *CareerRepo) SetApplicationStatus(ctx context.Context, id uuid.UUID, status string, by uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(
		`UPDATE job_application SET status = ?, updated_by = ? WHERE id = ?`,
		status, uuidOrNil(by), id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
