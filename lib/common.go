package lib

import "fmt"
import "math"
import "strings"

import "github.com/golang/geo/s2"

type portalIndex uint16

const invalidPortalIndex portalIndex = math.MaxUint16

type portalData struct {
	Index  portalIndex
	LatLng s2.Point
}

func portalsToPortalData(portals []Portal) []portalData {
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{Index: portalIndex(i), LatLng: portal.LatLng})
	}
	return portalsData
}

const invalidLength uint16 = math.MaxUint16

type bestSolution struct {
	Index  portalIndex
	Length uint16
}

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
func partitionPortalsInsideWedge(portals []portalData, a, b, c portalData) []portalData {
	wedge := newTriangleWedgeQuery(a.LatLng, b.LatLng, c.LatLng)
	length := len(portals)
	for i := 0; i < length; {
		p := portals[i]
		if p.Index != a.Index && p.Index != b.Index && p.Index != c.Index &&
			wedge.ContainsPoint(p.LatLng) {
			i++
		} else {
			portals[i], portals[length-1] = portals[length-1], portals[i]
			length--
		}
	}
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

func float64Min(v0, v1 float64) float64 {
	if v0 < v1 {
		return v0
	}
	return v1
}
func float64Max(v0, v1 float64) float64 {
	if v0 > v1 {
		return v0
	}
	return v1
}
func float32Min(v0, v1 float32) float32 {
	if v0 < v1 {
		return v0
	}
	return v1
}

func hasAllIndicesInTheTriple(indices []int, a, b, c int) bool {
	for _, index := range indices {
		if a != index && b != index && c != index {
			return false
		}
	}
	return true
}

func hasAllIndicesInThePair(indices []int, a, b int) bool {
	for _, index := range indices {
		if a != index && b != index {
			return false
		}
	}
	return true
}

func pointToJSONCoords(point s2.Point) string {
	latlng := s2.LatLngFromPoint(point)
	return fmt.Sprintf(`{"lat":%f,"lng":%f}`, latlng.Lat.Degrees(), latlng.Lng.Degrees())
}
func PolylineFromPortalList(portals []Portal) string {
	var json strings.Builder
	json.WriteString(`{"type":"polyline","latLngs":[`)
	if len(portals) > 0 {
		fmt.Fprint(&json, pointToJSONCoords(portals[0].LatLng))
		for _, portal := range portals[1:] {
			fmt.Fprintf(&json, ",%s", pointToJSONCoords(portal.LatLng))
		}
	}
	json.WriteString(`],"color":"#a24ac3"}`)
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
