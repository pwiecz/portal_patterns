package lib

import (
	"math/rand"
	"runtime"
)

type HomogeneousOption interface {
	requires2() bool
	apply(params *homogeneousParams)
	apply2(params *homogeneous2Params)
	applyPure(params *homogeneousPureParams)
}

type HomogeneousMaxDepth int

func (h HomogeneousMaxDepth) requires2() bool { return false }

func (h HomogeneousMaxDepth) apply(params *homogeneousParams) {
	params.maxDepth = int(h)
}
func (h HomogeneousMaxDepth) apply2(params *homogeneous2Params) {
	params.maxDepth = int(h)
}
func (h HomogeneousMaxDepth) applyPure(params *homogeneousPureParams) {
	params.maxDepth = int(h)
}

type HomogeneousSpreadAround struct{}

func (h HomogeneousSpreadAround) requires2() bool                 { return true }
func (h HomogeneousSpreadAround) apply(params *homogeneousParams) { panic("unsupported") }
func (h HomogeneousSpreadAround) apply2(params *homogeneous2Params) {
	params.scorer = newThickTrianglesScorer(params.numPortals)
	params.topLevelScorer = params.scorer
}
func (h HomogeneousSpreadAround) applyPure(params *homogeneousPureParams) {
	params.scorer = thickTrianglesPureScorer{}
}

type HomogeneousClumpTogether struct{}

func (h HomogeneousClumpTogether) requires2() bool                 { return true }
func (h HomogeneousClumpTogether) apply(params *homogeneousParams) { panic("unsupported") }
func (h HomogeneousClumpTogether) apply2(params *homogeneous2Params) {
	params.scorer = newClumpPortalsScorer(params.numPortals)
	params.topLevelScorer = params.scorer
}
func (h HomogeneousClumpTogether) applyPure(params *homogeneousPureParams) {
	panic("not implemented")
}

type HomogeneousRandom struct {
	Rand *rand.Rand
}

func (h HomogeneousRandom) requires2() bool { return false }

func (h HomogeneousRandom) apply(params *homogeneousParams) {
	params.topLevelScorer = randomScorer{h.Rand}
}
func (h HomogeneousRandom) apply2(params *homogeneous2Params) {
	params.topLevelScorer = randomScorer{h.Rand}
}
func (h HomogeneousRandom) applyPure(params *homogeneousPureParams) {
	params.scorer = randomScorer{h.Rand}
}

type HomogeneousLargestArea struct{}

func (h HomogeneousLargestArea) requires2() bool { return false }
func (h HomogeneousLargestArea) apply(params *homogeneousParams) {
	params.topLevelScorer = largestTriangleScorer{}
}
func (h HomogeneousLargestArea) apply2(params *homogeneous2Params) {
	params.topLevelScorer = largestTriangleScorer{}
}
func (h HomogeneousLargestArea) applyPure(params *homogeneousPureParams) {
	params.scorer = largestTriangleScorer{}
}

type HomogeneousSmallestArea struct{}

func (h HomogeneousSmallestArea) requires2() bool { return false }

func (h HomogeneousSmallestArea) apply(params *homogeneousParams) {
	params.topLevelScorer = smallestTriangleScorer{}
}
func (h HomogeneousSmallestArea) apply2(params *homogeneous2Params) {
	params.topLevelScorer = smallestTriangleScorer{}
}
func (h HomogeneousSmallestArea) applyPure(params *homogeneousPureParams) {
	params.scorer = smallestTriangleScorer{}
}

type HomogeneousMostEquilateralTriangle struct{}

func (h HomogeneousMostEquilateralTriangle) requires2() bool { return false }

func (h HomogeneousMostEquilateralTriangle) apply(params *homogeneousParams) {
	params.topLevelScorer = mostEquilateralTriangleScorer{}
}
func (h HomogeneousMostEquilateralTriangle) apply2(params *homogeneous2Params) {
	params.topLevelScorer = mostEquilateralTriangleScorer{}
}
func (h HomogeneousMostEquilateralTriangle) applyPure(params *homogeneousPureParams) {
	params.scorer = mostEquilateralTriangleScorer{}
}

type HomogeneousNumWorkers int

func (h HomogeneousNumWorkers) requires2() bool { return false }

func (h HomogeneousNumWorkers) apply(params *homogeneousParams)   {}
func (h HomogeneousNumWorkers) apply2(params *homogeneous2Params) {}
func (h HomogeneousNumWorkers) applyPure(params *homogeneousPureParams) {
	params.numWorkers = (int)(h)
}

type HomogeneousProgressFunc func(int, int)

func (h HomogeneousProgressFunc) requires2() bool { return false }

func (h HomogeneousProgressFunc) apply(params *homogeneousParams) {
	params.progressFunc = (func(int, int))(h)
}
func (h HomogeneousProgressFunc) apply2(params *homogeneous2Params) {
	params.progressFunc = (func(int, int))(h)
}
func (h HomogeneousProgressFunc) applyPure(params *homogeneousPureParams) {
	params.progressFunc = (func(int, int))(h)
}

type HomogeneousFixedCornerIndices []int

func (h HomogeneousFixedCornerIndices) requires2() bool { return false }

func (h HomogeneousFixedCornerIndices) apply(params *homogeneousParams) {
	params.fixedCornerIndices = []int(h)
}
func (h HomogeneousFixedCornerIndices) apply2(params *homogeneous2Params) {
	params.fixedCornerIndices = []int(h)
}
func (h HomogeneousFixedCornerIndices) applyPure(params *homogeneousPureParams) {
	params.fixedCornerIndices = []int(h)
}

type HomogeneousPure bool

func (h HomogeneousPure) requires2() bool { return false }
func (h HomogeneousPure) apply(params *homogeneousParams) {
	if h {
		panic("unsupported")
	}
}
func (h HomogeneousPure) apply2(params *homogeneous2Params) {
	if h {
		panic("unsupported")
	}
}
func (h HomogeneousPure) applyPure(params *homogeneousPureParams) {}

type homogeneousParams struct {
	maxDepth           int
	topLevelScorer     homogeneousTopLevelScorer
	fixedCornerIndices []int
	progressFunc       func(int, int)
}

func defaultHomogeneousParams() homogeneousParams {
	return homogeneousParams{
		maxDepth:       6,
		topLevelScorer: smallestTriangleScorer{},
		progressFunc:   func(int, int) {},
	}
}

type homogeneous2Params struct {
	homogeneousParams
	numPortals int
	scorer     homogeneousScorer
}

func defaultHomogeneous2Params(numPortals int) homogeneous2Params {
	defaultScorer := newThickTrianglesScorer(numPortals)
	return homogeneous2Params{
		homogeneousParams: homogeneousParams{
			maxDepth: 6,
			// by default pick top level triangle with the highest score
			topLevelScorer: defaultScorer,
			progressFunc:   func(int, int) {},
		},
		numPortals: numPortals,
		scorer:     defaultScorer,
	}
}

type homogeneousPureParams struct {
	maxDepth           int
	scorer             homogeneousPureScorer
	fixedCornerIndices []int
	numWorkers         int
	progressFunc       func(int, int)
}

func defaultHomogeneousPureParams() homogeneousPureParams {
	return homogeneousPureParams{
		maxDepth:     6,
		scorer:       smallestTriangleScorer{},
		numWorkers:   runtime.GOMAXPROCS(0),
		progressFunc: func(int, int) {},
	}
}
