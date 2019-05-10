package main

import "fmt"
import "strings"

import "github.com/golang/geo/r3"
import "github.com/golang/geo/s2"

type portalData struct {
	Index  int
	LatLng s2.Point
}

type indexedPortal struct {
	Index  int
	Portal Portal
}

func portalsToPortalData(portals []Portal) []portalData {
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{Index: i, LatLng: portal.LatLng})
	}
	return portalsData
}

type bestSolution struct {
	Index  int
	Length int
}

type triangleQuery struct {
	a, b, c                   s2.Point
	aCrossB, cCrossA, bCrossC r3.Vector
}

func newTriangleQuery(a, b, c s2.Point) triangleQuery {
	if !s2.Sign(a, b, c) {
		a, c = c, a
	}
	aCrossB := a.Cross(b.Vector)
	cCrossA := c.Cross(a.Vector)
	bCrossC := b.Cross(c.Vector)
	return triangleQuery{a, b, c, aCrossB, cCrossA, bCrossC}
}

func sign(aCrossB r3.Vector, c s2.Point) bool {
	return aCrossB.Dot(c.Vector) > 0
}

func (t *triangleQuery) ContainsPoint(o s2.Point) bool {
	if sign(t.aCrossB, o) && sign(t.cCrossA, o) && sign(t.bCrossC, o) {
		return true
	}
	return false
}

type orderedCCWQuery struct {
	a, o    s2.Point
	aocSign bool
	cCrossO r3.Vector
}

func newOrderedCCWQuery(a, c, o s2.Point) orderedCCWQuery {
	////

	////
	aCrossO := a.Cross(o.Vector)
	aocSign := sign(aCrossO, c)
	cCrossO := c.Cross(o.Vector)
	return orderedCCWQuery{a, o, aocSign, cCrossO}
}

func (t *orderedCCWQuery) Ordered(b s2.Point) bool {
	if t.aocSign {
		return sign(b.Cross(t.o.Vector), t.a) ||
			sign(t.cCrossO, b)
	}
	return sign(b.Cross(t.o.Vector), t.a) &&
		sign(t.cCrossO, b)
}

// triangleWedgeQuery helps to answer question wether a point is contained
// inside a wedge which is a contained between semilines ab and ac, where angle
// between ab and ac is < pi.
type triangleWedgeQuery struct {
	ccwQuery orderedCCWQuery
}

func newTriangleWedgeQuery(a, b, c s2.Point) triangleWedgeQuery {
	if sign(a.Cross(b.Vector), c) {
		return triangleWedgeQuery{newOrderedCCWQuery(b, c, a)}
	}
	return triangleWedgeQuery{newOrderedCCWQuery(c, b, a)}
}

func (t *triangleWedgeQuery) ContainsPoint(o s2.Point) bool {
	return t.ccwQuery.Ordered(o)
}

func keepOnlyContainedPortals(portals []portalData, triangle triangleQuery) []portalData {
	filteredPortals := portals
	for i := 0; i < len(filteredPortals); {
		portal := filteredPortals[i]
		if portal.LatLng == triangle.a || portal.LatLng == triangle.b || portal.LatLng == triangle.c ||
			!triangle.ContainsPoint(portal.LatLng) {
			filteredPortals[i], filteredPortals[len(filteredPortals)-1] = filteredPortals[len(filteredPortals)-1], filteredPortals[i]
			filteredPortals = filteredPortals[:len(filteredPortals)-1]
			continue
		}
		i++
	}
	return filteredPortals
}

func removeContainedPortals(portals []portalData, ix int, triangle triangleQuery) []portalData {
	for ix < len(portals) {
		if triangle.ContainsPoint(portals[ix].LatLng) {
			portals[ix], portals[len(portals)-1] = portals[len(portals)-1], portals[ix]
			portals = portals[:len(portals)-1]
		} else {
			ix++
		}
	}
	return portals
}

func pointToJSONCoords(point s2.Point) string {
	latlng := s2.LatLngFromPoint(point)
	return fmt.Sprintf(`{"lat":%f,"lng":%f}`, latlng.Lat.Degrees(), latlng.Lng.Degrees())
}
func polylineFromPortalList(portals []Portal) string {
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

func printProgressBar(done int, total int) {
	const maxWidth = 50
	doneWidth := done * maxWidth / total
	var b strings.Builder
	b.WriteString("\r[")
	for i := 1; i < doneWidth; i++ {
		b.WriteRune('=')
	}
	b.WriteRune('>')
	for i := doneWidth; i < maxWidth; i++ {
		b.WriteRune(' ')
	}
	percent := 100. * float32(done) / float32(total)
	b.WriteString(fmt.Sprintf(" ] %3.2f%% (%d/%d)", percent, done, total))
	fmt.Print(b.String())
}
