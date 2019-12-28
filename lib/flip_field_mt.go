package lib

import "sync"
import "github.com/golang/geo/s2"

type bestFlipFieldMtQuery struct {
	maxBackbonePortals int
	numPortalLimit     PortalLimit
	maxFlipPortals     int
	simpleBackbone     bool
	portals            []portalData
}

func (f *bestFlipFieldMtQuery) findBestFlipField(p0, p1 portalData, ccw bool, backbone, candidates []portalData, bestSolution int) ([]portalData, []portalData, float64) {
	if ccw {
		candidates = portalsLeftOfLine(f.portals, p0, p1, candidates[:0])
	} else {
		candidates = portalsLeftOfLine(f.portals, p1, p0, candidates[:0])
	}
	flipPortals := candidates
	backbone = append(backbone[:0], p0, p1)
	backboneLength := distance(p0, p1)
	for {
		if len(backbone) >= f.maxBackbonePortals {
			break
		}
		if len(flipPortals)*(2*f.maxBackbonePortals-1) < bestSolution {
			break
		}
		bestNumFields := len(flipPortals) * (2*len(backbone) - 1)
		bestBackboneLength := backboneLength
		if f.numPortalLimit == EQUAL {
			bestNumFields = 0
		}
		bestCandidate := -1
		bestInsertPosition := -1
		for i, candidate := range candidates {
			for pos := 1; pos < len(backbone); pos++ {
				if f.simpleBackbone {
					q := newTriangleWedgeQuery(backbone[0].LatLng, backbone[pos-1].LatLng, backbone[pos].LatLng)
					if !q.ContainsPoint(candidate.LatLng) {
						continue
					}
				}
				var numFlipPortals int
				if ccw {
					numFlipPortals = numPortalsLeftOfTwoLines(flipPortals, backbone[pos-1], candidate, backbone[pos])
				} else {
					numFlipPortals = numPortalsLeftOfTwoLines(flipPortals, backbone[pos], candidate, backbone[pos-1])
				}
				numFields := numFlipPortals * (2*len(backbone) + 1)
				if numFields > bestNumFields {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = pos
					bestBackboneLength = backboneLength - distance(backbone[pos-1], backbone[pos]) + distance(backbone[pos-1], candidate) + distance(candidate, backbone[pos])
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength - distance(backbone[pos-1], backbone[pos]) + distance(backbone[pos-1], candidate) + distance(candidate, backbone[pos])
					if newBackboneLength < bestBackboneLength {
						bestNumFields = numFields
						bestCandidate = i
						bestInsertPosition = pos
						bestBackboneLength = newBackboneLength
					}
				}

			}
		}
		if bestCandidate < 0 {
			pos := len(backbone) - 1
			for i, candidate := range f.portals {
				if backbone[pos].Index == candidate.Index || backbone[0].Index == candidate.Index {
					continue
				}
				var numFlipPortals int
				if ccw {
					if s2.Sign(backbone[0].LatLng, backbone[pos].LatLng, candidate.LatLng) {
						continue
					}
					numFlipPortals = numPortalsLeftOfLine(flipPortals, backbone[pos], candidate)
				} else {
					if s2.Sign(backbone[pos].LatLng, backbone[0].LatLng, candidate.LatLng) {
						continue
					}
					numFlipPortals = numPortalsLeftOfLine(flipPortals, candidate, backbone[pos])
				}

				numFields := numFlipPortals * (2*len(backbone) + 1)
				if numFields > bestNumFields {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = len(backbone)
					bestBackboneLength = backboneLength + distance(backbone[pos], candidate)
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength + distance(backbone[pos], candidate)
					if newBackboneLength < bestBackboneLength {
						bestNumFields = numFields
						bestCandidate = i
						bestInsertPosition = len(backbone)
						bestBackboneLength = newBackboneLength
					}
				}
			}

		}
		if bestCandidate < 0 {
			break
		}
		if bestInsertPosition < len(backbone) {
			backbone = append(backbone, portalData{})
			copy(backbone[bestInsertPosition+1:], backbone[bestInsertPosition:])
			backbone[bestInsertPosition] = candidates[bestCandidate]
			backboneLength = bestBackboneLength
			candidates[bestCandidate], candidates[len(candidates)-1] =
				candidates[len(candidates)-1], candidates[bestCandidate]
			candidates = candidates[:len(candidates)-1]
			// If candidates and flipPortals were the same slice, we must shrink it before partitioning
			// otherwise reordering may bring the removed object back among candidates and confuse the algorithm
			// (and also possibly remove a proper candidate).
			if len(flipPortals) > len(candidates) {
				flipPortals = flipPortals[:len(flipPortals)-1]
			}
			if ccw {
				flipPortals = partitionPortalsLeftOfLine(flipPortals, backbone[bestInsertPosition-1], backbone[bestInsertPosition])
				flipPortals = partitionPortalsLeftOfLine(flipPortals, backbone[bestInsertPosition], backbone[bestInsertPosition+1])
			} else {
				flipPortals = partitionPortalsLeftOfLine(flipPortals, backbone[bestInsertPosition+1], backbone[bestInsertPosition])
				flipPortals = partitionPortalsLeftOfLine(flipPortals, backbone[bestInsertPosition], backbone[bestInsertPosition-1])
			}
		} else {
			backbone = append(backbone, f.portals[bestCandidate])
			backboneLength = bestBackboneLength
			tq := newTriangleQuery(backbone[0].LatLng, backbone[len(backbone)-1].LatLng, backbone[len(backbone)-2].LatLng)
			for _, portal := range f.portals {
				if tq.ContainsPoint(portal.LatLng) {
					foundDup := false
					for _, c := range candidates {
						if c.Index == portal.Index {
							foundDup = true
							break
						}
					}
					if foundDup {
						continue
					}
					for _, c := range backbone {
						if c.Index == portal.Index {
							foundDup = true
							break
						}
					}
					if foundDup {
						continue
					}
					candidates = append(candidates, portal)
				}
			}
			if ccw {
				flipPortals = partitionPortalsLeftOfLine(flipPortals, backbone[len(backbone)-2], backbone[len(backbone)-1])
			} else {
				flipPortals = partitionPortalsLeftOfLine(flipPortals, backbone[len(backbone)-1], backbone[len(backbone)-2])
			}
		}
	}
	return backbone, flipPortals, backboneLength
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
