package lib

import "testing"

import "github.com/golang/geo/s2"

type indexedPortalData struct {
	Index int
	Portal portalData
}

func isCorrectThreeCorner(p [3]indexedPortalData, portals []indexedPortalData) bool {
	if len(portals) == 0 {
		return true
	}
	triangle := newTriangleQuery(p[0].Portal.LatLng, p[1].Portal.LatLng, p[2].Portal.LatLng)
	if !triangle.ContainsPoint(portals[0].Portal.LatLng) {
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
		t.Errorf("Result is not correct three corner fielding")
	}
	indexedPortalsData := make([]indexedPortalData, 0, len(portals))
	for i, portal := range portals {
		indexedPortalsData = append(indexedPortalsData, indexedPortalData{
			Index: portal.Index,
			Portal: portalData{
				Index: portalIndex(i),
				LatLng: s2.PointFromLatLng(portal.Portal.LatLng),
			},
		})
	}
	if !isCorrectThreeCorner([3]indexedPortalData{indexedPortalsData[0], indexedPortalsData[1], indexedPortalsData[2]}, indexedPortalsData[3:]) {
		t.Errorf("Result is not correct three corner fielding")
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
