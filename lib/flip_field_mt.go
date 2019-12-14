package lib

import "sync"

type bestFlipFieldMtQuery struct {
	maxBackbonePortals int
	numPortalLimit     PortalLimit
	portals            []portalData
}

func (f *bestFlipFieldMtQuery) findBestFlipField(p0, p1 portalData, backbone, candidates []portalData, bestSolution int) ([]portalData, []portalData) {
	candidates = candidates[:0]
	for _, portal := range f.portals {
		if portal.Index == p0.Index || portal.Index == p1.Index {
			continue
		}
		if Sign(p0.LatLng, p1.LatLng, portal.LatLng) <= 0 {
			continue
		}
		candidates = append(candidates, portal)
	}
	flipPortals := candidates
	backbone = append(backbone[:0], p0, p1)
	for {
		if len(backbone) >= f.maxBackbonePortals {
			break
		}
		if len(flipPortals)*(2*f.maxBackbonePortals-1) < bestSolution {
			break
		}
		bestNumFields := len(flipPortals) * (2*len(backbone) - 1)
		if f.numPortalLimit == EQUAL {
			bestNumFields = 0
		}
		bestCandidate := -1
		bestInsertPosition := -1
		for i, candidate := range candidates {
			for pos := 1; pos < len(backbone); pos++ {
				numFlipPortals := numPortalsLeftOfTwoLines(flipPortals, backbone[pos-1], candidate, backbone[pos])
				numFields := numFlipPortals * (2*len(backbone) + 1)
				if numFields > bestNumFields {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = pos
				}
			}
		}
		if bestCandidate < 0 {
			break
		}
		backbone = append(backbone, portalData{})
		copy(backbone[bestInsertPosition+1:], backbone[bestInsertPosition:])
		backbone[bestInsertPosition] = candidates[bestCandidate]
		candidates[bestCandidate], candidates[len(candidates)-1] =
			candidates[len(candidates)-1], candidates[bestCandidate]
		candidates = candidates[:len(candidates)-1]
		flipPortals = partitionPortalsLeftOfLine(flipPortals, backbone[bestInsertPosition-1], backbone[bestInsertPosition])
		flipPortals = partitionPortalsLeftOfLine(flipPortals, backbone[bestInsertPosition], backbone[bestInsertPosition+1])
	}
	return backbone, flipPortals
}

type flipFieldRequest struct {
	p0, p1                portalData
	backbone, flipPortals []portalData
}

func bestFlipFieldWorker(
	q *bestFlipFieldMtQuery,
	requestChannel, responseChannel chan flipFieldRequest,
	wg *sync.WaitGroup) {
	var localBestNumFields int
	for req := range requestChannel {
		b, f := q.findBestFlipField(req.p0, req.p1, req.backbone, req.flipPortals, localBestNumFields)
		if q.numPortalLimit != EQUAL || len(b) == q.maxBackbonePortals {
			numFields := len(f) * (2*len(b) - 1)
			if numFields > localBestNumFields {
				localBestNumFields = numFields
			}
		}
		req.backbone = b
		req.flipPortals = f
		responseChannel <- req
	}
	wg.Done()
}

func LargestFlipFieldMT(portals []Portal, maxBackbonePortals int, numPortalLimit PortalLimit, numWorkers int, progressFunc func(int, int)) ([]Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

	backboneCache := sync.Pool{
		New: func() interface{} {
			return make([]portalData, 0, maxBackbonePortals)
		},
	}
	flipPortalsCache := sync.Pool{
		New: func() interface{} {
			return make([]portalData, 0, len(portals))
		},
	}

	requestChannel := make(chan flipFieldRequest, numWorkers)
	responseChannel := make(chan flipFieldRequest, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	q := &bestFlipFieldMtQuery{maxBackbonePortals, numPortalLimit, portalsData}
	for i := 0; i < numWorkers; i++ {
		go bestFlipFieldWorker(q, requestChannel, responseChannel, &wg)
	}
	go func() {
		for _, p0 := range portalsData {
			for _, p1 := range portalsData {
				if p0.Index == p1.Index {
					continue
				}
				requestChannel <- flipFieldRequest{
					p0:          p0,
					p1:          p1,
					backbone:    backboneCache.Get().([]portalData),
					flipPortals: flipPortalsCache.Get().([]portalData),
				}
			}
		}
		close(requestChannel)
	}()
	go func() {
		wg.Wait()
		close(responseChannel)
	}()
	numPairs := len(portals) * (len(portals) - 1)
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	numProcessedPairs := 0
	numProcessedPairsModN := 0
	progressFunc(0, numPairs)

	var bestNumFields int
	bestBackbone, bestFlipPortals := []portalData(nil), []portalData(nil)
	var bestDistanceSq float64
	for resp := range responseChannel {
		if numPortalLimit != EQUAL || len(resp.backbone) == maxBackbonePortals {
			numFields := len(resp.flipPortals) * (2*len(resp.backbone) - 1)
			distanceSq := DistanceSq(resp.p0.LatLng, resp.p1.LatLng)
			if numFields > bestNumFields || (numFields == bestNumFields && distanceSq < bestDistanceSq) {
				bestNumFields = numFields
				bestBackbone = append(bestBackbone[:0], resp.backbone...)
				bestFlipPortals = append(bestFlipPortals[:0], resp.flipPortals...)
				bestDistanceSq = distanceSq
			}
		}
		backboneCache.Put(resp.backbone)
		flipPortalsCache.Put(resp.flipPortals)
		numProcessedPairs++
		numProcessedPairsModN++
		if numProcessedPairsModN == everyNth {
			numProcessedPairsModN = 0
			progressFunc(numProcessedPairs, numPairs)
		}
	}
	progressFunc(numPairs, numPairs)

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
