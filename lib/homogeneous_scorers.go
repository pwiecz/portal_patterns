package lib

import "math/rand"

type homogeneousTopLevelScorer interface {
	scoreTriangle(a, b, c portalData) float32
}
type homogeneousPureScorer interface {
	scoreTrianglePure(a, b, c portalData, level int, portals []portalData) float32
}

type arbitraryScorer struct{}
type randomScorer struct {
	rand *rand.Rand
}
type largestTriangleScorer struct{}
type smallestTriangleScorer struct{}
type mostEquilateralTriangleScorer struct{}

func (s arbitraryScorer) scoreTriangle(a, b, c portalData) float32 {
	return 0
}
func (s arbitraryScorer) scoreTrianglePure(a, b, c portalData, _ int, _ []portalData) float32 {
	return s.scoreTriangle(a, b, c)
}
func (s randomScorer) scoreTriangle(a, b, c portalData) float32 {
	return s.rand.Float32()
}
func (s randomScorer) scoreTrianglePure(a, b, c portalData, _ int, _ []portalData) float32 {
	return s.scoreTriangle(a, b, c)
}

func (s largestTriangleScorer) scoreTriangle(a, b, c portalData) float32 {
	// We multiply by unitAreaToSquare not to obtain any meaningful distance measure
	// (we use value only for comparisons), but just to scale the number up
	// to make it fit in float32 precision range.
	return float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}
func (s largestTriangleScorer) scoreTrianglePure(a, b, c portalData, _ int, _ []portalData) float32 {
	return s.scoreTriangle(a, b, c)
}
func (s smallestTriangleScorer) scoreTriangle(a, b, c portalData) float32 {
	// We multiply by unitAreaToSquare not to obtain any meaningful distance measure
	// (we use value only for comparisons), but just to scale the number up
	// to make it fit in float32 precision range.
	return -float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}
func (s smallestTriangleScorer) scoreTrianglePure(a, b, c portalData, _ int, _ []portalData) float32 {
	return s.scoreTriangle(a, b, c)
}

func (s mostEquilateralTriangleScorer) scoreTriangle(a, b, c portalData) float32 {
	distAB := distance(a, b)
	distBC := distance(b, c)
	distAC := distance(a, c)
	minDist := float64Min(distAB, float64Min(distBC, distAC))
	maxDist := float64Max(distAB, float64Max(distBC, distAC))
	return float32(minDist / maxDist)
}
func (s mostEquilateralTriangleScorer) scoreTrianglePure(a, b, c portalData, _ int, _ []portalData) float32 {
	return s.scoreTriangle(a, b, c)
}
func (s mostEquilateralTriangleScorer) scoreTriangle2(a, b, c portalData, scorer homogeneousScorer) float32 {
	distAB := distance(a, b)
	distBC := distance(b, c)
	distAC := distance(a, c)
	minDist := float64Min(distAB, float64Min(distBC, distAC))
	maxDist := float64Max(distAB, float64Max(distBC, distAC))
	return float32(minDist / maxDist)
}
