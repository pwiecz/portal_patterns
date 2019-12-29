package lib

import "sync"

type bestFlipFieldMtQuery struct {
	maxBackbonePortals int
	numPortalLimit     PortalLimit
	maxFlipPortals     int
	simpleBackbone     bool
	portals            []portalData
}

func (f *bestFlipFieldMtQuery) findBestFlipField(p0, p1 portalData, ccw bool, backbone, candidates []portalData, bestSolution int) ([]portalData, []portalData, float64) {
	fq := bestFlipFieldQuery{
		maxBackbonePortals: f.maxBackbonePortals,
		numPortalLimit:     f.numPortalLimit,
		maxFlipPortals:     f.maxFlipPortals,
		simpleBackbone:     f.simpleBackbone,
		bestSolution:       bestSolution,
		portals:            f.portals,
		backbone:           backbone,
		candidates:         candidates,
	}
	return fq.findBestFlipField(p0, p1, ccw)
}

type flipFieldRequest struct {
	p0, p1                portalData
	ccw                   bool
	backbone, flipPortals []portalData
	backboneLength        float64
}

func bestFlipFieldWorker(
	q *bestFlipFieldMtQuery,
	requestChannel, responseChannel chan flipFieldRequest,
	wg *sync.WaitGroup) {
	var localBestNumFields int
	for req := range requestChannel {
		b, f, bl := q.findBestFlipField(req.p0, req.p1, req.ccw, req.backbone, req.flipPortals, localBestNumFields)
		if q.numPortalLimit != EQUAL || len(b) == q.maxBackbonePortals {
			numFlipPortals := len(f)
			if q.maxFlipPortals > 0 && numFlipPortals > q.maxFlipPortals {
				numFlipPortals = q.maxFlipPortals
			}
			numFields := numFlipPortals * (2*len(b) - 1)
			if numFields > localBestNumFields {
				localBestNumFields = numFields
			}
		}
		req.backbone = b
		req.flipPortals = f
		req.backboneLength = bl
		responseChannel <- req
	}
	wg.Done()
}

func LargestFlipFieldMT(portals []Portal, params flipFieldParams) ([]Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

	backboneCache := sync.Pool{
		New: func() interface{} {
			return make([]portalData, 0, params.maxBackbonePortals)
		},
	}
	flipPortalsCache := sync.Pool{
		New: func() interface{} {
			return make([]portalData, 0, len(portals))
		},
	}

	requestChannel := make(chan flipFieldRequest, params.numWorkers)
	responseChannel := make(chan flipFieldRequest, params.numWorkers)
	var wg sync.WaitGroup
	wg.Add(params.numWorkers)
	q := &bestFlipFieldMtQuery{params.maxBackbonePortals, params.backbonePortalLimit, params.maxFlipPortals, params.simpleBackbone, portalsData}
	for i := 0; i < params.numWorkers; i++ {
		go bestFlipFieldWorker(q, requestChannel, responseChannel, &wg)
	}
	go func() {
		for _, p0 := range portalsData {
			for _, p1 := range portalsData {
				if p0.Index == p1.Index {
					continue
				}
				for _, ccw := range []bool{true, false} {
					requestChannel <- flipFieldRequest{
						p0:          p0,
						p1:          p1,
						ccw:         ccw,
						backbone:    backboneCache.Get().([]portalData),
						flipPortals: flipPortalsCache.Get().([]portalData),
					}
				}
			}
		}
		close(requestChannel)
	}()
	go func() {
		wg.Wait()
		close(responseChannel)
	}()
	numPairs := len(portals) * (len(portals) - 1) * 2
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	numProcessedPairs := 0
	numProcessedPairsModN := 0
	params.progressFunc(0, numPairs)

	var bestNumFields int
	bestBackbone, bestFlipPortals := []portalData(nil), []portalData(nil)
	var bestBackboneLength float64
	for resp := range responseChannel {
		if params.backbonePortalLimit != EQUAL || len(resp.backbone) == params.maxBackbonePortals {
			numFlipPortals := len(resp.flipPortals)
			if params.maxFlipPortals > 0 && numFlipPortals > params.maxFlipPortals {
				numFlipPortals = params.maxFlipPortals
			}
			numFields := numFlipPortals * (2*len(resp.backbone) - 1)
			if numFields > bestNumFields || (numFields == bestNumFields && resp.backboneLength < bestBackboneLength) {
				bestNumFields = numFields
				bestBackbone = append(bestBackbone[:0], resp.backbone...)
				bestFlipPortals = append(bestFlipPortals[:0], resp.flipPortals...)
				bestBackboneLength = resp.backboneLength
			}
		}
		backboneCache.Put(resp.backbone)
		flipPortalsCache.Put(resp.flipPortals)
		numProcessedPairs++
		numProcessedPairsModN++
		if numProcessedPairsModN == everyNth {
			numProcessedPairsModN = 0
			params.progressFunc(numProcessedPairs, numPairs)
		}
	}
	params.progressFunc(numPairs, numPairs)

	resultBackbone := make([]Portal, 0, len(bestBackbone))
	for _, p := range bestBackbone {
		resultBackbone = append(resultBackbone, portals[p.Index])
	}
	resultFlipPortals := make([]Portal, 0, len(bestFlipPortals))
	for _, p := range bestFlipPortals {
		resultFlipPortals = append(resultFlipPortals, portals[p.Index])
	}
	return resultBackbone, resultFlipPortals
}
