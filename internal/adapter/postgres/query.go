package postgres

import (
	"strings"

	"gorm.io/gorm"
)

// ListParams is the shape every list endpoint accepts.
//
// Search is not optional: "every screen rendering a list or table has a search
// box that filters it. No exceptions" (CLAUDE.md §7). Putting Q here means a
// list query cannot be written without at least deciding what it searches.
type ListParams struct {
	Q        string
	Page     int
	PageSize int
	Sort     string
	Desc     bool
	Active   *bool
}

// Normalise applies the defaults and the ceiling.
//
// The page-size ceiling is a control, not a nicety: without it, `?page_size=1000000`
// is a cheap way to make the server assemble a gigabyte of JSON.
func (p ListParams) Normalise(defaultSort string, allowedSorts ...string) ListParams {
	if p.Page < 1 {
		p.Page = 1
	}
	switch {
	case p.PageSize < 1:
		p.PageSize = 25
	case p.PageSize > 200:
		p.PageSize = 200
	}
	// Sort is an ALLOW-LIST, never interpolation: the column name reaches an
	// ORDER BY clause, which no amount of escaping makes safe.
	ok := false
	for _, s := range allowedSorts {
		if p.Sort == s {
			ok = true
			break
		}
	}
	if !ok {
		p.Sort = defaultSort
	}
	return p
}

// Offset for the SQL.
func (p ListParams) Offset() int { return (p.Page - 1) * p.PageSize }

// OrderBy renders the validated sort. Safe only because Normalise allow-listed
// the column.
func (p ListParams) OrderBy() string {
	dir := "ASC"
	if p.Desc {
		dir = "DESC"
	}
	return p.Sort + " " + dir
}

// Page is one page of results plus the count the UI needs.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// NewPage assembles a result page, never returning a nil slice — a JSON `null`
// where the client expects an array is a class of frontend crash.
func NewPage[T any](items []T, total int64, p ListParams) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Total: total, Page: p.Page, PageSize: p.PageSize}
}

// SearchPattern builds a case-insensitive LIKE pattern, escaping the wildcards
// so a customer searching for "100%" does not match everything.
func SearchPattern(q string) string {
	q = strings.TrimSpace(q)
	r := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return "%" + strings.ToLower(r.Replace(q)) + "%"
}

// applyActive adds the is_active filter when one was asked for.
func applyActive(db *gorm.DB, active *bool) *gorm.DB {
	if active == nil {
		return db
	}
	return db.Where("is_active = ?", *active)
}
