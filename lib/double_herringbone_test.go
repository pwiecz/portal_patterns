package lib

import (
	"strconv"
	"testing"

	"github.com/golang/geo/s2"
)

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

func generateDoubleHerringbonePortals(length0, length1 int) []Portal {
	base0 := s2.LatLngFromDegrees(20, 20)
	base1 := s2.LatLngFromDegrees(20, 22)
	portals := []Portal{
		{Guid: "b0", LatLng: base0},
		{Guid: "b1", LatLng: base1}}
	lat := 20.01
	for i := range length0 {
		portals = append(portals, Portal{Guid: "bb0" + strconv.Itoa(i), LatLng: s2.LatLngFromDegrees(lat, 21)})
		lat += 0.01
	}
	lat = 19.99
	for i := range length1 {
		portals = append(portals, Portal{Guid: "bb1" + strconv.Itoa(i), LatLng: s2.LatLngFromDegrees(lat, 21)})
		lat -= 0.01
	}
	return portals
}

func TestDoubleHerringboneSyntheticPortals(t *testing.T) {
	portals := generateDoubleHerringbonePortals(25, 30)
	b0, b1, backbone0, backbone1 := LargestDoubleHerringbone(portals, []int{}, 1, func(int, int) {})
	checkValidHerringboneResult(25, b0, b1, backbone0, t)
	checkValidHerringboneResult(30, b0, b1, backbone1, t)
}

func benchmarkDoubleHerringbone(length0, length1 int, b *testing.B) {
	portals := generateDoubleHerringbonePortals(length0, length1)
	for b.Loop() {
		LargestDoubleHerringbone(portals, []int{}, 1, func(int, int) {})
	}
}

func BenchmarkDoubleHerringbone20_30(b *testing.B) { benchmarkDoubleHerringbone(20, 30, b) }
func BenchmarkDoubleHerringbone40_60(b *testing.B) { benchmarkDoubleHerringbone(40, 60, b) }
func BenchmarkDoubleHerringbone90_60(b *testing.B) { benchmarkDoubleHerringbone(90, 60, b) }
