package lib

import "testing"

import "github.com/golang/geo/s2"

func latLngSign(l0, l1, l2 s2.LatLng) bool {
	return s2.Sign(s2.PointFromLatLng(l0), s2.PointFromLatLng(l1), s2.PointFromLatLng(l2))
}

func TestHerringboneDoubleMultiThreaded(t *testing.T) {
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
	b0, b1, backbone0, backbone1 := LargestDoubleHerringbone(portals, []int{}, 6, func(int, int) {})
	checkValidHerringboneResult(14, b0, b1, backbone0, t)
	checkValidHerringboneResult(16, b0, b1, backbone1, t)
	if !latLngSign(backbone0[0].LatLng, b0.LatLng, b1.LatLng) {
		t.Errorf("Incorrect orientation of first herringbone backbone")
	}
	if latLngSign(backbone1[0].LatLng, b0.LatLng, b1.LatLng) {
		t.Errorf("Incorrect orientation of second herringbone backbone")
	}
}

func TestHerringboneDoubleSingleThread(t *testing.T) {
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
	b0, b1, backbone0, backbone1 := LargestDoubleHerringbone(portals, []int{}, 1, func(int, int) {})
	checkValidHerringboneResult(14, b0, b1, backbone0, t)
	checkValidHerringboneResult(16, b0, b1, backbone1, t)
	if !latLngSign(backbone0[0].LatLng, b0.LatLng, b1.LatLng) {
		t.Errorf("Incorrect orientation of first herringbone backbone")
	}
	if latLngSign(backbone1[0].LatLng, b0.LatLng, b1.LatLng) {
		t.Errorf("Incorrect orientation of second herringbone backbone")
	}
}
