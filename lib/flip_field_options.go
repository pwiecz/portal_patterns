package lib

type FlipFieldOption interface {
	apply(params *flipFieldParams)
}
type FlipFieldBackbonePortalLimit struct {
	Value     int
	LimitType PortalLimit
}

func (f FlipFieldBackbonePortalLimit) apply(params *flipFieldParams) {
	params.maxBackbonePortals = f.Value
	params.backbonePortalLimit = f.LimitType
}

type FlipFieldMaxFlipPortals int

func (f FlipFieldMaxFlipPortals) apply(params *flipFieldParams) {
	params.maxFlipPortals = int(f)
}

type FlipFieldProgressFunc func(int, int)

func (f FlipFieldProgressFunc) apply(params *flipFieldParams) {
	params.progressFunc = (func(int, int))(f)
}

type FlipFieldNumWorkers int

func (f FlipFieldNumWorkers) apply(params *flipFieldParams) {
	params.numWorkers = int(f)
}

type FlipFieldSimpleBackbone bool

func (f FlipFieldSimpleBackbone) apply(params *flipFieldParams) {
	params.simpleBackbone = bool(f)
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
