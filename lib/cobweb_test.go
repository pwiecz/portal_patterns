package lib

import "testing"

import "github.com/golang/geo/s2"

func isCorrectCobweb(p0, p1, p2 s2.Point, portals []s2.Point) bool {
	if len(portals) == 0 {
		return true
	}
	triangle := NewS2TriangleQuery(p0, p1, p2)
	if !triangle.ContainsPoint(portals[0]) {
		return false
	}
	return isCorrectCobweb(p1, p2, portals[0], portals[1:])
}

func checkValidCobwebResult(expectedLength int, portals []Portal, t *testing.T) {
	if len(portals) != expectedLength {
		t.Errorf("Expected length %d, actual length %d", expectedLength, len(portals))
	}
	points := portalsToS2Points(portals)
	if !isCorrectCobweb(points[0], points[1], points[2], points[3:]) {
		t.Errorf("Result is not correct cobweb fielding")
	}
}

func TestCobweb(t *testing.T) {
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
	cobweb := LargestCobweb(portals, []int{}, func(int, int) {})
	checkValidCobwebResult(22, cobweb, t)
}
