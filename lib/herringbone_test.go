package lib

import "testing"

import "github.com/golang/geo/s2"

func portalsToS2Points(portals []Portal) []s2.Point {
	result := make([]s2.Point, 0, len(portals))
	for _, portal := range portals {
		result = append(result, s2.PointFromLatLng(portal.LatLng))
	}
	return result
}
func isCorrectHerringbone(b0, b1 s2.Point, backbone []s2.Point) bool {
	if len(backbone) <= 1 {
		return true
	}
	triangle := NewS2TriangleQuery(b0, b1, backbone[0])
	if !triangle.ContainsPoint(backbone[1]) {
		return false
	}
	return isCorrectHerringbone(b0, b1, backbone[1:])
}

func checkValidHerringboneResult(expectedLength int, b0, b1 Portal, backbone []Portal, t *testing.T) {
	if len(backbone) != expectedLength {
		t.Errorf("Expected length %d, actual length %d", expectedLength, len(backbone))
	}
	backbonePoints := portalsToS2Points(backbone);
	if !isCorrectHerringbone(s2.PointFromLatLng(b0.LatLng), s2.PointFromLatLng(b1.LatLng), backbonePoints) {
		t.Errorf("Result is not correct herringbone fielding")
	}
}

func TestHerringboneMultiThreaded(t *testing.T) {
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
	b0, b1, backbone := LargestHerringbone(portals, []int{}, 6, func(int, int) {})
	checkValidHerringboneResult(19, b0, b1, backbone, t)
}

func TestHerringboneSingleThread(t *testing.T) {
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
	b0, b1, backbone := LargestHerringbone(portals, []int{}, 1, func(int, int) {})
	checkValidHerringboneResult(19, b0, b1, backbone, t)
}
