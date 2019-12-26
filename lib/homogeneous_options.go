package lib

type HomogeneousOption interface {
	apply(param *homogeneousParams)
	apply2(param *homogeneous2Params)
}
type HomogeneousMaxDepth int

func (h HomogeneousMaxDepth) apply(param *homogeneousParams) {
	param.maxDepth = int(h)
}
func (h HomogeneousMaxDepth) apply2(param *homogeneous2Params) {
	param.maxDepth = int(h)
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

type HomogeneousProgressFunc func(int, int)

func (h HomogeneousProgressFunc) apply(param *homogeneousParams) {
	param.progressFunc = (func(int, int))(h)
}
func (h HomogeneousProgressFunc) apply2(param *homogeneous2Params) {
	param.progressFunc = (func(int, int))(h)
}

type HomogeneousFixedCornerIndices []int

func (h HomogeneousFixedCornerIndices) apply(param *homogeneousParams) {
	param.fixedCornerIndices = []int(h)
}
func (h HomogeneousFixedCornerIndices) apply2(param *homogeneous2Params) {
	param.fixedCornerIndices = []int(h)
}

type HomogeneousPerfect bool

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
	maxDepth           int
	perfect            bool
	scorer             homogeneousScorer
	topLevelScorer     homogeneousTopLevelScorer
	fixedCornerIndices []int
	progressFunc       func(int, int)
}

func defaultHomogeneous2Params(numPortals int) homogeneous2Params {
	return homogeneous2Params{
		maxDepth:       6,
		perfect:        false,
		scorer:         newThickTrianglesScorer(numPortals),
		topLevelScorer: arbitraryScorer{},
		progressFunc:   func(int, int) {},
	}
}
