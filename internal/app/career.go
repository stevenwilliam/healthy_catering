package app

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/i18n"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
)

// Career is the public application form and the back office behind it.
//
// Two rules, both from Steven and both enforced here rather than trusted to
// the form:
//
//   - Every field is sanitised and validated on the SERVER. The form is a
//     courtesy; this endpoint is reachable with curl (CLAUDE.md §4).
//   - NO FILES. There is no attachment field, no multipart parsing and no
//     storage. An upload endpoint on an unauthenticated public page is the
//     highest-risk surface a marketing site can have, and a CV is something a
//     person emails after we have replied.
type Career struct {
	repo  *postgres.CareerRepo
	audit *postgres.AuditRepo
}

func NewCareer(r *postgres.CareerRepo, a *postgres.AuditRepo) *Career {
	return &Career{repo: r, audit: a}
}

// Openings lists what is being hired for, for the page and the form.
func (s *Career) Openings(ctx context.Context) ([]postgres.JobOpening, error) {
	return s.repo.ActiveOpenings(ctx)
}

func (s *Career) AllOpenings(ctx context.Context) ([]postgres.JobOpening, error) {
	return s.repo.AllOpenings(ctx)
}

// ApplicationInput is what the form sends. There is no file field, and adding
// one is a product decision, not an implementation detail.
type ApplicationInput struct {
	FullName string
	Email    string
	Phone    string
	Position string
	Message  string
	Locale   i18n.Locale
	IP       string
	UserAgent string
}

// FieldErrors maps a form field to its message, so the page can mark the field
// that is wrong instead of showing one error at the top and making the
// applicant guess.
type FieldErrors map[string]string

// Apply validates, sanitises and stores an application.
//
// Returns per-field errors rather than a single message: a form that says only
// "invalid input" for five fields is a form people abandon.
func (s *Career) Apply(ctx context.Context, in ApplicationInput) (FieldErrors, error) {
	errs := FieldErrors{}
	out := postgres.JobApplication{Locale: string(in.Locale)}

	if v, err := sanitize.Required("full_name", in.FullName, 120); err != nil {
		errs["full_name"] = "required"
	} else {
		out.FullName = v
	}

	if v, err := sanitize.Email("email", in.Email, 254); err != nil {
		errs["email"] = "invalid"
	} else {
		out.Email = v
	}

	// Optional: plenty of applicants would rather be emailed, and demanding a
	// phone number costs applications.
	if strings.TrimSpace(in.Phone) != "" {
		if v, err := sanitize.Phone("phone", in.Phone); err != nil {
			errs["phone"] = "invalid"
		} else {
			out.Phone = v
		}
	}

	// The position must be one currently open. Checked against the database,
	// not against the submitted list — the form can be edited before it is
	// sent, and an application for a role nobody is hiring is work for HR.
	openings, err := s.repo.ActiveOpenings(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	titles := make([]string, 0, len(openings))
	for _, o := range openings {
		titles = append(titles, o.Slug)
	}
	if v, err := sanitize.Enum("position", in.Position, titles...); err != nil {
		errs["position"] = "invalid"
	} else {
		out.Position = v
	}

	if v, err := sanitize.Text("message", in.Message, 4000); err != nil {
		errs["message"] = "invalid"
	} else {
		out.Message = v
	}
	if strings.TrimSpace(out.Message) == "" {
		errs["message"] = "required"
	}

	if len(errs) > 0 {
		return errs, nil
	}

	// Truncated rather than validated: these are diagnostics, and a header is
	// attacker-controlled and can be arbitrarily long.
	out.SubmittedIP = truncate(in.IP, 64)
	out.UserAgent = truncate(in.UserAgent, 300)

	id, err := s.repo.CreateApplication(ctx, out)
	if err != nil {
		return nil, apierror.Internal(err)
	}

	// Audited without the message body: the audit log is read by more people
	// than the application queue is, and a covering letter is personal.
	if s.audit != nil {
		_ = s.audit.Write(ctx, nil, postgres.Entry{
			Action: "career.apply", EntityType: "job_application", EntityID: &id,
			After: map[string]string{"position": out.Position, "locale": out.Locale},
			IP:    out.SubmittedIP, UserAgent: out.UserAgent,
		})
	}
	return nil, nil
}

// ListApplications is the admin grid.
func (s *Career) ListApplications(ctx context.Context, p postgres.ListParams, status string) (postgres.Page[postgres.JobApplication], error) {
	page, err := s.repo.ListApplications(ctx, p, status)
	if err != nil {
		return page, apierror.Internal(err)
	}
	return page, nil
}

// SetStatus moves an application through the pipeline.
func (s *Career) SetStatus(ctx context.Context, id uuid.UUID, status string, by Actor) error {
	v, err := sanitize.Enum("status", status,
		"NEW", "REVIEWING", "CONTACTED", "REJECTED", "HIRED")
	if err != nil {
		return validationFrom(err)
	}
	if err := s.repo.SetApplicationStatus(ctx, id, v, by.UserID); err != nil {
		return notFoundOr(err, "No such application.")
	}
	if s.audit != nil {
		_ = s.audit.Write(ctx, nil, postgres.Entry{
			ActorID: &by.UserID, ActorEmail: by.Email,
			Action: "career.status", EntityType: "job_application", EntityID: &id,
			After: map[string]string{"status": v}, IP: by.IP, UserAgent: by.UA,
		})
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
