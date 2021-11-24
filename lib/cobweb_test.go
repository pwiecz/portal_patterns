package lib

import (
	"fmt"
	"testing"

	"github.com/golang/geo/s2"
)

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

func appendPortalsBetween(depth int, p Portal, ll s2.LatLng, portals []Portal) []Portal {
	diff := s2.PointFromLatLng(p.LatLng).Sub(s2.PointFromLatLng(ll).Vector)
	for i := 1; i < depth; i++ {
		midPoint := s2.PointFromLatLng(p.LatLng).Add(
			diff.Mul(float64(i) / float64(depth)))
		midLL := s2.LatLngFromPoint(s2.Point{midPoint})
		midPortal := Portal{Guid: p.Guid + fmt.Sprintf("%d", i), LatLng: midLL}
		portals = append(portals, midPortal)
	}
	return portals
}

func generateCobwebPortals(depth int) []Portal {
	ll0 := s2.LatLngFromDegrees(20, 20)
	ll1 := s2.LatLngFromDegrees(20, 22)
	ll2 := s2.LatLngFromDegrees(21, 21)
	portals := []Portal{
		Portal{Guid: "a", LatLng: ll0},
		Portal{Guid: "b", LatLng: ll1},
		Portal{Guid: "c", LatLng: ll2}}
	midLL := s2.LatLngFromDegrees((20+20+21)/3., (20+22+21)/3.)
	portals = appendPortalsBetween(depth, portals[0], midLL, portals)
	portals = appendPortalsBetween(depth, portals[1], midLL, portals)
	portals = appendPortalsBetween(depth, portals[2], midLL, portals)
	return portals
}

func TestCobwebSyntheticPortals(t *testing.T) {
	portals := generateCobwebPortals(10)
	res := LargestCobweb(portals, []int{}, func(int, int) {})
	checkValidCobwebResult(len(portals), res, t)
}

func benchmarkCobweb(depth int, b *testing.B) {
	portals := generateCobwebPortals(depth)
	for n := 0; n < b.N; n++ {
		res := LargestCobweb(portals, []int{}, func(int, int) {})
		if len(res) != len(portals) {
			panic(fmt.Sprintf("%d != %d: %v", len(res), len(portals), res))
		}
	}
}

func BenchmarkCobweb20(b *testing.B) { benchmarkCobweb(20, b) }
func BenchmarkCobweb30(b *testing.B) { benchmarkCobweb(30, b) }
func BenchmarkCobweb40(b *testing.B) { benchmarkCobweb(40, b) }
