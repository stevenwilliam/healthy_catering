// Package routing assigns a delivery to a kitchen.
//
// Steven, 2026-08-12: "auto assign location customer to nearest kitchen"
// (D-34). The brief's §9.3 ranks by priority → distance → remaining capacity;
// seeding every kitchen at the same priority collapses that to nearest-first,
// and leaves priority as the manual override for the case the map gets wrong —
// three kilometres away but across a toll road.
//
// It must be deterministic and explainable: staff will be asked "why did this
// go to Kitchen B?" and the answer is persisted on the delivery row.
//
// Pure: the caller supplies candidate kitchens and their capacity. Distance is
// computed here with the haversine formula so the decision can be unit-tested
// without a database; the repository pre-filters with PostGIS so the candidate
// list is short.
package routing

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ErrNotServiceable means no kitchen covers the point. The order is blocked and
// the attempt is logged — that log is the map of where to open next (§9.3.5).
var ErrNotServiceable = errors.New("routing: address is not serviceable")

// Point is a WGS84 coordinate.
type Point struct {
	Lat float64
	Lng float64
}

// Kitchen is a routing candidate.
type Kitchen struct {
	ID       uuid.UUID
	Code     string
	Name     string
	At       Point
	RadiusKM float64
	// HasPolygon reports that a service_area polygon exists. When it does it
	// OVERRIDES the radius (PROMPT §9.2), and PolygonCovers carries the result
	// of the PostGIS ST_Covers test the repository already ran.
	HasPolygon    bool
	PolygonCovers bool
	Priority      int
	Active        bool
	ServesSlot    bool
	OpenOnDate    bool
	// Capacity: MaxPortions 0 with Unlimited false means "full".
	Unlimited        bool
	MaxPortions      int
	ReservedPortions int
}

// Remaining portions before this kitchen is full.
func (k Kitchen) Remaining() int {
	if k.Unlimited {
		return math.MaxInt32
	}
	if r := k.MaxPortions - k.ReservedPortions; r > 0 {
		return r
	}
	return 0
}

// Covers reports whether the kitchen serves the point. A polygon, when present,
// is the whole answer: it exists precisely because the circle was wrong.
func (k Kitchen) Covers(p Point) bool {
	if k.HasPolygon {
		return k.PolygonCovers
	}
	return DistanceMeters(k.At, p) <= k.RadiusKM*1000
}

// Assignment is the routing decision, with its explanation.
type Assignment struct {
	KitchenID   uuid.UUID
	KitchenCode string
	KitchenName string
	DistanceM   int
	Mode        string // AUTO — MANUAL never comes from this package
	Reason      string
}

// Request is one routing question.
type Request struct {
	To          Point
	ServiceDate time.Time
	QtyNeeded   int
}

// Route picks the kitchen. Ranking, in order:
//
//  1. priority ascending  — the manual override, equal for everyone by default
//  2. distance ascending  — "nearest kitchen", which is the rule Steven gave
//  3. remaining capacity descending — spread load between equals
//  4. kitchen code        — so two identical candidates still resolve the same
//     way on every call, rather than depending on slice order
func Route(req Request, candidates []Kitchen) (Assignment, error) {
	qty := req.QtyNeeded
	if qty < 1 {
		qty = 1
	}

	type scored struct {
		k Kitchen
		d float64
	}
	var eligible []scored
	for _, k := range candidates {
		if !k.Active || !k.ServesSlot || !k.OpenOnDate {
			continue
		}
		if !k.Covers(req.To) {
			continue
		}
		if k.Remaining() < qty {
			continue
		}
		eligible = append(eligible, scored{k: k, d: DistanceMeters(k.At, req.To)})
	}
	if len(eligible) == 0 {
		return Assignment{}, ErrNotServiceable
	}

	sort.Slice(eligible, func(i, j int) bool {
		a, b := eligible[i], eligible[j]
		if a.k.Priority != b.k.Priority {
			return a.k.Priority < b.k.Priority
		}
		if a.d != b.d {
			return a.d < b.d
		}
		if a.k.Remaining() != b.k.Remaining() {
			return a.k.Remaining() > b.k.Remaining()
		}
		return a.k.Code < b.k.Code
	})

	win := eligible[0]
	reason := fmt.Sprintf("nearest covering kitchen, %.1f km", win.d/1000)
	if win.k.HasPolygon {
		reason = fmt.Sprintf("inside %s service area, %.1f km", win.k.Code, win.d/1000)
	}
	if len(eligible) > 1 && eligible[1].k.Priority > win.k.Priority {
		reason = fmt.Sprintf("priority kitchen, %.1f km", win.d/1000)
	}
	return Assignment{
		KitchenID:   win.k.ID,
		KitchenCode: win.k.Code,
		KitchenName: win.k.Name,
		DistanceM:   int(math.Round(win.d)),
		Mode:        "AUTO",
		Reason:      reason,
	}, nil
}

// Nearest returns the closest kitchen regardless of coverage. Used only to
// enrich an out-of-range log — "your nearest kitchen is 14 km away" tells
// operations something the bare rejection does not.
func Nearest(p Point, candidates []Kitchen) (Kitchen, int, bool) {
	var best Kitchen
	bestD := math.MaxFloat64
	found := false
	for _, k := range candidates {
		if !k.Active {
			continue
		}
		if d := DistanceMeters(k.At, p); d < bestD {
			best, bestD, found = k, d, true
		}
	}
	return best, int(math.Round(bestD)), found
}

// earthRadiusM is the mean radius. Over Jakarta-sized distances the error
// against a proper geodesic is metres, well inside a service radius expressed
// in whole kilometres.
const earthRadiusM = 6_371_008.8

// DistanceMeters is the great-circle distance between two points.
func DistanceMeters(a, b Point) float64 {
	φ1 := rad(a.Lat)
	φ2 := rad(b.Lat)
	Δφ := rad(b.Lat - a.Lat)
	Δλ := rad(b.Lng - a.Lng)

	s := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	return 2 * earthRadiusM * math.Asin(math.Min(1, math.Sqrt(s)))
}

func rad(deg float64) float64 { return deg * math.Pi / 180 }

// Envelope is the coarse sanity bound for a coordinate (docs/03 Q-11). It is a
// parameter, not a constant, so expanding beyond Jabodetabek is a settings
// change. Outside the envelope is an INPUT ERROR — a mis-signed longitude or a
// (0,0) — which is a different message from "not serviceable yet".
type Envelope struct {
	MinLat, MaxLat float64
	MinLng, MaxLng float64
}

// JabodetabekEnvelope is the seeded default.
var JabodetabekEnvelope = Envelope{MinLat: -6.60, MaxLat: -5.90, MinLng: 106.50, MaxLng: 107.10}

// Contains reports whether a point is plausible.
func (e Envelope) Contains(p Point) bool {
	return p.Lat >= e.MinLat && p.Lat <= e.MaxLat &&
		p.Lng >= e.MinLng && p.Lng <= e.MaxLng
}
