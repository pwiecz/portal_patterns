package lib

import "runtime"

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

type FlipFieldFixedBaseIndices []int

func (f FlipFieldFixedBaseIndices) apply(params *flipFieldParams) {
	params.fixedBaseIndices = []int(f)
}

type flipFieldParams struct {
	progressFunc        func(int, int)
	maxBackbonePortals  int
	backbonePortalLimit PortalLimit
	fixedBaseIndices    []int
	maxFlipPortals      int
	numWorkers          int
	simpleBackbone      bool
}

func defaultFlipFieldParams() flipFieldParams {
	return flipFieldParams{
		maxBackbonePortals:  16,
		backbonePortalLimit: EQUAL,
		fixedBaseIndices:    nil,
		maxFlipPortals:      0,
		simpleBackbone:      false,
		numWorkers:          runtime.GOMAXPROCS(0),
		progressFunc:        func(int, int) {},
	}
}
