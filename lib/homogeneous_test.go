package lib

import "math"
import "testing"

import "github.com/golang/geo/s2"

import "github.com/pwiecz/portal_patterns/lib/s2geo"

var portals []Portal

func numPortalsPerDepth(depth uint16) int {
	return int(math.Pow(3, float64(depth-1)))/2 + 3
}
func isCorrectHomogeneous(p0, p1, p2 s2.Point, depth uint16, portals []s2.Point) ([]s2.Point, bool) {
	if depth == 1 {
		return portals, true
	}
	portal := portals[0]
	triangle := s2geo.NewTriangleQuery(p0, p1, p2)
	if !triangle.ContainsPoint(portal) {
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
	points := portalsToS2Points(portals)
	if _, ok := isCorrectHomogeneous(points[0], points[1], points[2], depth, points[3:]); !ok {
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
	result, depth := DeepestHomogeneous(portals, HomogeneousMaxDepth{6}, HomogeneousLargestArea{})
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
	result, depth := DeepestHomogeneous2(portals, HomogeneousMaxDepth{6}, HomogeneousLargestArea{})
	checkValidHomogeneousResult(5, result, depth, t)
}
