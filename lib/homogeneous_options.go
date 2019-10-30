package lib

type HomogeneousOption interface {
	apply(param *homogeneousParams)
	apply2(param *homogeneous2Params)
}
type HomogeneousMaxDepth struct {
	MaxDepth int
}

func (h HomogeneousMaxDepth) apply(param *homogeneousParams) {
	param.maxDepth = h.MaxDepth
}
func (h HomogeneousMaxDepth) apply2(param *homogeneous2Params) {
	param.maxDepth = h.MaxDepth
}

type HomogeneousLargestArea struct{}

func (h HomogeneousLargestArea) apply(param *homogeneousParams) {
	param.topLevelScorer = largestTriangleScorer{}
}
func (h HomogeneousLargestArea) apply2(param *homogeneous2Params) {
	param.topLevelScorer = largestTriangleScorer{}
}

type HomogeneousSmallestArea struct{}

func (h HomogeneousSmallestArea) apply(param *homogeneousParams) {
	param.topLevelScorer = smallestTriangleScorer{}
}
func (h HomogeneousSmallestArea) apply2(param *homogeneous2Params) {
	param.topLevelScorer = smallestTriangleScorer{}
}

type HomogeneousProgressFunc struct {
	ProgressFunc func(int, int)
}

func (h HomogeneousProgressFunc) apply(param *homogeneousParams) {
	param.progressFunc = h.ProgressFunc
}
func (h HomogeneousProgressFunc) apply2(param *homogeneous2Params) {
	param.progressFunc = h.ProgressFunc
}

type HomogeneousFixedCornerIndices struct {
	Indices []int
}

func (h HomogeneousFixedCornerIndices) apply(param *homogeneousParams) {
	param.fixedCornerIndices = h.Indices
}
func (h HomogeneousFixedCornerIndices) apply2(param *homogeneous2Params) {
	param.fixedCornerIndices = h.Indices
}

type homogeneousParams struct {
	maxDepth           int
	topLevelScorer     homogeneousTopLevelScorer
	fixedCornerIndices []int
	progressFunc       func(int, int)
}

func defaultHomogeneousParams() homogeneousParams {
	return homogeneousParams{
		maxDepth:       6,
		topLevelScorer: arbitraryScorer{},
		progressFunc:   func(int, int) {},
	}
}

type homogeneous2Params struct {
	maxDepth           int
	scorer             HomogeneousScorer
	topLevelScorer     homogeneousTopLevelScorer
	fixedCornerIndices []int
	progressFunc       func(int, int)
}

func defaultHomogeneous2Params(numPortals int) homogeneous2Params {
	return homogeneous2Params{
		maxDepth:       6,
		scorer:         newThickTrianglesScorer(numPortals),
		topLevelScorer: arbitraryScorer{},
		progressFunc:   func(int, int) {},
	}
}
