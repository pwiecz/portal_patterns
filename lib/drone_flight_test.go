package lib

import "math"
import "testing"

import "github.com/golang/geo/s1"
import "github.com/golang/geo/s2"

func isCorrectDroneFlight(route []Portal) bool {
	if len(route) <= 1 {
		return true
	}
	secondPortalCellId := s2.CellIDFromLatLng(route[1].LatLng)
	if secondPortalCellId.Level() < 16 {
		panic(secondPortalCellId.Level())
	}
	secondPortalCellLvl16 := s2.CellFromCellID(secondPortalCellId.Parent(16))
	distance := secondPortalCellLvl16.Distance(s2.PointFromLatLng(route[0].LatLng)).Angle()
	if distance > s1.Angle(500/RadiansToMeters) {
		return false
	}
	return isCorrectDroneFlight(route[1:])
}

func checkValidDroneFlight(expectedLength float64, route []Portal, t *testing.T) {
	if len(route) < 2 {
		t.Errorf("Expected at least 2 portals in route, got %d", len(route))
	}
	if !isCorrectDroneFlight(route) {
		t.Errorf("Result is not a correct drone flight route")
	}
	routeLengthRadians := route[0].LatLng.Distance(route[len(route)-1].LatLng)
	routeLengthM := float64(routeLengthRadians) * RadiansToMeters
	if math.Abs(expectedLength-routeLengthM) > 0.0001 {
		t.Errorf("Expected length %f, actual length %f", expectedLength, routeLengthM)
	}
}

func TestDroneFlight(t *testing.T) {
	portals, err := ParseFile("testdata/portals_test.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals) < 2 {
		t.FailNow()
	}
	route := LongestDroneFlight(portals, -1, -1, func(int, int) {})
	checkValidDroneFlight(542.555248, route, t)
}

func TestDroneFlightFrom(t *testing.T) {
	portals, err := ParseFile("testdata/portals_test.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals) < 2 {
		t.FailNow()
	}
	route := LongestDroneFlight(portals, 1, -1, func(int, int) {})
	if route[0].Guid != portals[1].Guid {
		t.Errorf("Expected %s as first route portal, got %s", portals[1].Guid, route[0].Guid)
	}
	checkValidDroneFlight(354.740861, route, t)
}

func TestDroneFlightTo(t *testing.T) {
	portals, err := ParseFile("testdata/portals_test.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals) < 2 {
		t.FailNow()
	}
	route := LongestDroneFlight(portals, -1, 2, func(int, int) {})
	if route[len(route)-1].Guid != portals[2].Guid {
		t.Errorf("Expected %s as last route portal, got %s", portals[2].Guid, route[len(route)-1].Guid)
	}
	checkValidDroneFlight(475.863679, route, t)
}

func TestDroneFlightFromTo(t *testing.T) {
	portals, err := ParseFile("testdata/portals_test.json")
	if err != nil {
		panic(err)
	}
	if testing.Short() {
		t.Skip()
	}
	if len(portals) < 2 {
		t.FailNow()
	}
	route := LongestDroneFlight(portals, 3, 4, func(int, int) {})
	if route[0].Guid != portals[3].Guid {
		t.Errorf("Expected %s as first route portal, got %s", portals[3].Guid, route[0].Guid)
	}
	if route[len(route)-1].Guid != portals[4].Guid {
		t.Errorf("Expected %s as last route portal, got %s", portals[4].Guid, route[len(route)-1].Guid)
	}
	checkValidDroneFlight(139.842564, route, t)
}
