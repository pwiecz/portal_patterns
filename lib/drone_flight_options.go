package lib

import "runtime"

type DroneFlightOption interface {
	apply(params *droneFlightParams)
}
type DroneFlightUseLongJumps bool

func (d DroneFlightUseLongJumps) apply(params *droneFlightParams) {
	params.useLongJumps = bool(d)
}

type DroneFlightStartPortalIndex int

func (d DroneFlightStartPortalIndex) apply(params *droneFlightParams) {
	if d < 0 {
		params.startPortalIndex = invalidPortalIndex
	} else {
		params.startPortalIndex = portalIndex(d)
	}
}

type DroneFlightEndPortalIndex int

func (d DroneFlightEndPortalIndex) apply(params *droneFlightParams) {
	if d < 0 {
		params.endPortalIndex = invalidPortalIndex
	} else {
		params.endPortalIndex = portalIndex(d)
	}
}

type DroneFlightLeastJumps struct{}

func (d DroneFlightLeastJumps) apply(params *droneFlightParams) {
	params.optimizeNumKeys = false
}

type DroneFlightLeastKeys struct{}

func (d DroneFlightLeastKeys) apply(params *droneFlightParams) {
	params.optimizeNumKeys = true
}

type DroneFlightNumWorkers int

func (d DroneFlightNumWorkers) apply(params *droneFlightParams) {
	params.numWorkers = int(d)
}

type DroneFlightProgressFunc func(int, int)

func (d DroneFlightProgressFunc) apply(params *droneFlightParams) {
	params.progressFunc = (func(int, int))(d)
}

type droneFlightParams struct {
	progressFunc     func(int, int)
	numWorkers       int
	startPortalIndex portalIndex
	endPortalIndex   portalIndex
	useLongJumps     bool
	optimizeNumKeys  bool
}

func defaultDroneFlightParams() droneFlightParams {
	return droneFlightParams{
		startPortalIndex: invalidPortalIndex,
		endPortalIndex:   invalidPortalIndex,
		useLongJumps:     false,
		optimizeNumKeys:  true,
		numWorkers:       runtime.GOMAXPROCS(0),
		progressFunc:     func(int, int) {},
	}
}
