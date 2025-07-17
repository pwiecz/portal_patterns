package lib

import "math/rand"

type homogeneousTopLevelScorer interface {
	scoreTriangle(a, b, c portalData) float32
}
type homogeneousPureScorer interface {
	scoreTrianglePure(a, b, c portalData, level int, portals []portalData) float32
}

type randomScorer struct {
	rand *rand.Rand
}
type largestTriangleScorer struct{}
type smallestAreaTriangleScorer struct{}
type smallestSidesTriangleScorer struct{}
type mostEquilateralTriangleScorer struct{}

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
func (s smallestAreaTriangleScorer) scoreTriangle(a, b, c portalData) float32 {
	// We multiply by unitAreaToSquare not to obtain any meaningful distance measure
	// (we use value only for comparisons), but just to scale the number up
	// to make it fit in float32 precision range.
	return -float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}
func (s smallestAreaTriangleScorer) scoreTrianglePure(a, b, c portalData, _ int, _ []portalData) float32 {
	return s.scoreTriangle(a, b, c)
}

func (s smallestSidesTriangleScorer) scoreTriangle(a, b, c portalData) float32 {
	return -float32(max(distance(a, b), distance(b, c), distance(a, c)))
}
func (s smallestSidesTriangleScorer) scoreTrianglePure(a, b, c portalData, _ int, _ []portalData) float32 {
	return s.scoreTriangle(a, b, c)
}

func (s mostEquilateralTriangleScorer) scoreTriangle(a, b, c portalData) float32 {
	distAB := distance(a, b)
	distBC := distance(b, c)
	distAC := distance(a, c)
	minDist := min(distAB, distBC, distAC)
	maxDist := max(distAB, distBC, distAC)
	return float32(minDist / maxDist)
}
func (s mostEquilateralTriangleScorer) scoreTrianglePure(a, b, c portalData, _ int, _ []portalData) float32 {
	return s.scoreTriangle(a, b, c)
}
