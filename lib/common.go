package lib

import (
	"fmt"
	"math"
	"strings"

	"github.com/golang/geo/s2"
	"golang.org/x/exp/constraints"
)

type portalIndex uint16

const invalidPortalIndex portalIndex = math.MaxUint16

type portalData struct {
	Index  portalIndex
	LatLng s2.Point
}

func portalsToPortalData(portals []Portal) []portalData {
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{
			Index:  portalIndex(i),
			LatLng: s2.PointFromLatLng(portal.LatLng)})
	}
	return portalsData
}

const invalidLength uint16 = math.MaxUint16

type bestSolution struct {
	Index  portalIndex
	Length uint16
}

//lint:ignore U1000 We keep some unused functions that may be potentially useful
func portalsInsideWedge(portals []portalData, a, b, c portalData, result []portalData) []portalData {
	wedge := newTriangleWedgeQuery(a.LatLng, b.LatLng, c.LatLng)
	result = result[:0]
	for _, p := range portals {
		if p.Index != a.Index && p.Index != b.Index && p.Index != c.Index &&
			wedge.ContainsPoint(p.LatLng) {
			result = append(result, p)
		}
	}
	return result
}

// returns a subset of portals from portals that lie inside wedge ab, ac.
// It reorders the input portals slice and returns its subslice
func partitionPortalsInsideWedge(portals []portalData, a, b, c portalData) []portalData {
	wedge := newTriangleWedgeQuery(a.LatLng, b.LatLng, c.LatLng)
	length := partition(portals, func(p portalData) bool {
		return p.Index != a.Index && p.Index != b.Index && p.Index != c.Index &&
			wedge.ContainsPoint(p.LatLng)
	})
	return portals[:length]
}

func portalsInsideTriangle(portals []portalData, a, b, c portalData, result []portalData) []portalData {
	triangle := newTriangleQuery(a.LatLng, b.LatLng, c.LatLng)
	result = result[:0]
	for _, p := range portals {
		if p.Index != a.Index && p.Index != b.Index && p.Index != c.Index &&
			triangle.ContainsPoint(p.LatLng) {
			result = append(result, p)
		}
	}
	return result
}

func min[T constraints.Ordered](v0, v1 T) T {
	if v0 < v1 {
		return v0
	}
	return v1
}
func max[T constraints.Ordered](v0, v1 T) T {
	if v0 > v1 {
		return v0
	}
	return v1
}

// partition moves elements that do not satisfy the f predicate to the end of the slice
// (swaps them with the elements at the end).
// Returns number of elements that satisfy the predicate
// Does not preserve relative order of elements.
func partition[S ~[]E, E any](s S, f func(E) bool) int {
	length := len(s)
	for i := 0; i < length; {
		if !f(s[i]) {
			s[i], s[length-1] = s[length-1], s[i]
			length--
		} else {
			i++
		}
	}
	return length
}

func hasAllElementsInTheTriple[T comparable](indices []T, a, b, c T) bool {
	for _, index := range indices {
		if a != index && b != index && c != index {
			return false
		}
	}
	return true
}
func hasAllElementsInThePair[T comparable](indices []T, a, b T) bool {
	for _, index := range indices {
		if a != index && b != index {
			return false
		}
	}
	return true
}

//lint:ignore U1000 We keep some unused functions that may be potentially useful
func pointToJSONCoords(point s2.Point) string {
	return latLngToJSONCoords(s2.LatLngFromPoint(point))
}
func latLngToJSONCoords(latLng s2.LatLng) string {
	return fmt.Sprintf(`{"lat":%f,"lng":%f}`, latLng.Lat.Degrees(), latLng.Lng.Degrees())
}
func PolylineFromPortalList(portals []Portal) string {
	var json strings.Builder
	json.WriteString(`{"type":"polyline","latLngs":[`)
	if len(portals) > 0 {
		fmt.Fprint(&json, latLngToJSONCoords(portals[0].LatLng))
		for _, portal := range portals[1:] {
			fmt.Fprintf(&json, ",%s", latLngToJSONCoords(portal.LatLng))
		}
	}
	json.WriteString(`],"color":"#a24ac3"}`)
	return json.String()
}
func MarkersFromPortalList(portals []Portal) string {
	var json strings.Builder
	for i, portal := range portals {
		if i > 0 {
			fmt.Fprintf(&json, ", ")
		}
		fmt.Fprintf(&json, `{"type":"marker","latLng":`)
		fmt.Fprintf(&json, "%s", latLngToJSONCoords(portal.LatLng))
		fmt.Fprintf(&json, `,"color":"#a24ac3"}`)
	}
	return json.String()
}

func PrintProgressBar(done int, total int) {
	const maxWidth = 50
	doneWidth := done * maxWidth / total
	var b strings.Builder
	b.WriteString("\r[")
	for i := 1; i < doneWidth; i++ {
		b.WriteRune('=')
	}
	if done < total {
		b.WriteRune('>')
	} else {
		b.WriteRune('=')
	}
	for i := doneWidth; i < maxWidth; i++ {
		b.WriteRune(' ')
	}
	percent := 100. * float32(done) / float32(total)
	b.WriteString(fmt.Sprintf("] %3.1f%% (%d/%d)", percent, done, total))
	fmt.Print(b.String())
}
