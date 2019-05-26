package main

import "testing"

func isCorrectThreeCorner(p0, p1, p2 indexedPortal, portals []indexedPortal) bool {
	if len(portals) == 0 {
		return true
	}
	if p0.Index == p1.Index || p0.Index == p2.Index || p1.Index == p2.Index {
		return false
	}
	triangle := newTriangleQuery(p0.Portal.LatLng, p1.Portal.LatLng, p2.Portal.LatLng)
	if !triangle.ContainsPoint(portals[0].Portal.LatLng) {
		return false
	}
	switch portals[0].Index {
	case 0:
		return isCorrectThreeCorner(portals[0], p1, p2, portals[1:])
	case 1:
		return isCorrectThreeCorner(p0, portals[0], p2, portals[1:])
	case 2:
		return isCorrectThreeCorner(p0, p1, portals[0], portals[1:])
	default:
		return false
	}

}

func checkValidThreeCornerResult(expectedLength int, portals []indexedPortal, t *testing.T) {
	if len(portals) != expectedLength {
		t.Errorf("Expected length %d, actual length %d", expectedLength, len(portals))
	}
	if !isCorrectThreeCorner(portals[0], portals[1], portals[2], portals[3:]) {
		t.Errorf("Result is not correct three corner fielding")
	}
}

func TestThreeCorner(t *testing.T) {
	portals0, err := ParseJSONFile("portals_test_tc0.json")
	if err != nil {
		panic(err)
	}
	portals1, err := ParseJSONFile("portals_test_tc1.json")
	if err != nil {
		panic(err)
	}
	portals2, err := ParseJSONFile("portals_test_tc2.json")
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
	checkValidThreeCornerResult(13, threeCorner, t)
}
