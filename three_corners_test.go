package main

import "testing"

func isCorrectThreeCorner(p [3]indexedPortal, portals []indexedPortal) bool {
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

func checkValidThreeCornerResult(expectedLength int, portals []indexedPortal, t *testing.T) {
	if len(portals) != expectedLength {
		t.Errorf("Expected length %d, actual length %d", expectedLength, len(portals))
	}
	if portals[0].Index != 0 || portals[1].Index != 1 || portals[2].Index != 2 {
		t.Errorf("Result is not correct three corner fielding")
	}
	if !isCorrectThreeCorner([3]indexedPortal{portals[0], portals[1], portals[2]}, portals[3:]) {
		t.Errorf("Result is not correct three corner fielding")
	}
}

func TestThreeCorner(t *testing.T) {
	portals0, err := ParseJSONFile("testdata/portals_test_tc0.json")
	if err != nil {
		panic(err)
	}
	portals1, err := ParseJSONFile("testdata/portals_test_tc1.json")
	if err != nil {
		panic(err)
	}
	portals2, err := ParseJSONFile("testdata/portals_test_tc2.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals0) < 1 || len(portals1) < 1 || len(portals2) < 1 {
		t.FailNow()
	}
	threeCorner := LargestThreeCorner(portals0, portals1, portals2)
	checkValidThreeCornerResult(16, threeCorner, t)
}
