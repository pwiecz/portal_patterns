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
	backbone    []portalData
	candidates  []portalData
	flipPortals []portalData
	portals     []portalData
	// Best solution found so far, we can abort early if we're sure we won't improve current best solution
	bestSolution       int
	maxBackbonePortals int
	numPortalLimit     PortalLimit
	maxFlipPortals     int
	simpleBackbone     bool
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

// Number of portals on the left of lines ab and bc.
// We only check for b to be among the portals.
func numPortalsLeftOfTwoLines(portals []portalData, a, b, c portalData) int {
	result := 0
	ab := newCCWQuery(a.LatLng, b.LatLng)
	bc := newCCWQuery(b.LatLng, c.LatLng)
	for _, p := range portals {
		if p.Index == b.Index {
			continue
		}
		if ab.IsCCW(p.LatLng) && bc.IsCCW(p.LatLng) {
			result++
		}
	}
	return result
}

// Number of portals on the left of line ab.
func numPortalsLeftOfLine(portals []portalData, a, b portalData) int {
	result := 0
	ab := newCCWQuery(a.LatLng, b.LatLng)
	for _, p := range portals {
		if p.Index != a.Index && p.Index != b.Index && ab.IsCCW(p.LatLng) {
			result++
		}
	}
	return result
}

func portalsLeftOfLine(portals []portalData, a, b portalData, result []portalData) []portalData {
	result = result[:0]
	ab := newCCWQuery(a.LatLng, b.LatLng)
	for _, p := range portals {
		if p.Index != a.Index && p.Index != b.Index && ab.IsCCW(p.LatLng) {
			result = append(result, p)
		}
	}
	return result
}

func partitionPortalsLeftOfLine(portals []portalData, a, b portalData) []portalData {
	length := len(portals)
	ab := newCCWQuery(a.LatLng, b.LatLng)
	for i := 0; i < length; {
		p := portals[i]
		if p.Index != a.Index && p.Index != b.Index && ab.IsCCW(p.LatLng) {
			i++
		} else {
			portals[i], portals[length-1] = portals[length-1], portals[i]
			length--
		}
	}
	return portals[:length]
}

func numFlipFields(numFlipPortals, numBackbonePortals int) int {
	return numFlipPortals * (2*numBackbonePortals - 3)
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
	nonBeneficialTriples := make(map[uint64]struct{})
	numAllPortals := uint64(len(f.portals))
	for {
		if len(f.backbone) >= f.maxBackbonePortals {
			break
		}
		if numFlipFields(len(f.flipPortals), f.maxBackbonePortals) < f.bestSolution {
			break
		}
		bestNumFields := numFlipFields(len(f.flipPortals), len(f.backbone))
		bestBackboneLength := backboneLength
		if f.numPortalLimit == EQUAL {
			bestNumFields = 0
		}
		bestCandidate := -1
		bestInsertPosition := -1
		for pos := 1; pos < len(f.backbone); pos++ {
			posCCW := newCCWQuery(f.backbone[0].LatLng, f.backbone[pos].LatLng)
			prevPosCCW := newCCWQuery(f.backbone[0].LatLng, f.backbone[pos-1].LatLng)
			segCCW := newCCWQuery(f.backbone[pos-1].LatLng, f.backbone[pos].LatLng)
			tripleIndexBase := (uint64(f.backbone[pos-1].Index)*numAllPortals + uint64(f.backbone[pos].Index)) * numAllPortals
			for i, candidate := range f.candidates {
				tripleIndex := tripleIndexBase + uint64(candidate.Index)
				if _, ok := nonBeneficialTriples[tripleIndex]; ok {
					continue
				}
				if f.simpleBackbone {
					if ccw != posCCW.IsCCW(candidate.LatLng) {
						continue
					}
					if pos > 1 && ccw == prevPosCCW.IsCCW(candidate.LatLng) {
						continue
					}
				}
				var numFlipPortals int
				// Don't consider candidates "behind" the backbone, they tend not to bring any benefit,
				// and it takes time to check them.
				if ccw != segCCW.IsCCW(candidate.LatLng) {
					continue
				}
				if ccw {
					numFlipPortals = numPortalsLeftOfTwoLines(f.flipPortals, f.backbone[pos-1], candidate, f.backbone[pos])
				} else {
					numFlipPortals = numPortalsLeftOfTwoLines(f.flipPortals, f.backbone[pos], candidate, f.backbone[pos-1])
				}
				if numFlipFields(numFlipPortals, f.maxBackbonePortals) < f.bestSolution {
					nonBeneficialTriples[tripleIndex] = struct{}{}
					continue
				}
				numFields := numFlipFields(numFlipPortals, len(f.backbone)+1)
				if numFields < bestNumFields {
					continue
				}
				newBackboneLength := backboneLength - distance(f.backbone[pos-1], f.backbone[pos]) + distance(f.backbone[pos-1], candidate) + distance(candidate, f.backbone[pos])
				if numFields > bestNumFields || (numFields == bestNumFields && newBackboneLength < bestBackboneLength) {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = pos
					bestBackboneLength = newBackboneLength
				}
			}
		}
		// Check if appending a new backbone portal would improve the solution.
		{
			pos := len(f.backbone) - 1
			zeroLast := newCCWQuery(f.backbone[0].LatLng, f.backbone[pos].LatLng)
			for i, candidate := range f.portals {
				if f.backbone[pos].Index == candidate.Index || f.backbone[0].Index == candidate.Index {
					continue
				}
				if ccw == zeroLast.IsCCW(candidate.LatLng) {
					continue
				}
				var numFlipPortals int
				if ccw {
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, f.backbone[pos], candidate)
				} else {
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, candidate, f.backbone[pos])
				}
				if numFlipFields(numFlipPortals, f.maxBackbonePortals) < f.bestSolution {
					continue
				}
				numFields := numFlipFields(numFlipPortals, len(f.backbone)+1)
				if numFields < bestNumFields {
					continue
				}
				newBackboneLength := backboneLength + distance(f.backbone[pos], candidate)
				if numFields > bestNumFields || (numFields == bestNumFields && newBackboneLength < bestBackboneLength) {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = len(f.backbone)
					bestBackboneLength = newBackboneLength
				}
			}
		}
		// Check if prepending a new backbone portal would improve the solution.
		{
			zeroLast := newCCWQuery(f.backbone[0].LatLng, f.backbone[len(f.backbone)-1].LatLng)
			for i, candidate := range f.portals {
				if f.backbone[len(f.backbone)-1].Index == candidate.Index || f.backbone[0].Index == candidate.Index {
					continue
				}
				if ccw == zeroLast.IsCCW(candidate.LatLng) {
					continue
				}
				if f.simpleBackbone {
					ok := true
					for j := 1; j < len(f.backbone); j++ {
						if ccw == s2.Sign(candidate.LatLng, f.backbone[j-1].LatLng, f.backbone[j].LatLng) {
							ok = false
							break
						}
					}
					if !ok {
						continue
					}
				}
				var numFlipPortals int
				if ccw {
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, candidate, f.backbone[0])
				} else {
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, f.backbone[0], candidate)
				}
				if numFlipFields(numFlipPortals, f.maxBackbonePortals) < f.bestSolution {
					continue
				}
				numFields := numFlipFields(numFlipPortals, len(f.backbone)+1)
				if numFields < bestNumFields {
					continue
				}
				newBackboneLength := backboneLength + distance(f.backbone[0], candidate)
				if numFields > bestNumFields || (numFields == bestNumFields && newBackboneLength < bestBackboneLength) {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = 0
					bestBackboneLength = newBackboneLength
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
			backboneLength = bestBackboneLength
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
		numFields := numFlipFields(numFlipPortals, len(f.backbone))
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
					numFields := numFlipFields(numFlipPortals, len(b))
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
