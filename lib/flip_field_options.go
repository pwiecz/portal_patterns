package lib

type FlipFieldOption interface {
	apply(param *flipFieldParams)
}
type FlipFieldBackbonePortalLimit struct {
	Value     int
	LimitType PortalLimit
}

func (f FlipFieldBackbonePortalLimit) apply(param *flipFieldParams) {
	param.maxBackbonePortals = f.Value
	param.backbonePortalLimit = f.LimitType
}

type FlipFieldMaxFlipPortals struct {
	Value int
}

func (f FlipFieldMaxFlipPortals) apply(param *flipFieldParams) {
	param.maxFlipPortals = f.Value
}

type FlipFieldProgressFunc struct {
	ProgressFunc func(int, int)
}

func (f FlipFieldProgressFunc) apply(param *flipFieldParams) {
	param.progressFunc = f.ProgressFunc
}

type FlipFieldNumWorkers struct {
	NumWorkers int
}

func (f FlipFieldNumWorkers) apply(param *flipFieldParams) {
	param.numWorkers = f.NumWorkers
}

type FlipFieldSimpleBackbone bool

func (f FlipFieldSimpleBackbone) apply(param *flipFieldParams) {
	param.simpleBackbone = bool(f)
}

type flipFieldParams struct {
	maxBackbonePortals  int
	backbonePortalLimit PortalLimit
	maxFlipPortals      int
	simpleBackbone      bool
	numWorkers          int
	progressFunc        func(int, int)
}

func defaultFlipFieldParams() flipFieldParams {
	return flipFieldParams{
		maxBackbonePortals:  16,
		backbonePortalLimit: EQUAL,
		maxFlipPortals:      0,
		simpleBackbone:      false,
		numWorkers:          0,
		progressFunc:        func(int, int) {},
	}
}
