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
	flipPortals  []portalData
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
		flipPortals:        make([]portalData, 0, len(portals)),
	}
}

func (f *bestFlipFieldQuery) findBestFlipField(p0, p1 portalData, ccw bool) ([]portalData, []portalData, float64) {
	if ccw {
		f.candidates = portalsLeftOfLine(f.portals, p0, p1, f.candidates[:0])
	} else {
		f.candidates = portalsLeftOfLine(f.portals, p1, p0, f.candidates[:0])
	}
	f.flipPortals = append(f.flipPortals[:0], f.candidates...)
	f.backbone = append(f.backbone[:0], p0, p1)
	backboneLength := distance(p0, p1)
	for {
		if len(f.backbone) >= f.maxBackbonePortals {
			break
		}
		if len(f.flipPortals)*(2*f.maxBackbonePortals-1) < f.bestSolution {
			break
		}
		bestNumFields := len(f.flipPortals) * (2*len(f.backbone) - 1)
		bestBackboneLength := backboneLength
		if f.numPortalLimit == EQUAL {
			bestNumFields = 0
		}
		bestCandidate := -1
		bestInsertPosition := -1
		for i, candidate := range f.candidates {
			for pos := 1; pos < len(f.backbone); pos++ {
				if f.simpleBackbone {
					if pos == 1 {
						if ccw != s2.Sign(f.backbone[0].LatLng, f.backbone[1].LatLng, candidate.LatLng) {
							continue
						}
					} else {
						q := newTriangleWedgeQuery(f.backbone[0].LatLng, f.backbone[pos-1].LatLng, f.backbone[pos].LatLng)
						if !q.ContainsPoint(candidate.LatLng) {
							continue
						}
					}
				}
				var numFlipPortals int
				if ccw {
					numFlipPortals = numPortalsLeftOfTwoLines(f.flipPortals, f.backbone[pos-1], candidate, f.backbone[pos])
				} else {
					numFlipPortals = numPortalsLeftOfTwoLines(f.flipPortals, f.backbone[pos], candidate, f.backbone[pos-1])
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
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, f.backbone[pos], candidate)
				} else {
					if s2.Sign(f.backbone[pos].LatLng, f.backbone[0].LatLng, candidate.LatLng) {
						continue
					}
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, candidate, f.backbone[pos])
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
			for i, candidate := range f.portals {
				if f.backbone[len(f.backbone)-1].Index == candidate.Index || f.backbone[0].Index == candidate.Index {
					continue
				}
				var numFlipPortals int
				if ccw == s2.Sign(f.backbone[0].LatLng, f.backbone[len(f.backbone)-1].LatLng, candidate.LatLng) {
					continue
				}
				if f.simpleBackbone {
					ok := true
					for i := 1; i < len(f.backbone); i++ {
						if ccw == s2.Sign(candidate.LatLng, f.backbone[i-1].LatLng, f.backbone[i].LatLng) {
							ok = false
							break
						}
					}
					if !ok {
						continue
					}
				}
				if ccw {
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, candidate, f.backbone[0])
				} else {
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, f.backbone[0], candidate)
				}
				numFields := numFlipPortals * (2*len(f.backbone) + 1)
				if numFields > bestNumFields {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = 0
					bestBackboneLength = backboneLength + distance(f.backbone[0], candidate)
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength + distance(f.backbone[0], candidate)
					if newBackboneLength < bestBackboneLength {
						bestNumFields = numFields
						bestCandidate = i
						bestInsertPosition = 0
						bestBackboneLength = newBackboneLength
					}
				}

			}
		}
		if bestCandidate < 0 {
			break
		}
		if bestInsertPosition == 0 {
			f.backbone = append(f.backbone, portalData{})
			copy(f.backbone[1:], f.backbone[0:])
			f.backbone[0] = f.portals[bestCandidate]
			tq := newTriangleWedgeQuery(f.backbone[len(f.backbone)-1].LatLng, f.backbone[0].LatLng, f.backbone[1].LatLng)
			for _, portal := range f.portals {
				if portal.Index != f.backbone[0].Index &&
					portal.Index != f.backbone[1].Index &&
					portal.Index != f.backbone[len(f.backbone)-1].Index &&
					tq.ContainsPoint(portal.LatLng) {
					f.candidates = append(f.candidates, portal)
				}
			}
			if ccw {
				f.flipPortals = partitionPortalsLeftOfLine(f.flipPortals, f.backbone[0], f.backbone[1])
				f.candidates = partitionPortalsLeftOfLine(f.candidates, f.backbone[0], f.backbone[1])
			} else {
				f.flipPortals = partitionPortalsLeftOfLine(f.flipPortals, f.backbone[1], f.backbone[0])
				f.candidates = partitionPortalsLeftOfLine(f.candidates, f.backbone[1], f.backbone[0])
			}
		} else if bestInsertPosition < len(f.backbone) {
			f.backbone = append(f.backbone, portalData{})
			copy(f.backbone[bestInsertPosition+1:], f.backbone[bestInsertPosition:])
			f.backbone[bestInsertPosition] = f.candidates[bestCandidate]
			backboneLength = bestBackboneLength
			f.candidates[bestCandidate], f.candidates[len(f.candidates)-1] =
				f.candidates[len(f.candidates)-1], f.candidates[bestCandidate]
			f.candidates = f.candidates[:len(f.candidates)-1]
			if ccw {
				f.flipPortals = partitionPortalsLeftOfLine(f.flipPortals, f.backbone[bestInsertPosition-1], f.backbone[bestInsertPosition])
				f.flipPortals = partitionPortalsLeftOfLine(f.flipPortals, f.backbone[bestInsertPosition], f.backbone[bestInsertPosition+1])
			} else {
				f.flipPortals = partitionPortalsLeftOfLine(f.flipPortals, f.backbone[bestInsertPosition+1], f.backbone[bestInsertPosition])
				f.flipPortals = partitionPortalsLeftOfLine(f.flipPortals, f.backbone[bestInsertPosition], f.backbone[bestInsertPosition-1])
			}
		} else {
			f.backbone = append(f.backbone, f.portals[bestCandidate])
			backboneLength = bestBackboneLength
			tq := newTriangleWedgeQuery(f.backbone[0].LatLng, f.backbone[len(f.backbone)-1].LatLng, f.backbone[len(f.backbone)-2].LatLng)
			for _, portal := range f.portals {
				if portal.Index != f.backbone[0].Index &&
					portal.Index != f.backbone[len(f.backbone)-1].Index &&
					portal.Index != f.backbone[len(f.backbone)-2].Index &&
					tq.ContainsPoint(portal.LatLng) {
					f.candidates = append(f.candidates, portal)
				}
			}
			if ccw {
				f.flipPortals = partitionPortalsLeftOfLine(f.flipPortals, f.backbone[len(f.backbone)-2], f.backbone[len(f.backbone)-1])
				f.candidates = partitionPortalsLeftOfLine(f.candidates, f.backbone[len(f.backbone)-2], f.backbone[len(f.backbone)-1])
			} else {
				f.flipPortals = partitionPortalsLeftOfLine(f.flipPortals, f.backbone[len(f.backbone)-1], f.backbone[len(f.backbone)-2])
				f.candidates = partitionPortalsLeftOfLine(f.candidates, f.backbone[len(f.backbone)-1], f.backbone[len(f.backbone)-2])
			}
		}
	}
	if f.numPortalLimit != EQUAL || len(f.backbone) == f.maxBackbonePortals {
		numFlipPortals := len(f.flipPortals)
		if f.maxFlipPortals > 0 && numFlipPortals > f.maxFlipPortals {
			numFlipPortals = f.maxFlipPortals
		}
		numFields := numFlipPortals * (2*len(f.backbone) - 1)
		if numFields > f.bestSolution {
			f.bestSolution = numFields
		}
	}
	return f.backbone, f.flipPortals, backboneLength
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
