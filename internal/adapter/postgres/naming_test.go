package postgres

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"gorm.io/gorm/schema"

	"github.com/stevenwilliam/healthy_catering/db"
)

// gorm's NamingStrategy renders "PriceIDR" as `price_id_r`, not `price_idr`:
// it knows "ID" is an initialism and splits the run there. A scan into a
// mismatched column does not error — it leaves the field at its zero value, so
// every price silently reads as Rp 0.
//
// That cost a real debugging session twice. The pricing engine resolved the
// right tier, applied the right scope, and quoted Rp 0; then on 2026-08-18 the
// public price list rendered "Rp 0" against real 55.000 and 48.000 rows because
// the new structs were written without the tags.
//
// This guard used to be a hand-maintained list of ten types, which only ever
// protected the structs somebody remembered to add — vigilance, not a guard.
// It now walks the package's own source, so a struct written tomorrow is
// covered the moment it is saved.
func TestEveryTrappedFieldHasACorrectColumnTag(t *testing.T) {
	ns := schema.NamingStrategy{}

	// Proof that the trap is real, so this test explains itself if it ever
	// fails for a new field.
	if ns.ColumnName("", "PriceIDR") == "price_idr" {
		t.Skip("gorm now maps IDR correctly; the explicit tags are harmless")
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool { return true }, 0)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	columns := schemaColumns(t, pkgs)

	checked := 0
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}
				for _, f := range st.Fields.List {
					for _, name := range f.Names {
						if !hasAcronymRun(name.Name) {
							continue
						}
						gormName := ns.ColumnName("", name.Name)
						intended := snakeKeepingAcronyms(name.Name)

						// An explicit tag is authoritative — a Go name may differ
						// from its column on purpose (ExpectedIDR is
						// expected_amount_idr). What it may NOT be is a column that
						// does not exist, which is the check the previous version of
						// this test claimed to make and did not: it re-tested the
						// condition it had already returned on, and the column name
						// it computed was never read.
						if col, ok := columnOf(structTag(f)); ok {
							checked++
							if !columns[col] {
								t.Errorf("%s.%s (%s): tagged column:%s, which no migration defines — "+
									"a scan into a column that does not exist does not error, it reads zero",
									ts.Name.Name, name.Name, shortPath(path), col)
							}
							continue
						}
						if columns[gormName] {
							continue // gorm's split happens to be right (seo_title)
						}
						if columns[intended] {
							checked++
							t.Errorf("%s.%s (%s): gorm maps this to %q, which is not a column, "+
								"while %q is — add gorm:\"column:%s\" or the scan is a silent zero",
								ts.Name.Name, name.Name, shortPath(path), gormName, intended, intended)
						}
						// Neither name is a column: the field is not database-backed.
					}
				}
				return true
			})
		}
	}

	if checked == 0 {
		t.Fatal("the sweep matched no fields at all — it has stopped testing anything")
	}
	t.Logf("checked %d trapped field(s) across the package", checked)
}

// hasAcronymRun reports a run of two or more capitals — the only shape where
// gorm's initialism splitting can disagree with the column.
func hasAcronymRun(name string) bool {
	run := 0
	for _, r := range name {
		if unicode.IsUpper(r) {
			run++
			if run >= 2 {
				return true
			}
			continue
		}
		run = 0
	}
	return false
}

// schemaColumns is every column name the migrations define. The migrations are
// the source of truth (CLAUDE.md §4), so they are the right oracle for whether
// a tag names something real.
func schemaColumns(t *testing.T, pkgs map[string]*ast.Package) map[string]bool {
	t.Helper()
	ms, err := db.Migrations()
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	defRe := regexp.MustCompile(`(?m)^\s+([a-z_][a-z0-9_]*)\s+[A-Z]`)
	addRe := regexp.MustCompile(`(?i)ADD COLUMN(?:\s+IF NOT EXISTS)?\s+([a-z_][a-z0-9_]*)`)
	out := map[string]bool{}
	for _, m := range ms {
		for _, re := range []*regexp.Regexp{defRe, addRe} {
			for _, g := range re.FindAllStringSubmatch(m.Up, -1) {
				out[strings.ToLower(g[1])] = true
			}
		}
	}
	if len(out) < 100 {
		t.Fatalf("only %d columns parsed out of %d migrations — the extractor has stopped working", len(out), len(ms))
	}

	// A report selects computed values under an alias — SUM(...) AS gross_idr is
	// scanned by name exactly like a physical column, so the package's own SQL is
	// part of the oracle.
	//
	// STRING LITERALS ONLY, via the AST. Reading the raw file text instead let a
	// COMMENT into the oracle: pricing_public.go explains that gorm "renders
	// UnitPriceIDR as unit_price_id_r, which matches nothing" — and the alias
	// regex read that prose as a column definition, so the guard cheerfully
	// accepted the exact broken name it exists to catch. The comment warning
	// about the incident was switching off the test for the incident.
	aliasRe := regexp.MustCompile(`(?i)\bAS\s+([a-z_][a-z0-9_]*)`)
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				for _, g := range aliasRe.FindAllStringSubmatch(lit.Value, -1) {
					out[strings.ToLower(g[1])] = true
				}
				return true
			})
		}
	}
	return out
}

// snakeKeepingAcronyms renders a Go field name the way the hand-written SQL
// does: a run of capitals stays one word, so UnitPriceIDR is unit_price_idr.
func snakeKeepingAcronyms(name string) string {
	var out []string
	var cur []rune
	var prevUpper bool
	for i, r := range name {
		isUpper := unicode.IsUpper(r)
		if i > 0 && isUpper && !prevUpper {
			out = append(out, string(cur))
			cur = nil
		}
		cur = append(cur, unicode.ToLower(r))
		prevUpper = isUpper
	}
	out = append(out, string(cur))
	return strings.Join(out, "_")
}

func structTag(f *ast.Field) reflect.StructTag {
	if f.Tag == nil {
		return ""
	}
	return reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
}

func columnOf(tag reflect.StructTag) (string, bool) {
	for _, part := range strings.Split(tag.Get("gorm"), ";") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(part), "column:"); ok {
			return rest, true
		}
	}
	return "", false
}

func shortPath(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// The same trap applies to any field whose Go name contains an initialism gorm
// does not know. This lists the ones we rely on, so a rename is caught here.
func TestKnownColumnNames(t *testing.T) {
	ns := schema.NamingStrategy{}
	safe := map[string]string{
		"ScopeKey":    "scope_key",
		"TierID":      "tier_id",
		"CustomerID":  "customer_id",
		"OrderCode":   "order_code",
		"ServiceDate": "service_date",
		"IsActive":    "is_active",
		"QtyReserved": "qty_reserved",
		"TaxRateBps":  "tax_rate_bps",
	}
	for field, want := range safe {
		if got := ns.ColumnName("", field); got != want {
			t.Errorf("gorm maps %s to %q, but the SQL selects %q", field, got, want)
		}
	}
}
