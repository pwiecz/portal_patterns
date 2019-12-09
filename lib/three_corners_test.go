package lib

import "testing"

import "github.com/golang/geo/s2"

import "github.com/pwiecz/portal_patterns/lib/s2geo"

type indexedPoint struct {
	Index  int
	LatLng s2.Point
}

func isCorrectThreeCorner(p [3]indexedPoint, portals []indexedPoint) bool {
	if len(portals) == 0 {
		return true
	}
	triangle := s2geo.NewTriangleQuery(p[0].LatLng, p[1].LatLng, p[2].LatLng)
	if !triangle.ContainsPoint(portals[0].LatLng) {
		return false
	}
	p[portals[0].Index] = portals[0]
	return isCorrectThreeCorner(p, portals[1:])
}

func checkValidThreeCornerResult(expectedLength int, portals []IndexedPortal, t *testing.T) {
	if len(portals) != expectedLength {
		t.Errorf("Expected length %d, actual length %d", expectedLength, len(portals))
	}
	if portals[0].Index != 0 || portals[1].Index != 1 || portals[2].Index != 2 {
		t.Errorf("Result is not a correct three corner fielding")
	}
	indexedPoints := make([]indexedPoint, 0, len(portals))
	for _, portal := range portals {
		indexedPoints = append(indexedPoints, indexedPoint{
			Index:  portal.Index,
			LatLng: s2.PointFromLatLng(portal.Portal.LatLng),
		})
	}
	if !isCorrectThreeCorner([3]indexedPoint{indexedPoints[0], indexedPoints[1], indexedPoints[2]}, indexedPoints[3:]) {
		t.Errorf("Result is not a correct three corner fielding")
	}
}

func TestThreeCorner(t *testing.T) {
	portals0, err := ParseFile("testdata/portals_test_tc0.json")
	if err != nil {
		panic(err)
	}
	portals1, err := ParseFile("testdata/portals_test_tc1.json")
	if err != nil {
		panic(err)
	}
	portals2, err := ParseFile("testdata/portals_test_tc2.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals0) < 1 || len(portals1) < 1 || len(portals2) < 1 {
		t.FailNow()
	}
	threeCorner := LargestThreeCorner(portals0, portals1, portals2, func(int, int) {})
	checkValidThreeCornerResult(16, threeCorner, t)
}
