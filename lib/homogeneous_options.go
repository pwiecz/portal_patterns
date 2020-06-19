package lib

import "math/rand"

type HomogeneousOption interface {
	requires2() bool
	apply(param *homogeneousParams)
	apply2(param *homogeneous2Params)
}

type HomogeneousMaxDepth int

func (h HomogeneousMaxDepth) requires2() bool { return false }

func (h HomogeneousMaxDepth) apply(param *homogeneousParams) {
	param.maxDepth = int(h)
}
func (h HomogeneousMaxDepth) apply2(param *homogeneous2Params) {
	param.maxDepth = int(h)
}

type HomogeneousSpreadAround int

func (h HomogeneousSpreadAround) requires2() bool                { return true }
func (h HomogeneousSpreadAround) apply(param *homogeneousParams) { panic("unsupported") }
func (h HomogeneousSpreadAround) apply2(param *homogeneous2Params) {
	param.scorer = newThickTrianglesScorer(int(h))
	param.topLevelScorer = param.scorer
}

type HomogeneousClumpTogether int

func (h HomogeneousClumpTogether) requires2() bool                { return true }
func (h HomogeneousClumpTogether) apply(param *homogeneousParams) { panic("unsupported") }
func (h HomogeneousClumpTogether) apply2(param *homogeneous2Params) {
	param.scorer = newClumpPortalsScorer(int(h))
	param.topLevelScorer = param.scorer
}

type HomogeneousRandom struct {
	Rand *rand.Rand
}

func (h HomogeneousRandom) requires2() bool { return false }

func (h HomogeneousRandom) apply(param *homogeneousParams) {
	param.topLevelScorer = randomScorer{h.Rand}
}
func (h HomogeneousRandom) apply2(param *homogeneous2Params) {
	param.topLevelScorer = randomScorer{h.Rand}
}

type HomogeneousLargestArea struct{}

func (h HomogeneousLargestArea) requires2() bool { return false }
func (h HomogeneousLargestArea) apply(param *homogeneousParams) {
	param.topLevelScorer = largestTriangleScorer{}
}
func (h HomogeneousLargestArea) apply2(param *homogeneous2Params) {
	param.topLevelScorer = largestTriangleScorer{}
}

type HomogeneousSmallestArea struct{}

func (h HomogeneousSmallestArea) requires2() bool { return false }

func (h HomogeneousSmallestArea) apply(param *homogeneousParams) {
	param.topLevelScorer = smallestTriangleScorer{}
}
func (h HomogeneousSmallestArea) apply2(param *homogeneous2Params) {
	param.topLevelScorer = smallestTriangleScorer{}
}

type HomogeneousMostEquilateralTriangle struct{}

func (h HomogeneousMostEquilateralTriangle) requires2() bool { return false }

func (h HomogeneousMostEquilateralTriangle) apply(param *homogeneousParams) {
	param.topLevelScorer = mostEquilateralTriangleScorer{}
}
func (h HomogeneousMostEquilateralTriangle) apply2(param *homogeneous2Params) {
	param.topLevelScorer = mostEquilateralTriangleScorer{}
}

type HomogeneousProgressFunc func(int, int)

func (h HomogeneousProgressFunc) requires2() bool { return false }

func (h HomogeneousProgressFunc) apply(param *homogeneousParams) {
	param.progressFunc = (func(int, int))(h)
}
func (h HomogeneousProgressFunc) apply2(param *homogeneous2Params) {
	param.progressFunc = (func(int, int))(h)
}

type HomogeneousFixedCornerIndices []int

func (h HomogeneousFixedCornerIndices) requires2() bool { return false }

func (h HomogeneousFixedCornerIndices) apply(param *homogeneousParams) {
	param.fixedCornerIndices = []int(h)
}
func (h HomogeneousFixedCornerIndices) apply2(param *homogeneous2Params) {
	param.fixedCornerIndices = []int(h)
}

type HomogeneousPerfect bool

func (h HomogeneousPerfect) requires2() bool { return false }

func (h HomogeneousPerfect) apply(param *homogeneousParams) {
	param.perfect = bool(h)
}
func (h HomogeneousPerfect) apply2(param *homogeneous2Params) {
	param.perfect = bool(h)
}

type homogeneousParams struct {
	maxDepth           int
	perfect            bool
	topLevelScorer     homogeneousTopLevelScorer
	fixedCornerIndices []int
	progressFunc       func(int, int)
}

func defaultHomogeneousParams() homogeneousParams {
	return homogeneousParams{
		maxDepth:       6,
		perfect:        false,
		topLevelScorer: arbitraryScorer{},
		progressFunc:   func(int, int) {},
	}
}

type homogeneous2Params struct {
	homogeneousParams
	scorer homogeneousScorer
}

func defaultHomogeneous2Params(numPortals int) homogeneous2Params {
	defaultScorer := newThickTrianglesScorer(numPortals)
	return homogeneous2Params{
		homogeneousParams: homogeneousParams{
			maxDepth: 6,
			perfect:  false,
			// by default pick top level triangle with the highest score
			topLevelScorer: defaultScorer,
			progressFunc:   func(int, int) {},
		},
		scorer: defaultScorer,
	}
}
