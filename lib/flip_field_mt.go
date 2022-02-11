package lib

import (
	"fmt"
	"sync"
)

type bestFlipFieldMtQuery struct {
	portals            []portalData
	fixedBaseIndices   []portalIndex
	maxBackbonePortals int
	maxFlipPortals     int
	numPortalLimit     PortalLimit
	simpleBackbone     bool
}

func (f *bestFlipFieldMtQuery) findBestFlipField(p0, p1 portalData, ccw bool, backbone, flipPortals, candidates []portalData, bestSolution int) ([]portalData, []portalData, float64) {
	fq := bestFlipFieldQuery{
		maxBackbonePortals: f.maxBackbonePortals,
		numPortalLimit:     f.numPortalLimit,
		maxFlipPortals:     f.maxFlipPortals,
		simpleBackbone:     f.simpleBackbone,
		bestSolution:       bestSolution,
		portals:            f.portals,
		fixedBaseIndices:   f.fixedBaseIndices,
		backbone:           backbone,
		candidates:         candidates,
		flipPortals:        flipPortals,
	}
	return fq.findBestFlipField(p0, p1, ccw)
}

type flipFieldRequest struct {
	backbone       []portalData
	flipPortals    []portalData
	p0             portalData
	p1             portalData
	backboneLength float64
	ccw            bool
}

func bestFlipFieldWorker(
	q *bestFlipFieldMtQuery,
	requestChannel, responseChannel chan flipFieldRequest,
	wg *sync.WaitGroup) {
	var localBestNumFields int
	candidates := make([]portalData, 0, len(q.portals))
	for req := range requestChannel {
		b, f, bl := q.findBestFlipField(req.p0, req.p1, req.ccw, req.backbone, req.flipPortals, candidates, localBestNumFields)
		if q.numPortalLimit != EQUAL || len(b) == q.maxBackbonePortals {
			numFlipPortals := len(f)
			if q.maxFlipPortals > 0 && numFlipPortals > q.maxFlipPortals {
				numFlipPortals = q.maxFlipPortals
			}
			numFields := numFlipFields(numFlipPortals, len(b))
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
	if params.numWorkers < 1 {
		panic(fmt.Errorf("too few workers: %d", params.numWorkers))
	}
	if len(portals) < 3 {
		panic(fmt.Errorf("too short portal list: %d", len(portals)))
	}
	portalsData := portalsToPortalData(portals)
	fixedBaseIndices := []portalIndex{}
	for _, i := range params.fixedBaseIndices {
		fixedBaseIndices = append(fixedBaseIndices, portalsData[i].Index)
	}

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
	q := &bestFlipFieldMtQuery{
		maxBackbonePortals: params.maxBackbonePortals,
		numPortalLimit:     params.backbonePortalLimit,
		maxFlipPortals:     params.maxFlipPortals,
		simpleBackbone:     params.simpleBackbone,
		portals:            portalsData,
		fixedBaseIndices:   fixedBaseIndices}
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
		if len(resp.backbone) >= 2 &&
			hasAllPortalIndicesInThePair(fixedBaseIndices, resp.backbone[0].Index, resp.backbone[len(resp.backbone)-1].Index) &&
			(params.backbonePortalLimit != EQUAL || len(resp.backbone) == params.maxBackbonePortals) {
			numFlipPortals := len(resp.flipPortals)
			if params.maxFlipPortals > 0 && numFlipPortals > params.maxFlipPortals {
				numFlipPortals = params.maxFlipPortals
			}
			numFields := numFlipFields(numFlipPortals, len(resp.backbone))
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
