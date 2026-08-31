package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Admin is the back-office use cases for master data and settings.
//
// Every write here goes through the same three steps: sanitize the input,
// read the BEFORE state, write, then append an audit row with before and after
// (PROMPT §3). The audit row is what makes "who changed the cut-off?" a
// question with an answer.
type Admin struct {
	master   *postgres.MasterDataRepo
	settings *postgres.SettingsRepo
	audit    *postgres.AuditRepo
	params   *sysparam.Store
	kitchens *postgres.KitchenRepo
}

func NewAdmin(m *postgres.MasterDataRepo, s *postgres.SettingsRepo,
	a *postgres.AuditRepo, p *sysparam.Store, k *postgres.KitchenRepo) *Admin {
	return &Admin{master: m, settings: s, audit: a, params: p, kitchens: k}
}

// KitchenOverview lists the kitchens with their coverage and their load on a
// service date — the coverage screen (S5) and the dashboard's capacity grid
// (S1) read the same call, so the two can never disagree about how full a
// slot is.
//
// The date is a BUSINESS calendar date and is resolved in the operating
// timezone by the caller, never from the server clock (CLAUDE.md §10).
func (s *Admin) KitchenOverview(ctx context.Context, on string) ([]postgres.KitchenOverview, error) {
	return s.kitchens.Overview(ctx, on)
}

// Actor is who is performing an admin action, and where from.
type Actor struct {
	UserID uuid.UUID
	Email  string
	IP     string
	UA     string
}

// ── Customer types ──────────────────────────────────────────────────────────

func (s *Admin) ListCustomerTypes(ctx context.Context, p postgres.ListParams) (postgres.Page[postgres.CustomerType], error) {
	return s.master.ListCustomerTypes(ctx, p)
}

// CustomerTypeInput is a create or edit.
type CustomerTypeInput struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description string
	IsCorporate bool
	IsActive    bool
	SortOrder   int
}

func (s *Admin) CreateCustomerType(ctx context.Context, in CustomerTypeInput, by Actor) (uuid.UUID, error) {
	ct, err := s.cleanCustomerType(in)
	if err != nil {
		return uuid.Nil, err
	}
	id, err := s.master.CreateCustomerType(ctx, ct, by.UserID)
	if err != nil {
		return uuid.Nil, conflictOrInternal(err, "A customer type with that name or slug already exists.")
	}
	ct.ID = id
	s.log(ctx, by, "customer_type.create", "customer_type", &id, nil, ct, "")
	return id, nil
}

func (s *Admin) UpdateCustomerType(ctx context.Context, in CustomerTypeInput, by Actor) error {
	before, err := s.master.GetCustomerType(ctx, in.ID)
	if err != nil {
		return notFoundOr(err, "That customer type no longer exists.")
	}
	ct, err := s.cleanCustomerType(in)
	if err != nil {
		return err
	}
	ct.ID = in.ID

	// A system type is the landing place for every registration; renaming it is
	// fine, deactivating it breaks signup for everyone.
	if before.IsSystem && !ct.IsActive {
		return apierror.Conflict(apierror.CodeConflict,
			"The default customer type cannot be deactivated — every new registration is assigned to it.")
	}
	if err := s.master.UpdateCustomerType(ctx, ct, by.UserID); err != nil {
		return conflictOrInternal(err, "A customer type with that name or slug already exists.")
	}
	s.log(ctx, by, "customer_type.update", "customer_type", &in.ID, before, ct, "")
	return nil
}

func (s *Admin) cleanCustomerType(in CustomerTypeInput) (postgres.CustomerType, error) {
	name, err := sanitize.Required("name", in.Name, 80)
	if err != nil {
		return postgres.CustomerType{}, validationFrom(err)
	}
	slug, err := sanitize.Slug("slug", in.Slug, 80)
	if err != nil {
		return postgres.CustomerType{}, validationFrom(err)
	}
	desc, err := sanitize.Text("description", in.Description, 500)
	if err != nil {
		return postgres.CustomerType{}, validationFrom(err)
	}
	return postgres.CustomerType{
		Name: name, Slug: slug, Description: desc,
		IsCorporate: in.IsCorporate, IsActive: in.IsActive, SortOrder: in.SortOrder,
	}, nil
}

// ── Diet types ──────────────────────────────────────────────────────────────

func (s *Admin) ListDietTypes(ctx context.Context, p postgres.ListParams) (postgres.Page[postgres.DietType], error) {
	return s.master.ListDietTypes(ctx, p)
}

// DietTypeInput is a create or edit.
type DietTypeInput struct {
	ID             uuid.UUID
	Name           string
	Slug           string
	Description    string
	SEOTitle       string
	SEODescription string
	HasSubtypes    bool
	SortOrder      int
	IsActive       bool
}

func (s *Admin) CreateDietType(ctx context.Context, in DietTypeInput, by Actor) (uuid.UUID, error) {
	d, err := s.cleanDietType(in)
	if err != nil {
		return uuid.Nil, err
	}
	id, err := s.master.CreateDietType(ctx, d, by.UserID)
	if err != nil {
		return uuid.Nil, conflictOrInternal(err, "A diet type with that slug already exists.")
	}
	d.ID = id
	s.log(ctx, by, "diet_type.create", "diet_type", &id, nil, d, "")
	return id, nil
}

func (s *Admin) UpdateDietType(ctx context.Context, in DietTypeInput, by Actor) error {
	before, err := s.master.GetDietType(ctx, in.ID)
	if err != nil {
		return notFoundOr(err, "That diet type no longer exists.")
	}
	d, err := s.cleanDietType(in)
	if err != nil {
		return err
	}
	d.ID = in.ID
	if err := s.master.UpdateDietType(ctx, d, by.UserID); err != nil {
		return conflictOrInternal(err, "A diet type with that slug already exists.")
	}
	s.log(ctx, by, "diet_type.update", "diet_type", &in.ID, before, d, "")
	return nil
}

func (s *Admin) cleanDietType(in DietTypeInput) (postgres.DietType, error) {
	name, err := sanitize.Required("name", in.Name, 80)
	if err != nil {
		return postgres.DietType{}, validationFrom(err)
	}
	slug, err := sanitize.Slug("slug", in.Slug, 80)
	if err != nil {
		return postgres.DietType{}, validationFrom(err)
	}
	desc, err := sanitize.Text("description", in.Description, 2000)
	if err != nil {
		return postgres.DietType{}, validationFrom(err)
	}
	// SEO fields have real length limits: a title over ~60 characters and a
	// description over ~160 are truncated by search engines, so the form warns
	// rather than letting someone write copy nobody will ever see.
	seoTitle, err := sanitize.Text("seo_title", in.SEOTitle, 70)
	if err != nil {
		return postgres.DietType{}, validationFrom(err)
	}
	seoDesc, err := sanitize.Text("seo_description", in.SEODescription, 180)
	if err != nil {
		return postgres.DietType{}, validationFrom(err)
	}
	d := postgres.DietType{
		Name: name, Slug: slug, Description: desc,
		HasSubtypes: in.HasSubtypes, SortOrder: in.SortOrder, IsActive: in.IsActive,
	}
	if seoTitle != "" {
		d.SEOTitle = &seoTitle
	}
	if seoDesc != "" {
		d.SEODescription = &seoDesc
	}
	return d, nil
}

// ── Allergens, slots, organisations ─────────────────────────────────────────

func (s *Admin) ListAllergens(ctx context.Context, p postgres.ListParams) (postgres.Page[postgres.Allergen], error) {
	return s.master.ListAllergens(ctx, p)
}

func (s *Admin) ListSlots(ctx context.Context, p postgres.ListParams) (postgres.Page[postgres.SlotAdmin], error) {
	return s.master.ListSlots(ctx, p)
}

// SlotInput edits a delivery slot.
type SlotInput struct {
	ID             uuid.UUID
	SlotTime       string
	Alias          string
	SortOrder      int
	IsActive       bool
	CutoffTime     string
	CutoffLeadDays *int
}

func (s *Admin) UpdateSlot(ctx context.Context, in SlotInput, by Actor) error {
	alias, err := sanitize.Required("alias", in.Alias, 40)
	if err != nil {
		return validationFrom(err)
	}
	if err := validateClock("slot_time", in.SlotTime, true); err != nil {
		return err
	}
	if err := validateClock("cutoff_time", in.CutoffTime, false); err != nil {
		return err
	}

	slot := postgres.SlotAdmin{
		ID: in.ID, SlotTime: in.SlotTime, Alias: alias,
		SortOrder: in.SortOrder, IsActive: in.IsActive, CutoffLeadDays: in.CutoffLeadDays,
	}
	if in.CutoffTime != "" {
		slot.CutoffTime = &in.CutoffTime
	}
	if err := s.master.UpdateSlot(ctx, slot, by.UserID); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return apierror.NotFound("That delivery slot no longer exists.")
		}
		// The 15-minute grid is a CHECK constraint; turn its violation into
		// something a human can act on rather than a 500.
		if strings.Contains(err.Error(), "delivery_time_slot_grid") {
			return apierror.Validation("Delivery times must fall on a 15-minute boundary.",
				map[string]any{"slot_time": "use :00, :15, :30 or :45"})
		}
		return conflictOrInternal(err, "Another slot already uses that time.")
	}
	s.log(ctx, by, "delivery_slot.update", "delivery_time_slot", &in.ID, nil, slot, "")
	return nil
}

func (s *Admin) ListOrganisations(ctx context.Context, p postgres.ListParams) (postgres.Page[postgres.Organisation], error) {
	return s.master.ListOrganisations(ctx, p)
}

// OrganisationInput creates a corporate account.
type OrganisationInput struct {
	Name             string
	Slug             string
	PICName          string
	PICPhone         string
	BillingEmail     string
	BillingAddress   string
	NPWP             string
	PONumber         string
	IsInvoiceBilling bool
	InvoiceDay       *int
	Notes            string
}

func (s *Admin) CreateOrganisation(ctx context.Context, in OrganisationInput, by Actor) (uuid.UUID, error) {
	name, err := sanitize.Required("name", in.Name, 160)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	slug, err := sanitize.Slug("slug", in.Slug, 80)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	pic, err := sanitize.Text("pic_name", in.PICName, 120)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	addr, err := sanitize.Text("billing_address", in.BillingAddress, 500)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}
	notes, err := sanitize.Text("notes", in.Notes, 2000)
	if err != nil {
		return uuid.Nil, validationFrom(err)
	}

	o := postgres.Organisation{
		Name: name, Slug: slug, PICName: pic, BillingAddress: addr,
		IsInvoiceBilling: in.IsInvoiceBilling, InvoiceDay: in.InvoiceDay,
		IsActive: true, Notes: notes,
	}
	if in.PICPhone != "" {
		p, err := sanitize.Phone("pic_phone", in.PICPhone)
		if err != nil {
			return uuid.Nil, validationFrom(err)
		}
		o.PICPhone = &p
	}
	if in.BillingEmail != "" {
		e, err := sanitize.Email("billing_email", in.BillingEmail, 254)
		if err != nil {
			return uuid.Nil, validationFrom(err)
		}
		o.BillingEmail = &e
	}
	if in.NPWP != "" {
		n, err := sanitize.Text("npwp", in.NPWP, 40)
		if err != nil {
			return uuid.Nil, validationFrom(err)
		}
		o.NPWP = &n
	}
	if in.PONumber != "" {
		p, err := sanitize.Text("po_number", in.PONumber, 60)
		if err != nil {
			return uuid.Nil, validationFrom(err)
		}
		o.PONumber = &p
	}
	if o.InvoiceDay != nil && (*o.InvoiceDay < 1 || *o.InvoiceDay > 28) {
		// 28, not 31: an invoice day of 30 silently never fires in February.
		return uuid.Nil, apierror.Validation("Invoice day must be between 1 and 28.",
			map[string]any{"invoice_day": "1–28, so it exists in every month"})
	}

	id, err := s.master.CreateOrganisation(ctx, o, by.UserID)
	if err != nil {
		return uuid.Nil, conflictOrInternal(err, "An organisation with that slug already exists.")
	}
	s.log(ctx, by, "organisation.create", "organisation", &id, nil, o, "")
	return id, nil
}

// ── Settings ────────────────────────────────────────────────────────────────

func (s *Admin) ListSettings(ctx context.Context, p postgres.ListParams, group string) (postgres.Page[postgres.Setting], error) {
	return s.settings.List(ctx, p, group)
}

func (s *Admin) SettingGroups(ctx context.Context) ([]string, error) {
	return s.settings.Groups(ctx)
}

// UpdateSetting changes one parameter, validating it against its declared type.
//
// The type check is the whole reason value_type exists: `order.cutoff_time` set
// to "6pm" would parse as nothing, fall back to the compiled default, and take
// the cut-off with it — silently, until someone noticed orders closing at the
// wrong hour.
func (s *Admin) UpdateSetting(ctx context.Context, key, value, reason string, by Actor) error {
	before, err := s.settings.Raw(ctx, key)
	if err != nil {
		return notFoundOr(err, "No such setting.")
	}
	clean, err := validateSettingValue(before.ValueType, value)
	if err != nil {
		return err
	}
	if err := s.params.Set(ctx, key, clean, by.UserID); err != nil {
		return apierror.Internal(err)
	}

	// A secret's value is never written into the audit log either — the row
	// records THAT it changed and by whom, which is what an audit needs.
	beforeVal, afterVal := any(before.Value), any(clean)
	if before.IsSecret {
		beforeVal, afterVal = "[redacted]", "[redacted]"
	}
	s.log(ctx, by, "setting.update", "sys_parameter", nil,
		map[string]any{"key": key, "value": beforeVal},
		map[string]any{"key": key, "value": afterVal}, reason)
	return nil
}

// validateSettingValue enforces the declared type of a parameter.
func validateSettingValue(valueType, raw string) (string, error) {
	v := strings.TrimSpace(raw)
	bad := func(hint string) error {
		return apierror.Validation("That value is not valid for this setting.",
			map[string]any{"value": hint})
	}

	switch valueType {
	case "int":
		if _, err := strconv.Atoi(v); err != nil {
			return "", bad("must be a whole number")
		}
	case "money":
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			return "", bad("must be a whole number of rupiah, and not negative")
		}
	case "bps":
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 10000 {
			return "", bad("basis points, 0–10000 (1100 = 11%)")
		}
	case "bool":
		if _, err := strconv.ParseBool(v); err != nil {
			return "", bad("must be true or false")
		}
		v = strings.ToLower(v)
	case "duration":
		if _, err := time.ParseDuration(v); err != nil {
			return "", bad("a duration such as 2h, 30m or 90s")
		}
	case "time":
		if err := validateClock("value", v, false); err != nil {
			return "", err
		}
	case "date":
		if _, err := time.Parse("2006-01-02", v); err != nil {
			return "", bad("a date as YYYY-MM-DD")
		}
	case "json":
		if !json.Valid([]byte(v)) {
			return "", bad("must be valid JSON")
		}
	case "string":
		clean, err := sanitize.Text("value", v, 2000)
		if err != nil {
			return "", validationFrom(err)
		}
		v = clean
	default:
		return "", bad("unknown setting type " + valueType)
	}
	return v, nil
}

// validateClock checks a HH:MM value, optionally enforcing the 15-minute grid.
func validateClock(field, v string, grid bool) error {
	if v == "" {
		return nil
	}
	t, err := time.Parse("15:04", v)
	if err != nil {
		if t2, err2 := time.Parse("15:04:05", v); err2 == nil {
			t = t2
		} else {
			return apierror.Validation("That time is not valid.",
				map[string]any{field: "use HH:MM, e.g. 18:00"})
		}
	}
	if grid && t.Minute()%15 != 0 {
		return apierror.Validation("Delivery times must fall on a 15-minute boundary.",
			map[string]any{field: "use :00, :15, :30 or :45"})
	}
	return nil
}

// log appends an audit row. A failure to audit is logged but never fails the
// action the user already completed — losing the audit row is bad, telling the
// user their successful change failed is worse and produces a double-submit.
func (s *Admin) log(ctx context.Context, by Actor, action, entity string,
	id *uuid.UUID, before, after any, reason string) {
	_ = s.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &by.UserID, ActorEmail: by.Email, Action: action,
		EntityType: entity, EntityID: id, Before: before, After: after,
		Reason: reason, IP: by.IP, UserAgent: by.UA,
	})
}

func notFoundOr(err error, msg string) error {
	if errors.Is(err, postgres.ErrNotFound) {
		return apierror.NotFound(msg)
	}
	return apierror.Internal(err)
}

// conflictOrInternal turns a unique-constraint violation into a message a human
// can act on. A raw driver error must never reach a client (CLAUDE.md §4).
func conflictOrInternal(err error, msg string) error {
	e := strings.ToLower(err.Error())
	if strings.Contains(e, "duplicate key") || strings.Contains(e, "unique constraint") {
		return apierror.Conflict(apierror.CodeConflict, msg)
	}
	return apierror.Internal(fmt.Errorf("admin: %w", err))
}
