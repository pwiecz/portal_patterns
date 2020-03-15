package lib

type arbitraryScorer struct{}
type largestTriangleScorer struct{}
type smallestTriangleScorer struct{}

func (s arbitraryScorer) scoreTriangle(a, b, c portalData) float32 {
	return 0
}
func (s arbitraryScorer) scoreTriangle2(a, b, c portalData, scorer homogeneousScorer) float32 {
	return scorer.scoreTriangle(a, b, c)
}

func (s largestTriangleScorer) scoreTriangle(a, b, c portalData) float32 {
	// We multiply by unitAreaToSquare not to obtain any meaningful distance measure
	// (we use value only for comparisons), but just to scale the number up
	// to make it fit in float32 precision range.
	return float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}
func (s largestTriangleScorer) scoreTriangle2(a, b, c portalData, scorer homogeneousScorer) float32 {
	// We multiply by unitAreaToSquare not to obtain any meaningful distance measure
	// (we use value only for comparisons), but just to scale the number up
	// to make it fit in float32 precision range.
	return float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}

func (s smallestTriangleScorer) scoreTriangle(a, b, c portalData) float32 {
	// We multiply by unitAreaToSquare not to obtain any meaningful distance measure
	// (we use value only for comparisons), but just to scale the number up
	// to make it fit in float32 precision range.
	return -float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}
func (s smallestTriangleScorer) scoreTriangle2(a, b, c portalData, scorer homogeneousScorer) float32 {
	// We multiply by unitAreaToSquare not to obtain any meaningful distance measure
	// (we use value only for comparisons), but just to scale the number up
	// to make it fit in float32 precision range.
	return -float32(triangleArea(a, b, c) * unitAreaToSquareMeters)
}
