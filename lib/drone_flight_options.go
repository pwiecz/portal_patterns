package lib

type DroneFlightOption interface {
	apply(params *droneFlightParams)
}
type DroneFlightUseLongJumps bool

func (d DroneFlightUseLongJumps) apply(params *droneFlightParams) {
	params.useLongJumps = bool(d)
}

type DroneFlightStartPortalIndex int

func (d DroneFlightStartPortalIndex) apply(params *droneFlightParams) {
	params.startPortalIndex = int(d)
}

type DroneFlightEndPortalIndex int

func (d DroneFlightEndPortalIndex) apply(params *droneFlightParams) {
	params.endPortalIndex = int(d)
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
	startPortalIndex, endPortalIndex int
	useLongJumps                     bool
	numWorkers                       int
	progressFunc                     func(int, int)
}

func defaultDroneFlightParams() droneFlightParams {
	return droneFlightParams{
		startPortalIndex: -1,
		endPortalIndex:   -1,
		useLongJumps:     true,
		numWorkers:       0,
		progressFunc:     func(int, int) {},
	}
}
