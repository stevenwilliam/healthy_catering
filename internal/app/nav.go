package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
)

// Nav is the configurable header menu.
//
// A service rather than the repository handed straight to the HTTP layer:
// dependencies point inward (CLAUDE.md §2), and the audit write belongs on
// this side of the boundary with the rest of the business rules — a handler
// that remembers to audit is a handler that will one day forget.
type Nav struct {
	repo  *postgres.NavRepo
	audit *postgres.AuditRepo
}

func NewNav(r *postgres.NavRepo, a *postgres.AuditRepo) *Nav {
	return &Nav{repo: r, audit: a}
}

// Visible is what the public header renders.
func (s *Nav) Visible(ctx context.Context) ([]postgres.NavItem, error) {
	return s.repo.Visible(ctx)
}

// All is the admin grid, hidden items included.
func (s *Nav) All(ctx context.Context) ([]postgres.NavItem, error) {
	rows, err := s.repo.All(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return rows, nil
}

// Update changes visibility and position.
//
// Only those two. Path, kind and label_key are wiring, not configuration: a
// label typed in here would exist in one language on a trilingual site, and a
// path typed in here could point at a route that does not exist.
func (s *Nav) Update(ctx context.Context, id uuid.UUID, visible *bool, sort *int, by Actor) error {
	if visible == nil && sort == nil {
		return apierror.BadRequest(apierror.CodeValidation,
			"Send is_visible and/or sort_order.")
	}

	rows, err := s.repo.All(ctx)
	if err != nil {
		return apierror.Internal(err)
	}
	var before *postgres.NavItem
	for i := range rows {
		if rows[i].ID == id.String() {
			before = &rows[i]
			break
		}
	}
	if before == nil {
		return apierror.NotFound("No such menu item.")
	}

	// Read-modify-write, so a caller may send either field alone without the
	// other silently reverting to a zero value.
	newVisible, newSort := before.IsVisible, before.SortOrder
	if visible != nil {
		newVisible = *visible
	}
	if sort != nil {
		newSort = *sort
	}

	if err := s.repo.Update(ctx, id, newVisible, newSort, by.UserID); err != nil {
		return notFoundOr(err, "No such menu item.")
	}

	if s.audit != nil {
		_ = s.audit.Write(ctx, nil, postgres.Entry{
			ActorID: &by.UserID, ActorEmail: by.Email,
			Action: "nav.update", EntityType: "nav_item", EntityID: &id,
			Before: map[string]any{"key": before.Key,
				"is_visible": before.IsVisible, "sort_order": before.SortOrder},
			After: map[string]any{"key": before.Key,
				"is_visible": newVisible, "sort_order": newSort},
			IP: by.IP, UserAgent: by.UA,
		})
	}
	return nil
}
