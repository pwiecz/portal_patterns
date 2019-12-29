package lib

import "github.com/golang/geo/s2"

// LargestFlipField -
func LargestFlipField(portals []Portal, options ...FlipFieldOption) ([]Portal, []Portal) {
	params := defaultFlipFieldParams()
	for _, option := range options {
		option.apply(&params)
	}
	if params.numWorkers == 1 {
		return LargestFlipFieldST(portals, params)
	}
	return LargestFlipFieldMT(portals, params)
}

type PortalLimit int

const (
	EQUAL      PortalLimit = 0
	LESS_EQUAL PortalLimit = 1
)

type bestFlipFieldQuery struct {
	maxBackbonePortals int
	numPortalLimit     PortalLimit
	maxFlipPortals     int
	simpleBackbone     bool
	// Best solution found so far, we can abort early if we're sure we won't improve current best solution
	bestSolution int
	portals      []portalData
	backbone     []portalData
	candidates   []portalData
}

func newBestFlipFieldQuery(portals []portalData, maxBackbonePortals int, numPortalLimit PortalLimit, maxFlipPortals int, simpleBackbone bool) bestFlipFieldQuery {
	return bestFlipFieldQuery{
		maxBackbonePortals: maxBackbonePortals,
		numPortalLimit:     numPortalLimit,
		maxFlipPortals:     maxFlipPortals,
		simpleBackbone:     simpleBackbone,
		portals:            portals,
		backbone:           make([]portalData, 0, maxBackbonePortals),
		candidates:         make([]portalData, 0, len(portals)),
	}
}

func (f *bestFlipFieldQuery) findBestFlipField(p0, p1 portalData, ccw bool) ([]portalData, []portalData, float64) {
	if ccw {
		f.candidates = portalsLeftOfLine(f.portals, p0, p1, f.candidates[:0])
	} else {
		f.candidates = portalsLeftOfLine(f.portals, p1, p0, f.candidates[:0])
	}
	flipPortals := f.candidates
	f.backbone = append(f.backbone[:0], p0, p1)
	backboneLength := distance(p0, p1)
	for {
		if len(f.backbone) >= f.maxBackbonePortals {
			break
		}
		if len(flipPortals)*(2*f.maxBackbonePortals-1) < f.bestSolution {
			break
		}
		bestNumFields := len(flipPortals) * (2*len(f.backbone) - 1)
		bestBackboneLength := backboneLength
		if f.numPortalLimit == EQUAL {
			bestNumFields = 0
		}
		bestCandidate := -1
		bestInsertPosition := -1
		for i, candidate := range f.candidates {
			for pos := 1; pos < len(f.backbone); pos++ {
				if f.simpleBackbone {
					q := newTriangleWedgeQuery(f.backbone[0].LatLng, f.backbone[pos-1].LatLng, f.backbone[pos].LatLng)
					if !q.ContainsPoint(candidate.LatLng) {
						continue
					}
				}
				var numFlipPortals int
				if ccw {
					numFlipPortals = numPortalsLeftOfTwoLines(flipPortals, f.backbone[pos-1], candidate, f.backbone[pos])
				} else {
					numFlipPortals = numPortalsLeftOfTwoLines(flipPortals, f.backbone[pos], candidate, f.backbone[pos-1])
				}
				numFields := numFlipPortals * (2*len(f.backbone) + 1)
				if numFields > bestNumFields {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = pos
					bestBackboneLength = backboneLength - distance(f.backbone[pos-1], f.backbone[pos]) + distance(f.backbone[pos-1], candidate) + distance(candidate, f.backbone[pos])
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength - distance(f.backbone[pos-1], f.backbone[pos]) + distance(f.backbone[pos-1], candidate) + distance(candidate, f.backbone[pos])
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
			pos := len(f.backbone) - 1
			for i, candidate := range f.portals {
				if f.backbone[pos].Index == candidate.Index || f.backbone[0].Index == candidate.Index {
					continue
				}
				var numFlipPortals int
				if ccw {
					if s2.Sign(f.backbone[0].LatLng, f.backbone[pos].LatLng, candidate.LatLng) {
						continue
					}
					numFlipPortals = numPortalsLeftOfLine(flipPortals, f.backbone[pos], candidate)
				} else {
					if s2.Sign(f.backbone[pos].LatLng, f.backbone[0].LatLng, candidate.LatLng) {
						continue
					}
					numFlipPortals = numPortalsLeftOfLine(flipPortals, candidate, f.backbone[pos])
				}

				numFields := numFlipPortals * (2*len(f.backbone) + 1)
				if numFields > bestNumFields {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = len(f.backbone)
					bestBackboneLength = backboneLength + distance(f.backbone[pos], candidate)
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength + distance(f.backbone[pos], candidate)
					if newBackboneLength < bestBackboneLength {
						bestNumFields = numFields
						bestCandidate = i
						bestInsertPosition = len(f.backbone)
						bestBackboneLength = newBackboneLength
					}
				}
			}
		}
		if bestCandidate < 0 {
			break
		}
		if bestInsertPosition < len(f.backbone) {
			f.backbone = append(f.backbone, portalData{})
			copy(f.backbone[bestInsertPosition+1:], f.backbone[bestInsertPosition:])
			f.backbone[bestInsertPosition] = f.candidates[bestCandidate]
			backboneLength = bestBackboneLength
			f.candidates[bestCandidate], f.candidates[len(f.candidates)-1] =
				f.candidates[len(f.candidates)-1], f.candidates[bestCandidate]
			f.candidates = f.candidates[:len(f.candidates)-1]
			// If candidates and flipPortals were the same slice, we must shrink it before partitioning
			// otherwise reordering may bring the removed object back among candidates and confuse the algorithm
			// (and also possibly remove a proper candidate).
			if len(flipPortals) > len(f.candidates) {
				flipPortals = flipPortals[:len(flipPortals)-1]
			}
			if ccw {
				flipPortals = partitionPortalsLeftOfLine(flipPortals, f.backbone[bestInsertPosition-1], f.backbone[bestInsertPosition])
				flipPortals = partitionPortalsLeftOfLine(flipPortals, f.backbone[bestInsertPosition], f.backbone[bestInsertPosition+1])
			} else {
				flipPortals = partitionPortalsLeftOfLine(flipPortals, f.backbone[bestInsertPosition+1], f.backbone[bestInsertPosition])
				flipPortals = partitionPortalsLeftOfLine(flipPortals, f.backbone[bestInsertPosition], f.backbone[bestInsertPosition-1])
			}
		} else {
			f.backbone = append(f.backbone, f.portals[bestCandidate])
			backboneLength = bestBackboneLength
			tq := newTriangleQuery(f.backbone[0].LatLng, f.backbone[len(f.backbone)-1].LatLng, f.backbone[len(f.backbone)-2].LatLng)
			for _, portal := range f.portals {
				if tq.ContainsPoint(portal.LatLng) {
					foundDup := false
					for _, c := range f.candidates {
						if c.Index == portal.Index {
							foundDup = true
							break
						}
					}
					if foundDup {
						continue
					}
					for _, c := range f.backbone {
						if c.Index == portal.Index {
							foundDup = true
							break
						}
					}
					if foundDup {
						continue
					}
					f.candidates = append(f.candidates, portal)
				}
			}
			if ccw {
				flipPortals = partitionPortalsLeftOfLine(flipPortals, f.backbone[len(f.backbone)-2], f.backbone[len(f.backbone)-1])
			} else {
				flipPortals = partitionPortalsLeftOfLine(flipPortals, f.backbone[len(f.backbone)-1], f.backbone[len(f.backbone)-2])
			}
		}
	}
	if f.numPortalLimit != EQUAL || len(f.backbone) == f.maxBackbonePortals {
		numFlipPortals := len(flipPortals)
		if f.maxFlipPortals > 0 && numFlipPortals > f.maxFlipPortals {
			numFlipPortals = f.maxFlipPortals
		}
		numFields := numFlipPortals * (2*len(f.backbone) - 1)
		if numFields > f.bestSolution {
			f.bestSolution = numFields
		}
	}
	return f.backbone, flipPortals, backboneLength
}

func LargestFlipFieldST(portals []Portal, params flipFieldParams) ([]Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

	numPairs := len(portals) * (len(portals) - 1)
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
	q := newBestFlipFieldQuery(portalsData, params.maxBackbonePortals, params.backbonePortalLimit, params.maxFlipPortals, params.simpleBackbone)
	for _, p0 := range portalsData {
		for _, p1 := range portalsData {
			if p0.Index == p1.Index {
				continue
			}
			for _, ccw := range []bool{true, false} {
				b, f, bl := q.findBestFlipField(p0, p1, ccw)
				if params.backbonePortalLimit != EQUAL || len(b) == params.maxBackbonePortals {
					numFlipPortals := len(f)
					if params.maxFlipPortals > 0 && numFlipPortals > params.maxFlipPortals {
						numFlipPortals = params.maxFlipPortals
					}
					numFields := numFlipPortals * (2*len(b) - 1)
					if numFields > bestNumFields || (numFields == bestNumFields && bl < bestBackboneLength) {
						bestNumFields = numFields
						bestBackbone = append(bestBackbone[:0], b...)
						bestFlipPortals = append(bestFlipPortals[:0], f...)
						bestBackboneLength = bl
					}
				}
			}
			numProcessedPairs++
			numProcessedPairsModN++
			if numProcessedPairsModN == everyNth {
				numProcessedPairsModN = 0
				params.progressFunc(numProcessedPairs, numPairs)
			}
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
