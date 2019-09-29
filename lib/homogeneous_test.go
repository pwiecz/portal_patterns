package lib

import "math"
import "testing"

var portals []Portal

func numPortalsPerDepth(depth uint16) int {
	return int(math.Pow(3, float64(depth-1)))/2 + 3
}
func isCorrectHomogeneous(p0, p1, p2 Portal, depth uint16, portals []Portal) ([]Portal, bool) {
	if depth == 1 {
		return portals, true
	}
	portal := portals[0]
	triangle := newTriangleQuery(p0.LatLng, p1.LatLng, p2.LatLng)
	if !triangle.ContainsPoint(portal.LatLng) {
		return portals, false
	}
	var res bool
	if portals, res = isCorrectHomogeneous(portal, p1, p2, depth-1, portals[1:]); !res {
		return portals, false
	}
	if portals, res = isCorrectHomogeneous(p0, portal, p2, depth-1, portals); !res {
		return portals, false
	}
	if portals, res = isCorrectHomogeneous(p0, p1, portal, depth-1, portals); !res {
		return portals, false
	}
	return portals, true
}
func checkValidHomogeneousResult(expectedDepth uint16, portals []Portal, depth uint16, t *testing.T) {
	if depth != expectedDepth {
		t.Errorf("Expected depth %d, actual depth %d", expectedDepth, depth)
	}
	if numPortalsPerDepth(depth) != len(portals) {
		t.Errorf("Expected %d portals for depth %d, got %d portals", numPortalsPerDepth(depth), depth, len(portals))
		return
	}
	if _, ok := isCorrectHomogeneous(portals[0], portals[1], portals[2], depth, portals[3:]); !ok {
		t.Errorf("Result is not correct homogeneous fielding")
	}
}
func TestHomogeneous(t *testing.T) {
	portals, err := ParseFile("testdata/portals_test.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals) < 3 {
		t.FailNow()
	}
	scorer := largestTriangleScorer{}
	result, depth := DeepestHomogeneous(portals, 6, scorer, func(int, int) {})
	checkValidHomogeneousResult(5, result, depth, t)
}

func TestHomogeneousPretty(t *testing.T) {
	portals, err := ParseFile("testdata/portals_test.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals) < 3 {
		t.FailNow()
	}
	scorer := newThickTrianglesScorer(len(portals))
	result, depth := DeepestHomogeneous2(portals, 6, scorer, scorer, func(int, int) {})
	checkValidHomogeneousResult(5, result, depth, t)
}
