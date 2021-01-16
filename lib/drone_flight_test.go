package lib

import (
	"math"
	"testing"

	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

func portalIsOnList(portal Portal, list []Portal) bool {
	for _, p := range list {
		if portal.Guid == p.Guid {
			return true
		}
	}
	return false
}
func isCorrectDroneFlight(route, keys []Portal) bool {
	if len(route) <= 1 {
		return true
	}
	secondPortalCellId := s2.CellIDFromLatLng(route[1].LatLng)
	if secondPortalCellId.Level() < 16 {
		panic(secondPortalCellId.Level())
	}
	secondPortalCellLvl16 := s2.CellFromCellID(secondPortalCellId.Parent(16))
	distance := secondPortalCellLvl16.Distance(s2.PointFromLatLng(route[0].LatLng)).Angle()
	if distance > s1.Angle(1250/RadiansToMeters) {
		return false
	}
	if distance > s1.Angle(500/RadiansToMeters) && !portalIsOnList(route[1], keys) {
		return false
	}

	return isCorrectDroneFlight(route[1:], keys)
}

func checkValidDroneFlight(expectedLength float64, route, keys []Portal, t *testing.T) {
	if len(route) < 2 {
		t.Errorf("Expected at least 2 portals in route, got %d", len(route))
	}
	if !isCorrectDroneFlight(route, keys) {
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
	route, keys := LongestDroneFlight(portals, DroneFlightNumWorkers(1))
	checkValidDroneFlight(542.555248, route, keys, t)
}

func TestDroneFlightLeastJumps(t *testing.T) {
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
	route, keys := LongestDroneFlight(portals, DroneFlightNumWorkers(1), DroneFlightLeastJumps{})
	checkValidDroneFlight(542.555248, route, keys, t)
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
	route, keys := LongestDroneFlight(portals, DroneFlightStartPortalIndex(1), DroneFlightNumWorkers(6))
	if route[0].Guid != portals[1].Guid {
		t.Errorf("Expected %s as first route portal, got %s", portals[1].Guid, route[0].Guid)
	}
	checkValidDroneFlight(354.740861, route, keys, t)
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
	route, keys := LongestDroneFlight(portals, DroneFlightEndPortalIndex(2), DroneFlightNumWorkers(6))
	if route[len(route)-1].Guid != portals[2].Guid {
		t.Errorf("Expected %s as last route portal, got %s", portals[2].Guid, route[len(route)-1].Guid)
	}
	checkValidDroneFlight(475.863679, route, keys, t)
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
	route, keys := LongestDroneFlight(portals, DroneFlightStartPortalIndex(3), DroneFlightEndPortalIndex(4), DroneFlightNumWorkers(6))
	if route[0].Guid != portals[3].Guid {
		t.Errorf("Expected %s as first route portal, got %s", portals[3].Guid, route[0].Guid)
	}
	if route[len(route)-1].Guid != portals[4].Guid {
		t.Errorf("Expected %s as last route portal, got %s", portals[4].Guid, route[len(route)-1].Guid)
	}
	checkValidDroneFlight(139.842564, route, keys, t)
}
