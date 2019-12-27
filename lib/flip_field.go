package lib

//import "sort"
//import "github.com/golang/geo/r2"

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
		portals:            portals,
		backbone:           make([]portalData, 0, maxBackbonePortals),
		candidates:         make([]portalData, 0, len(portals)),
		simpleBackbone:     simpleBackbone,
	}
}

// Commented out code is for some experiments with using sort+search instead of linear scans
// it's way slower now, but let's keep the code in this experimental branch if we ever want
// to take a look at it
/*type pointByAngle []r2.Point

func (a pointByAngle) Len() int           { return len(a) }
func (a pointByAngle) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a pointByAngle) Less(i, j int) bool { return angleLess(a[i], a[j]) }*/

func (f *bestFlipFieldQuery) findBestFlipField(p0, p1 portalData, ccw bool) ([]portalData, []portalData, float64) {
	f.candidates = f.candidates[:0]
	for _, portal := range f.portals {
		if portal.Index == p0.Index || portal.Index == p1.Index {
			continue
		}
		if (ccw && Sign(p0.LatLng, p1.LatLng, portal.LatLng) <= 0) ||
			(!ccw && Sign(p0.LatLng, p1.LatLng, portal.LatLng) >= 0) {
			continue
		}
		f.candidates = append(f.candidates, portal)
	}
	flipPortals := f.candidates
	//	flipPortalAngles := make([]r2.Point, 0, len(flipPortals))
	f.backbone = append(f.backbone[:0], p0, p1)
	backboneLength := Distance(p0.LatLng, p1.LatLng)
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
			/*flipPortalAngles = flipPortalAngles[:0]
			for _, portal := range flipPortals {
				if portal.Index != candidate.Index {
					flipPortalAngles = append(flipPortalAngles, portal.LatLng.Sub(candidate.LatLng))
				}
			}
			if len(flipPortalAngles)*(2*len(f.backbone)+1) <= bestNumFields {
				continue
			}
			sort.Sort(pointByAngle(flipPortalAngles))*/
			for pos := 1; pos < len(f.backbone); pos++ {
				if f.simpleBackbone {
					q := NewWedgeQuery(f.backbone[0].LatLng, f.backbone[pos-1].LatLng, f.backbone[pos].LatLng)
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
				/*				b0, b1 := f.backbone[pos-1].LatLng, f.backbone[pos].LatLng
								var num0, num1, numFlipPortals int
								var ac bool
								if Sign(b0, b1, candidate.LatLng) > 0 {
									a0, a1 := candidate.LatLng.Sub(b0), candidate.LatLng.Sub(b1)
									num0 = sort.Search(len(flipPortalAngles), func(i int) bool { return angleLess(a0, flipPortalAngles[i]) })
									num1 = sort.Search(len(flipPortalAngles), func(i int) bool { return angleLess(a1, flipPortalAngles[i]) })
									ac = angleLess(a0, a1)
									if !ac && num1 == num0 {
										numFlipPortals = len(flipPortalAngles)
									} else {
										numFlipPortals = num1 - num0
										if numFlipPortals < 0 {
											numFlipPortals += len(flipPortalAngles)
										}
									}
								} else {
									a0, a1 := b0.Sub(candidate.LatLng), b1.Sub(candidate.LatLng)
									num0 = sort.Search(len(flipPortalAngles), func(i int) bool { return angleLess(a0, flipPortalAngles[i]) })
									num1 = sort.Search(len(flipPortalAngles), func(i int) bool { return angleLess(a1, flipPortalAngles[i]) })
									ac = angleLess(a0, a1)
									numFlipPortals = num1 - num0
									if !ac && numFlipPortals == len(flipPortalAngles) {
										numFlipPortals = 0
									}
									if !ac && numFlipPortals < 0 {
										numFlipPortals = -numFlipPortals
									} else if ac {
										numFlipPortals = len(flipPortals) - numFlipPortals
									}
									if numFlipPortals < 0 {
										numFlipPortals += len(flipPortalAngles)
									}
								}*/
				numFields := numFlipPortals * (2*len(f.backbone) + 1)
				if numFields > bestNumFields {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = pos
					bestBackboneLength = backboneLength - Distance(f.backbone[pos-1].LatLng, f.backbone[pos].LatLng) + Distance(f.backbone[pos-1].LatLng, candidate.LatLng) + Distance(candidate.LatLng, f.backbone[pos].LatLng)
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength - Distance(f.backbone[pos-1].LatLng, f.backbone[pos].LatLng) + Distance(f.backbone[pos-1].LatLng, candidate.LatLng) + Distance(candidate.LatLng, f.backbone[pos].LatLng)
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
			break
		}
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
