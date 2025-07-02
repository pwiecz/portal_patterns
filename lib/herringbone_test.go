package lib

import (
	"strconv"
	"testing"

	"github.com/golang/geo/s2"
)

func isCorrectHerringbone(b0, b1 s2.Point, backbone []portalData) bool {
	if len(backbone) <= 1 {
		return true
	}
	triangle := newTriangleQuery(b0, b1, backbone[0].LatLng)
	if !triangle.ContainsPoint(backbone[1].LatLng) {
		return false
	}
	return isCorrectHerringbone(b0, b1, backbone[1:])
}

func checkValidHerringboneResult(expectedLength int, b0, b1 Portal, backbone []Portal, t *testing.T) {
	if len(backbone) != expectedLength {
		t.Errorf("Expected length %d, actual length %d", expectedLength, len(backbone))
	}
	backboneData := portalsToPortalData(backbone)
	if !isCorrectHerringbone(s2.PointFromLatLng(b0.LatLng), s2.PointFromLatLng(b1.LatLng), backboneData) {
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

func generateHerringbonePortals(length int) []Portal {
	base0 := s2.LatLngFromDegrees(20, 20)
	base1 := s2.LatLngFromDegrees(20, 22)
	portals := []Portal{
		{Guid: "b0", LatLng: base0},
		{Guid: "b1", LatLng: base1}}
	lat := 20.01
	for i := range length {
		portals = append(portals, Portal{Guid: "bb" + strconv.Itoa(i), LatLng: s2.LatLngFromDegrees(lat, 21)})
		lat += 0.01
	}
	return portals
}

func TestHerringboneSyntheticPortals(t *testing.T) {
	portals := generateHerringbonePortals(30)
	b0, b1, backbone := LargestHerringbone(portals, []int{}, 1, func(int, int) {})
	checkValidHerringboneResult(30, b0, b1, backbone, t)
}

func benchmarkHerringbone(length int, b *testing.B) {
	portals := generateHerringbonePortals(length)
	for b.Loop() {
		LargestHerringbone(portals, []int{}, 1, func(int, int) {})
	}
}

func BenchmarkHerringbone50(b *testing.B)  { benchmarkHerringbone(50, b) }
func BenchmarkHerringbone100(b *testing.B) { benchmarkHerringbone(100, b) }
func BenchmarkHerringbone150(b *testing.B) { benchmarkHerringbone(150, b) }
