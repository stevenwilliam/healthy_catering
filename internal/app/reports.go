package app

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Reports is the operational reporting surface (PROMPT §12).
type Reports struct {
	repo   *postgres.ReportRepo
	params *sysparam.Store
	tz     *time.Location
}

func NewReports(r *postgres.ReportRepo, p *sysparam.Store, tz *time.Location) *Reports {
	return &Reports{repo: r, params: p, tz: tz}
}

// Scope is a validated report request.
type Scope struct {
	From      string
	To        string
	KitchenID *uuid.UUID
	SlotID    *uuid.UUID
	Q         string
}

// resolve turns a request into a repository scope, applying the caller's
// kitchen scoping.
//
// A kitchen-scoped user's KitchenID OVERRIDES whatever they asked for: the
// filter is not a convenience they can turn off by editing the query string
// (docs/02 D-21).
func (s *Reports) resolve(ident Identity, in Scope) (postgres.ReportScope, error) {
	from, err := s.date("from", in.From, 0)
	if err != nil {
		return postgres.ReportScope{}, err
	}
	to, err := s.date("to", in.To, 0)
	if err != nil {
		return postgres.ReportScope{}, err
	}
	if to.Before(from) {
		return postgres.ReportScope{}, apierror.Validation(
			"The end date is before the start date.", nil)
	}
	if to.Sub(from) > 366*24*time.Hour {
		return postgres.ReportScope{}, apierror.Validation(
			"Ask for at most a year at a time.", nil)
	}

	out := postgres.ReportScope{From: from, To: to, KitchenID: in.KitchenID, SlotID: in.SlotID, Q: in.Q}
	if ident.KitchenID != nil {
		out.KitchenID = ident.KitchenID
	}
	return out, nil
}

func (s *Reports) date(field, v string, defaultOffsetDays int) (time.Time, error) {
	if v == "" {
		now := time.Now().In(s.tz)
		return time.Date(now.Year(), now.Month(), now.Day()+defaultOffsetDays, 0, 0, 0, 0, s.tz), nil
	}
	t, err := time.ParseInLocation("2006-01-02", v, s.tz)
	if err != nil {
		return time.Time{}, apierror.Validation("That date is not valid.",
			map[string]any{field: "YYYY-MM-DD"})
	}
	return t, nil
}

func (s *Reports) ProductionSheet(ctx context.Context, ident Identity, in Scope) ([]postgres.ProductionRow, error) {
	sc, err := s.resolve(ident, in)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ProductionSheet(ctx, sc)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return rows, nil
}

func (s *Reports) PackingLabels(ctx context.Context, ident Identity, in Scope) ([]postgres.PackingLabel, error) {
	sc, err := s.resolve(ident, in)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.PackingLabels(ctx, sc)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return rows, nil
}

func (s *Reports) CourierManifest(ctx context.Context, ident Identity, in Scope) ([]postgres.ManifestStop, error) {
	sc, err := s.resolve(ident, in)
	if err != nil {
		return nil, err
	}
	if sc.KitchenID == nil {
		return nil, apierror.Validation(
			"A manifest is per kitchen — choose one.", map[string]any{"kitchen_id": "required"})
	}
	rows, err := s.repo.CourierManifest(ctx, sc)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return rows, nil
}

func (s *Reports) Coverage(ctx context.Context, ident Identity, in Scope) ([]postgres.CoverageRow, error) {
	sc, err := s.resolve(ident, in)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.CoverageReport(ctx, sc)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return rows, nil
}

func (s *Reports) Sales(ctx context.Context, ident Identity, in Scope, groupBy string) ([]postgres.SalesRow, error) {
	sc, err := s.resolve(ident, in)
	if err != nil {
		return nil, err
	}
	g := "day"
	if groupBy != "" {
		g, err = sanitize.Enum("group_by", groupBy, "day", "week", "month")
		if err != nil {
			return nil, validationFrom(err)
		}
	}
	rows, err := s.repo.SalesReport(ctx, sc, g)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return rows, nil
}

func (s *Reports) Credits(ctx context.Context, p postgres.ListParams) (postgres.Page[postgres.CreditReportRow], error) {
	page, err := s.repo.CreditReport(ctx, p)
	if err != nil {
		return postgres.Page[postgres.CreditReportRow]{}, apierror.Internal(err)
	}
	return page, nil
}

func (s *Reports) UnpaidAndExpiring(ctx context.Context) ([]postgres.UnpaidRow, error) {
	days := s.params.Int(ctx, sysparam.KeyExpiryWarningDays, 3)
	rows, err := s.repo.UnpaidAndExpiring(ctx, days)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return rows, nil
}

func (s *Reports) Retention(ctx context.Context) ([]postgres.RetentionRow, error) {
	rows, err := s.repo.Retention(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return rows, nil
}

// WriteCSV renders any report as CSV, neutralising spreadsheet formulas.
//
// Every cell goes through sanitize.CSVCell: a delivery note reading
// `=cmd|'/c calc'!A1` is a formula in Excel, and every one of these reports is
// exported and opened on a staff laptop (CLAUDE.md §4, D11).
// CSVDelimiter is the field separator for every export in the product. One
// constant, so a second exporter cannot quietly ship comma-separated files
// alongside pipe-separated ones.
const CSVDelimiter = '|'

func WriteCSV(w io.Writer, headers []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	// PIPE, not comma (99 §8). This data is Indonesian: addresses, dish names
	// and courier notes contain commas constantly, and a comma-delimited file
	// of it opens misaligned in Excel often enough to be useless.
	//
	// Still a real RFC 4180 writer, not a hand-joined string — a value that
	// itself contains a pipe, a quote or a newline is quoted and survives the
	// round trip. encoding/csv quotes on its Comma, whatever that is set to.
	cw.Comma = CSVDelimiter
	if err := cw.Write(headers); err != nil {
		return fmt.Errorf("csv: header: %w", err)
	}
	for _, r := range rows {
		safe := make([]string, len(r))
		for i, cell := range r {
			safe[i] = sanitize.CSVCell(cell)
		}
		if err := cw.Write(safe); err != nil {
			return fmt.Errorf("csv: row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}
