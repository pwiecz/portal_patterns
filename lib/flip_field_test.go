package lib

import (
	"testing"

	"github.com/golang/geo/s2"
)

func isCorrectCCWFlipField(backbone, flipPortals []portalData, simpleBackbone bool) bool {
	if len(backbone) < 2 {
		return true
	}
	for i := 1; i+1 < len(backbone); i++ {
		if !s2.Sign(backbone[0].LatLng, backbone[len(backbone)-1].LatLng, backbone[i].LatLng) {
			return false
		}
	}
	for i := 1; i < len(backbone); i++ {
		for _, portal := range flipPortals {
			if !s2.Sign(backbone[i-1].LatLng, backbone[i].LatLng, portal.LatLng) {
				return false
			}
		}
		if simpleBackbone && i > 1 && s2.Sign(backbone[0].LatLng, backbone[i-1].LatLng, backbone[i].LatLng) {
			return false
		}
	}
	return true
}
func isCorrectCWFlipField(backbone, flipPortals []portalData, simpleBackbone bool) bool {
	if len(backbone) < 2 {
		return true
	}
	for i := 1; i+1 < len(backbone); i++ {
		if s2.Sign(backbone[0].LatLng, backbone[len(backbone)-1].LatLng, backbone[i].LatLng) {
			return false
		}
	}
	for i := 1; i < len(backbone); i++ {
		for _, portal := range flipPortals {
			if s2.Sign(backbone[i-1].LatLng, backbone[i].LatLng, portal.LatLng) {
				return false
			}
		}
		if simpleBackbone && i > 1 && !s2.Sign(backbone[0].LatLng, backbone[i-1].LatLng, backbone[i].LatLng) {
			return false
		}
	}
	return true
}

func checkValidFlipFieldResult(expectedBackboneLength, expectedNumFlipPortals int, simpleBackbone bool, backbone, flipPortals []Portal, t *testing.T) {
	if len(backbone) != expectedBackboneLength {
		t.Errorf("Expected backbone length %d, actual length %d", expectedBackboneLength, len(backbone))
	}
	if len(flipPortals) != expectedNumFlipPortals {
		t.Errorf("Expected number of flip portals %d, actual number %d", expectedNumFlipPortals, len(flipPortals))
	}
	backboneData := portalsToPortalData(backbone)
	flipPortalData := portalsToPortalData(flipPortals)
	if !isCorrectCCWFlipField(backboneData, flipPortalData, simpleBackbone) && !isCorrectCWFlipField(backboneData, flipPortalData, simpleBackbone) {
		t.Errorf("Result is not correct flip fielding")
	}
}

func TestFlipFieldMultiThreaded(t *testing.T) {
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
	backbone, flipPortals := LargestFlipField(portals, FlipFieldBackbonePortalLimit{8, EQUAL}, FlipFieldMaxFlipPortals(0), FlipFieldNumWorkers(6))
	checkValidFlipFieldResult(8, 105, false, backbone, flipPortals, t)
}

func TestFlipFieldSingleThread(t *testing.T) {
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
	backbone, flipPortals := LargestFlipField(portals, FlipFieldBackbonePortalLimit{8, EQUAL}, FlipFieldMaxFlipPortals(0), FlipFieldNumWorkers(1))
	checkValidFlipFieldResult(8, 105, false, backbone, flipPortals, t)
}

func TestFlipFieldLessEqual(t *testing.T) {
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
	backbone, flipPortals := LargestFlipField(portals, FlipFieldBackbonePortalLimit{16, LESS_EQUAL}, FlipFieldMaxFlipPortals(0), FlipFieldNumWorkers(6))
	checkValidFlipFieldResult(9, 103, false, backbone, flipPortals, t)
}

func TestFlipFieldSimple(t *testing.T) {
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
	backbone, flipPortals := LargestFlipField(portals, FlipFieldBackbonePortalLimit{8, EQUAL}, FlipFieldMaxFlipPortals(0), FlipFieldNumWorkers(6), FlipFieldSimpleBackbone(true))
	checkValidFlipFieldResult(8, 105, true, backbone, flipPortals, t)
}
