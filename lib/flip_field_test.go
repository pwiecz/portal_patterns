package lib

import "testing"

import "github.com/golang/geo/s2"

func isCorrectFlipField(backbone, flipPortals []s2.Point) bool {
	if len(backbone) < 2 {
		return true
	}
	for i := 1; i+1 < len(backbone); i++ {
		if !s2.Sign(backbone[0], backbone[len(backbone)-1], backbone[i]) {
			return false
		}
	}
	for i := 1; i < len(backbone); i++ {
		for _, portal := range flipPortals {
			if !s2.Sign(backbone[i-1], backbone[i], portal) {
				return false
			}
		}
	}
	return true
}

func checkValidFlipFieldResult(expectedBackboneLength, expectedNumFlipPortals int, backbone, flipPortals []Portal, t *testing.T) {
	if len(backbone) != expectedBackboneLength {
		t.Errorf("Expected backbone length %d, actual length %d", expectedBackboneLength, len(backbone))
	}
	if len(flipPortals) != expectedNumFlipPortals {
		t.Errorf("Expected number of flip portals %d, actual number %d", expectedNumFlipPortals, len(flipPortals))
	}
	backboneData := portalsToS2Points(backbone)
	flipPortalData := portalsToS2Points(flipPortals)
	if !isCorrectFlipField(backboneData, flipPortalData) {
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
	backbone, flipPortals := LargestFlipField(portals, FlipFieldBackbonePortalLimit{8, EQUAL}, FlipFieldMaxFlipPortals{0}, FlipFieldNumWorkers{6})
	checkValidFlipFieldResult(8, 105, backbone, flipPortals, t)
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
	backbone, flipPortals := LargestFlipField(portals, FlipFieldBackbonePortalLimit{8, EQUAL}, FlipFieldMaxFlipPortals{0}, FlipFieldNumWorkers{1})
	checkValidFlipFieldResult(8, 105, backbone, flipPortals, t)
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
	backbone, flipPortals := LargestFlipField(portals, FlipFieldBackbonePortalLimit{16, LESS_EQUAL}, FlipFieldMaxFlipPortals{0}, FlipFieldNumWorkers{6})
	checkValidFlipFieldResult(9, 103, backbone, flipPortals, t)
}
