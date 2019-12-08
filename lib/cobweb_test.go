package lib

import "testing"

func isCorrectCobweb(p0, p1, p2 portalData, portals []portalData) bool {
	if len(portals) == 0 {
		return true
	}
	triangle := newTriangleQuery(p0.LatLng, p1.LatLng, p2.LatLng)
	if !triangle.ContainsPoint(portals[0].LatLng) {
		return false
	}
	return isCorrectCobweb(p1, p2, portals[0], portals[1:])
}

func checkValidCobwebResult(expectedLength int, portals []Portal, t *testing.T) {
	if len(portals) != expectedLength {
		t.Errorf("Expected length %d, actual length %d", expectedLength, len(portals))
	}
	portalsData := portalsToPortalData(portals)
	if !isCorrectCobweb(portalsData[0], portalsData[1], portalsData[2], portalsData[3:]) {
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
