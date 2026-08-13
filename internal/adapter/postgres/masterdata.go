package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MasterDataRepo owns the small admin-managed tables: customer types, diet
// types and their subtypes, allergens, delivery slots and organisations.
//
// They share a shape — searchable, sortable, soft-activated — so they share a
// repository rather than five near-identical ones.
type MasterDataRepo struct{ db *gorm.DB }

func NewMasterDataRepo(db *gorm.DB) *MasterDataRepo { return &MasterDataRepo{db: db} }

// CustomerType is a price-scope-bearing classification (PROMPT §4.1).
type CustomerType struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	IsCorporate   bool      `json:"is_corporate"`
	IsSystem      bool      `json:"is_system"`
	IsActive      bool      `json:"is_active"`
	SortOrder     int       `json:"sort_order"`
	CustomerCount int64     `json:"customer_count"`
}

// ListCustomerTypes returns a searchable page, with the customer count so an
// admin can see what deactivating one would affect.
func (r *MasterDataRepo) ListCustomerTypes(ctx context.Context, p ListParams) (Page[CustomerType], error) {
	p = p.Normalise("sort_order", "sort_order", "name", "slug")
	pattern := SearchPattern(p.Q)

	base := r.db.WithContext(ctx).Table("customer_type ct")
	if p.Q != "" {
		base = base.Where("lower(ct.name) LIKE ? OR lower(ct.slug) LIKE ?", pattern, pattern)
	}
	base = applyActive(base, p.Active)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[CustomerType]{}, fmt.Errorf("postgres: count customer types: %w", err)
	}

	var items []CustomerType
	err := base.Session(&gorm.Session{}).
		Select(`ct.id, ct.name, ct.slug, ct.description, ct.is_corporate,
		        ct.is_system, ct.is_active, ct.sort_order,
		        (SELECT count(*) FROM customer c WHERE c.customer_type_id = ct.id) AS customer_count`).
		Order(p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[CustomerType]{}, fmt.Errorf("postgres: list customer types: %w", err)
	}
	return NewPage(items, total, p), nil
}

// CreateCustomerType inserts one.
func (r *MasterDataRepo) CreateCustomerType(ctx context.Context, ct CustomerType, by uuid.UUID) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO customer_type (id, name, slug, description, is_corporate, is_active, sort_order, updated_by)
		VALUES (?,?,?,?,?,?,?,?)`,
		id, ct.Name, ct.Slug, ct.Description, ct.IsCorporate, ct.IsActive, ct.SortOrder, by).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres: create customer type: %w", err)
	}
	return id, nil
}

// GetCustomerType loads one, for the audit "before" state.
func (r *MasterDataRepo) GetCustomerType(ctx context.Context, id uuid.UUID) (CustomerType, error) {
	var ct CustomerType
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, slug, description, is_corporate, is_system, is_active, sort_order
		  FROM customer_type WHERE id = ?`, id).Scan(&ct).Error
	if err != nil {
		return CustomerType{}, fmt.Errorf("postgres: get customer type: %w", err)
	}
	if ct.ID == uuid.Nil {
		return CustomerType{}, ErrNotFound
	}
	return ct, nil
}

// UpdateCustomerType edits one.
func (r *MasterDataRepo) UpdateCustomerType(ctx context.Context, ct CustomerType, by uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE customer_type
		   SET name = ?, slug = ?, description = ?, is_corporate = ?,
		       is_active = ?, sort_order = ?, updated_by = ?
		 WHERE id = ?`,
		ct.Name, ct.Slug, ct.Description, ct.IsCorporate, ct.IsActive, ct.SortOrder, by, ct.ID)
	if res.Error != nil {
		return fmt.Errorf("postgres: update customer type: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DietType is a menu axis (PROMPT §4.2).
type DietType struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	HeroImageKey   *string   `json:"hero_image_key,omitempty"`
	SEOTitle       *string   `json:"seo_title,omitempty"`
	SEODescription *string   `json:"seo_description,omitempty"`
	HasSubtypes    bool      `json:"has_subtypes"`
	SortOrder      int       `json:"sort_order"`
	IsActive       bool      `json:"is_active"`
}

// ListDietTypes returns a searchable page.
func (r *MasterDataRepo) ListDietTypes(ctx context.Context, p ListParams) (Page[DietType], error) {
	p = p.Normalise("sort_order", "sort_order", "name", "slug")
	pattern := SearchPattern(p.Q)

	base := r.db.WithContext(ctx).Table("diet_type")
	if p.Q != "" {
		base = base.Where("lower(name) LIKE ? OR lower(slug) LIKE ? OR lower(description) LIKE ?",
			pattern, pattern, pattern)
	}
	base = applyActive(base, p.Active)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[DietType]{}, fmt.Errorf("postgres: count diet types: %w", err)
	}
	var items []DietType
	err := base.Session(&gorm.Session{}).
		Select("id, name, slug, description, hero_image_key, seo_title, seo_description, has_subtypes, sort_order, is_active").
		Order(p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[DietType]{}, fmt.Errorf("postgres: list diet types: %w", err)
	}
	return NewPage(items, total, p), nil
}

// GetDietType loads one.
func (r *MasterDataRepo) GetDietType(ctx context.Context, id uuid.UUID) (DietType, error) {
	var d DietType
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, slug, description, hero_image_key, seo_title, seo_description,
		       has_subtypes, sort_order, is_active
		  FROM diet_type WHERE id = ?`, id).Scan(&d).Error
	if err != nil {
		return DietType{}, fmt.Errorf("postgres: get diet type: %w", err)
	}
	if d.ID == uuid.Nil {
		return DietType{}, ErrNotFound
	}
	return d, nil
}

// GetDietTypeBySlug is the public lookup.
func (r *MasterDataRepo) GetDietTypeBySlug(ctx context.Context, slug string) (DietType, error) {
	var d DietType
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, slug, description, hero_image_key, seo_title, seo_description,
		       has_subtypes, sort_order, is_active
		  FROM diet_type WHERE slug = ? AND is_active`, slug).Scan(&d).Error
	if err != nil {
		return DietType{}, fmt.Errorf("postgres: get diet type by slug: %w", err)
	}
	if d.ID == uuid.Nil {
		return DietType{}, ErrNotFound
	}
	return d, nil
}

// CreateDietType inserts one.
func (r *MasterDataRepo) CreateDietType(ctx context.Context, d DietType, by uuid.UUID) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO diet_type (id, name, slug, description, hero_image_key, seo_title,
		                       seo_description, has_subtypes, sort_order, is_active, updated_by)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		id, d.Name, d.Slug, d.Description, d.HeroImageKey, d.SEOTitle, d.SEODescription,
		d.HasSubtypes, d.SortOrder, d.IsActive, by).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres: create diet type: %w", err)
	}
	return id, nil
}

// UpdateDietType edits one.
func (r *MasterDataRepo) UpdateDietType(ctx context.Context, d DietType, by uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE diet_type
		   SET name = ?, slug = ?, description = ?, hero_image_key = ?, seo_title = ?,
		       seo_description = ?, has_subtypes = ?, sort_order = ?, is_active = ?, updated_by = ?
		 WHERE id = ?`,
		d.Name, d.Slug, d.Description, d.HeroImageKey, d.SEOTitle, d.SEODescription,
		d.HasSubtypes, d.SortOrder, d.IsActive, by, d.ID)
	if res.Error != nil {
		return fmt.Errorf("postgres: update diet type: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Allergen is a warning label (PROMPT §13.2 — warn, never hide).
type Allergen struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	NameID    string    `json:"name_id"`
	NameEN    string    `json:"name_en"`
	Icon      *string   `json:"icon,omitempty"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
}

// ListAllergens returns a searchable page, searching BOTH languages: a staff
// member typing "kacang" and one typing "peanut" must both find it.
func (r *MasterDataRepo) ListAllergens(ctx context.Context, p ListParams) (Page[Allergen], error) {
	p = p.Normalise("sort_order", "sort_order", "code", "name_en")
	pattern := SearchPattern(p.Q)

	base := r.db.WithContext(ctx).Table("allergen")
	if p.Q != "" {
		base = base.Where("lower(code) LIKE ? OR lower(name_id) LIKE ? OR lower(name_en) LIKE ?",
			pattern, pattern, pattern)
	}
	base = applyActive(base, p.Active)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[Allergen]{}, fmt.Errorf("postgres: count allergens: %w", err)
	}
	var items []Allergen
	err := base.Session(&gorm.Session{}).
		Select("id, code, name_id, name_en, icon, sort_order, is_active").
		Order(p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[Allergen]{}, fmt.Errorf("postgres: list allergens: %w", err)
	}
	return NewPage(items, total, p), nil
}

// SlotAdmin is the staff view of a delivery slot: it shows the real time, which
// customers never see (PROMPT §8.1).
type SlotAdmin struct {
	ID             uuid.UUID `json:"id"`
	SlotTime       string    `json:"slot_time"`
	Alias          string    `json:"alias"`
	SortOrder      int       `json:"sort_order"`
	IsActive       bool      `json:"is_active"`
	CutoffTime     *string   `json:"cutoff_time,omitempty"`
	CutoffLeadDays *int      `json:"cutoff_lead_days,omitempty"`
}

// ListSlots returns every slot, active or not.
func (r *MasterDataRepo) ListSlots(ctx context.Context, p ListParams) (Page[SlotAdmin], error) {
	p = p.Normalise("sort_order", "sort_order", "slot_time", "alias")
	pattern := SearchPattern(p.Q)

	base := r.db.WithContext(ctx).Table("delivery_time_slot")
	if p.Q != "" {
		base = base.Where("lower(alias) LIKE ? OR slot_time::text LIKE ?", pattern, pattern)
	}
	base = applyActive(base, p.Active)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[SlotAdmin]{}, fmt.Errorf("postgres: count slots: %w", err)
	}
	var items []SlotAdmin
	err := base.Session(&gorm.Session{}).
		Select("id, slot_time::text AS slot_time, alias, sort_order, is_active, cutoff_time::text AS cutoff_time, cutoff_lead_days").
		Order(p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[SlotAdmin]{}, fmt.Errorf("postgres: list slots: %w", err)
	}
	return NewPage(items, total, p), nil
}

// UpdateSlot edits a slot. The 15-minute grid is a CHECK constraint, so an
// invalid time is refused by the database as well as by the form.
func (r *MasterDataRepo) UpdateSlot(ctx context.Context, s SlotAdmin, by uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE delivery_time_slot
		   SET slot_time = ?::time, alias = ?, sort_order = ?, is_active = ?,
		       cutoff_time = NULLIF(?,'')::time, cutoff_lead_days = ?, updated_by = ?
		 WHERE id = ?`,
		s.SlotTime, s.Alias, s.SortOrder, s.IsActive,
		derefString(s.CutoffTime), s.CutoffLeadDays, by, s.ID)
	if res.Error != nil {
		return fmt.Errorf("postgres: update slot: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Organisation is a corporate account (PROMPT §4.1, §13.6).
type Organisation struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	PICName          string    `json:"pic_name"`
	PICPhone         *string   `json:"pic_phone,omitempty"`
	BillingEmail     *string   `json:"billing_email,omitempty"`
	BillingAddress   string    `json:"billing_address"`
	NPWP             *string   `json:"npwp,omitempty"`
	PONumber         *string   `json:"po_number,omitempty"`
	IsInvoiceBilling bool      `json:"is_invoice_billing"`
	InvoiceDay       *int      `json:"invoice_day,omitempty"`
	IsActive         bool      `json:"is_active"`
	Notes            string    `json:"notes"`
	MemberCount      int64     `json:"member_count"`
}

// ListOrganisations returns a searchable page.
func (r *MasterDataRepo) ListOrganisations(ctx context.Context, p ListParams) (Page[Organisation], error) {
	p = p.Normalise("name", "name", "slug")
	pattern := SearchPattern(p.Q)

	base := r.db.WithContext(ctx).Table("organisation o")
	if p.Q != "" {
		base = base.Where(`lower(o.name) LIKE ? OR lower(o.slug) LIKE ?
		                   OR lower(o.pic_name) LIKE ? OR lower(o.billing_email::text) LIKE ?`,
			pattern, pattern, pattern, pattern)
	}
	base = applyActive(base, p.Active)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return Page[Organisation]{}, fmt.Errorf("postgres: count organisations: %w", err)
	}
	var items []Organisation
	err := base.Session(&gorm.Session{}).
		Select(`o.id, o.name, o.slug, o.pic_name, o.pic_phone, o.billing_email, o.billing_address,
		        o.npwp, o.po_number, o.is_invoice_billing, o.invoice_day, o.is_active, o.notes,
		        (SELECT count(*) FROM customer c WHERE c.organisation_id = o.id) AS member_count`).
		Order(p.OrderBy()).Limit(p.PageSize).Offset(p.Offset()).Scan(&items).Error
	if err != nil {
		return Page[Organisation]{}, fmt.Errorf("postgres: list organisations: %w", err)
	}
	return NewPage(items, total, p), nil
}

// CreateOrganisation inserts one.
func (r *MasterDataRepo) CreateOrganisation(ctx context.Context, o Organisation, by uuid.UUID) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO organisation (id, name, slug, pic_name, pic_phone, billing_email,
		                          billing_address, npwp, po_number, is_invoice_billing,
		                          invoice_day, is_active, notes, updated_by)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, o.Name, o.Slug, o.PICName, o.PICPhone, o.BillingEmail, o.BillingAddress,
		o.NPWP, o.PONumber, o.IsInvoiceBilling, o.InvoiceDay, o.IsActive, o.Notes, by).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres: create organisation: %w", err)
	}
	return id, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
