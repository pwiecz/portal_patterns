package lib

import (
	"math"
	"testing"

	"github.com/golang/geo/s2"
)

func numPortalsPerDepth(depth uint16) int {
	return int(math.Pow(3, float64(depth-1)))/2 + 3
}
func isCorrectHomogeneous(p0, p1, p2 portalData, depth uint16, portals []portalData) ([]portalData, bool) {
	if depth == 1 {
		return portals, true
	}
	portal := portals[0]
	triangle := newTriangleQuery(p0.LatLng, p1.LatLng, p2.LatLng)
	if !triangle.ContainsPoint(portal.LatLng) {
		return portals, false
	}
	var res bool
	if portals, res = isCorrectHomogeneous(portal, p1, p2, depth-1, portals[1:]); !res {
		return portals, false
	}
	if portals, res = isCorrectHomogeneous(p0, portal, p2, depth-1, portals); !res {
		return portals, false
	}
	if portals, res = isCorrectHomogeneous(p0, p1, portal, depth-1, portals); !res {
		return portals, false
	}
	return portals, true
}
func checkValidHomogeneousResult(expectedDepth uint16, result []Portal, depth uint16, t *testing.T) {
	if depth != expectedDepth {
		t.Errorf("Expected depth %d, actual depth %d", expectedDepth, depth)
	}
	if numPortalsPerDepth(depth) != len(result) {
		t.Errorf("Expected %d portals for depth %d, got %d portals", numPortalsPerDepth(depth), depth, len(result))
		return
	}
	portalsData := portalsToPortalData(result)
	if _, ok := isCorrectHomogeneous(portalsData[0], portalsData[1], portalsData[2], depth, portalsData[3:]); !ok {
		t.Errorf("Result is not correct homogeneous fielding")
	}
}
func checkValidPureHomogeneousResult(expectedDepth uint16, result []Portal, depth uint16, allPortals []Portal, t *testing.T) {
	checkValidHomogeneousResult(expectedDepth, result, depth, t)

	triangle := newTriangleQuery(
		s2.PointFromLatLng(result[0].LatLng),
		s2.PointFromLatLng(result[1].LatLng),
		s2.PointFromLatLng(result[2].LatLng))
	numPortalsInTriangle := 0
	for _, p := range allPortals {
		if p.Guid == result[0].Guid ||
			p.Guid == result[1].Guid ||
			p.Guid == result[2].Guid ||
			triangle.ContainsPoint(s2.PointFromLatLng(p.LatLng)) {
			numPortalsInTriangle++
		}
	}
	if numPortalsPerDepth(depth) != numPortalsInTriangle {
		t.Errorf("Not all portals used. There are %d portals in the area, only %d used",
			numPortalsInTriangle, numPortalsPerDepth(depth))
	}
}

func TestHomogeneous(t *testing.T) {
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
	result, depth := DeepestHomogeneous(portals, HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousNumWorkers(6))
	checkValidHomogeneousResult(5, result, depth, t)
}

func TestHomogeneousPure(t *testing.T) {
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
	result, depth := DeepestHomogeneous(portals, HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousPure(true), HomogeneousNumWorkers(6))
	checkValidPureHomogeneousResult(4, result, depth, portals, t)
}

func TestHomogeneousPretty(t *testing.T) {
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
	result, depth := DeepestHomogeneous(portals, HomogeneousSpreadAround{}, HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousNumWorkers(6))
	checkValidHomogeneousResult(5, result, depth, t)
}

func appendMidPortals(depth int, p0, p1, p2 Portal, portals []Portal) []Portal {
	if depth <= 0 {
		return portals
	}
	midPoint := s2.PointFromLatLng(p0.LatLng).Add(
		s2.PointFromLatLng(p1.LatLng).Vector).Add(
		s2.PointFromLatLng(p2.LatLng).Vector).Mul(1. / 3.)
	midLL := s2.LatLngFromPoint(s2.Point{Vector: midPoint})
	midPortal := Portal{Guid: p0.Guid + string(rune('0'+depth)), LatLng: midLL}
	portals = append(portals, midPortal)
	portals = appendMidPortals(depth-1, p0, p1, midPortal, portals)
	portals = appendMidPortals(depth-1, p1, p2, midPortal, portals)
	portals = appendMidPortals(depth-1, p2, p0, midPortal, portals)
	return portals
}

func generateHomogeneousPortals(depth int) []Portal {
	ll0 := s2.LatLngFromDegrees(20, 20)
	ll1 := s2.LatLngFromDegrees(20, 22)
	ll2 := s2.LatLngFromDegrees(21, 21)
	portals := []Portal{
		{Guid: "a", LatLng: ll0},
		{Guid: "b", LatLng: ll1},
		{Guid: "c", LatLng: ll2}}
	return appendMidPortals(depth-1, portals[0], portals[1], portals[2], portals)
}

func TestHomogeneousSyntheticPortals(t *testing.T) {
	portals := generateHomogeneousPortals(5)
	result, depth := DeepestHomogeneous(portals, HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousNumWorkers(6))
	checkValidHomogeneousResult(5, result, depth, t)
}

func TestHomogeneousPrettySyntheticPortals(t *testing.T) {
	portals := generateHomogeneousPortals(5)
	result, depth := DeepestHomogeneous(portals, HomogeneousSpreadAround{}, HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousNumWorkers(6))
	checkValidHomogeneousResult(5, result, depth, t)
}

func TestHomogeneousPureSyntheticPortals(t *testing.T) {
	portals := generateHomogeneousPortals(5)
	result, depth := DeepestHomogeneous(portals, HomogeneousPure(true), HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousNumWorkers(6))
	checkValidPureHomogeneousResult(5, result, depth, portals, t)
}

func benchmarkHomogeneous(depth int, b *testing.B) {
	portals := generateHomogeneousPortals(depth)
	for n := 0; n < b.N; n++ {
		_, resDepth := DeepestHomogeneous(portals, HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousNumWorkers(7))
		if depth != int(resDepth) {
			panic(resDepth)
		}
	}
}
func BenchmarkHomogeneous4(b *testing.B) { benchmarkHomogeneous(4, b) }
func BenchmarkHomogeneous5(b *testing.B) { benchmarkHomogeneous(5, b) }
func BenchmarkHomogeneous6(b *testing.B) { benchmarkHomogeneous(6, b) }

func benchmarkHomogeneousPretty(depth int, b *testing.B) {
	portals := generateHomogeneousPortals(depth)
	for n := 0; n < b.N; n++ {
		_, resDepth := DeepestHomogeneous(portals, HomogeneousSpreadAround{}, HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousNumWorkers(6))
		if depth != int(resDepth) {
			panic(resDepth)
		}
	}
}
func BenchmarkHomogeneousPretty4(b *testing.B) { benchmarkHomogeneousPretty(4, b) }
func BenchmarkHomogeneousPretty5(b *testing.B) { benchmarkHomogeneousPretty(5, b) }
func BenchmarkHomogeneousPretty6(b *testing.B) { benchmarkHomogeneousPretty(6, b) }

func benchmarkHomogeneousPure(depth int, b *testing.B) {
	portals := generateHomogeneousPortals(depth)
	for n := 0; n < b.N; n++ {
		_, resDepth := DeepestHomogeneous(portals, HomogeneousPure(true), HomogeneousMaxDepth(6), HomogeneousLargestArea{}, HomogeneousNumWorkers(6))
		if depth != int(resDepth) {
			panic(resDepth)
		}
	}
}
func BenchmarkHomogeneousPure4(b *testing.B) { benchmarkHomogeneousPure(4, b) }
func BenchmarkHomogeneousPure5(b *testing.B) { benchmarkHomogeneousPure(5, b) }
