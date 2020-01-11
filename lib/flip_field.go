package lib

import "sort"
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

type flipFieldCandidatePortal struct {
	isFlipPortal       bool
	visitedInThisRound bool
	index              portalIndex
	latLng             s2.Point
}

type bestFlipFieldQuery struct {
	maxBackbonePortals int
	numPortalLimit     PortalLimit
	maxFlipPortals     int
	simpleBackbone     bool
	// Best solution found so far, we can abort early if we're sure we won't improve current best solution
	bestSolution int
	portals      []portalData
	backbone     []portalData
	candidates   []flipFieldCandidatePortal
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
		flipPortals:        make([]portalData, 0, len(portals)),
		candidates:         make([]flipFieldCandidatePortal, 0, len(portals)),
	}
}

type byDistFrom struct {
	candidates []flipFieldCandidatePortal
	distQuery  distanceQuery
}

func (b *byDistFrom) Len() int {
	return len(b.candidates)
}
func (b *byDistFrom) Swap(i, j int) {
	b.candidates[i], b.candidates[j] = b.candidates[j], b.candidates[i]
}
func (b *byDistFrom) Less(i, j int) bool {
	return b.distQuery.ChordAngle(b.candidates[i].latLng) < b.distQuery.ChordAngle(b.candidates[j].latLng)
}

func (f *bestFlipFieldQuery) findBestFlipField(p0, p1 portalData, ccw bool) ([]portalData, []portalData, float64) {
	var candidateCCWQuery ccwQuery
	if ccw {
		candidateCCWQuery = newCCWQuery(p0.LatLng, p1.LatLng)
	} else {
		candidateCCWQuery = newCCWQuery(p1.LatLng, p0.LatLng)
	}
	f.candidates = f.candidates[:0]
	for _, portal := range f.portals {
		if portal.Index != p0.Index && portal.Index != p1.Index && candidateCCWQuery.IsCCW(portal.LatLng) {
			f.candidates = append(f.candidates, flipFieldCandidatePortal{
				isFlipPortal:       true,
				visitedInThisRound: false,
				index:              portal.Index,
				latLng:             portal.LatLng,
			})
		}
	}
	f.backbone = append(f.backbone[:0], p0, p1)
	backboneLength := distance(p0.LatLng, p1.LatLng)
	bestNumFlipPortals := len(f.candidates)
	for {
		if len(f.backbone) >= f.maxBackbonePortals {
			break
		}
		if bestNumFlipPortals*(2*f.maxBackbonePortals-1) < f.bestSolution {
			break
		}
		bestNumFields := bestNumFlipPortals * (2*len(f.backbone) - 1)
		bestBackboneLength := backboneLength
		if f.numPortalLimit == EQUAL {
			bestNumFields = 0
		}
		bestCandidate := -1
		bestInsertPosition := -1
		for pos := 1; pos < len(f.backbone); pos++ {
			byDist := &byDistFrom{
				candidates: f.candidates,
				distQuery:  newDistanceQuery(f.backbone[pos-1].LatLng, f.backbone[pos].LatLng),
			}
			sort.Sort(byDist)
			posCCW := newCCWQuery(f.backbone[0].LatLng, f.backbone[pos].LatLng)
			prevPosCCW := newCCWQuery(f.backbone[0].LatLng, f.backbone[pos-1].LatLng)
			segCCW := newCCWQuery(f.backbone[pos-1].LatLng, f.backbone[pos].LatLng)
			for i := range f.candidates {
				f.candidates[i].visitedInThisRound = false
			}
			for i, candidate := range f.candidates {
				if candidate.visitedInThisRound {
					continue
				}
				if f.simpleBackbone {
					if ccw != posCCW.IsCCW(candidate.latLng) {
						continue
					}
					if pos > 1 && ccw == prevPosCCW.IsCCW(candidate.latLng) {
						continue
					}
				}
				// Don't consider candidates "behind" the backbone, they don't tend to bring any benefit,
				// and it takes time to check them.
				if ccw != segCCW.IsCCW(candidate.latLng) {
					continue
				}
				var visQ1, visQ2 ccwQuery
				if ccw {
					visQ1 = newCCWQuery(f.backbone[pos-1].LatLng, candidate.latLng)
					visQ2 = newCCWQuery(candidate.latLng, f.backbone[pos].LatLng)
				} else {
					visQ1 = newCCWQuery(f.backbone[pos].LatLng, candidate.latLng)
					visQ2 = newCCWQuery(candidate.latLng, f.backbone[pos-1].LatLng)
				}
				var numFlipPortals int
				for j := i + 1; j < len(f.candidates); j++ {
					if visQ1.IsCCW(f.candidates[j].latLng) && visQ2.IsCCW(f.candidates[j].latLng) {
						f.candidates[j].visitedInThisRound = true
						if f.candidates[j].isFlipPortal {
							numFlipPortals++
						}
					}
				}
				if numFlipPortals*(2*f.maxBackbonePortals-1) < f.bestSolution {
					continue
				}
				numFields := numFlipPortals * (2*len(f.backbone) + 1)
				if numFields > bestNumFields {
					bestNumFlipPortals = numFlipPortals
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = pos
					bestBackboneLength = backboneLength - distance(f.backbone[pos-1].LatLng, f.backbone[pos].LatLng) + distance(f.backbone[pos-1].LatLng, candidate.latLng) + distance(candidate.latLng, f.backbone[pos].LatLng)
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength - distance(f.backbone[pos-1].LatLng, f.backbone[pos].LatLng) + distance(f.backbone[pos-1].LatLng, candidate.latLng) + distance(candidate.latLng, f.backbone[pos].LatLng)
					if newBackboneLength < bestBackboneLength {
						bestNumFlipPortals = numFlipPortals
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
			zeroLast := newCCWQuery(f.backbone[0].LatLng, f.backbone[pos].LatLng)
			for i, candidate := range f.portals {
				if f.backbone[pos].Index == candidate.Index || f.backbone[0].Index == candidate.Index {
					continue
				}
				if ccw == zeroLast.IsCCW(candidate.LatLng) {
					continue
				}
				var visQ ccwQuery
				if ccw {
					visQ = newCCWQuery(f.backbone[pos].LatLng, candidate.LatLng)
				} else {
					visQ = newCCWQuery(candidate.LatLng, f.backbone[pos].LatLng)
				}
				var numFlipPortals int
				for _, p := range f.candidates {
					if !p.isFlipPortal {
						continue
					}
					if visQ.IsCCW(p.latLng) {
						numFlipPortals++
					}
				}
				if numFlipPortals*(2*f.maxBackbonePortals-1) < f.bestSolution {
					continue
				}
				numFields := numFlipPortals * (2*len(f.backbone) + 1)
				if numFields > bestNumFields {
					bestNumFlipPortals = numFlipPortals
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = len(f.backbone)
					bestBackboneLength = backboneLength + distance(f.backbone[pos].LatLng, candidate.LatLng)
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength + distance(f.backbone[pos].LatLng, candidate.LatLng)
					if newBackboneLength < bestBackboneLength {
						bestNumFlipPortals = numFlipPortals
						bestNumFields = numFields
						bestCandidate = i
						bestInsertPosition = len(f.backbone)
						bestBackboneLength = newBackboneLength
					}
				}
			}
		}
		if bestCandidate < 0 {
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
				var visQ ccwQuery
				if ccw {
					visQ = newCCWQuery(candidate.LatLng, f.backbone[0].LatLng)
				} else {
					visQ = newCCWQuery(f.backbone[0].LatLng, candidate.LatLng)
				}
				var numFlipPortals int
				for _, p := range f.candidates {
					if !p.isFlipPortal {
						continue
					}
					if visQ.IsCCW(p.latLng) {
						numFlipPortals++
					}
				}
				if numFlipPortals*(2*f.maxBackbonePortals-1) < f.bestSolution {
					continue
				}
				numFields := numFlipPortals * (2*len(f.backbone) + 1)
				if numFields > bestNumFields {
					bestNumFlipPortals = numFlipPortals
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = 0
					bestBackboneLength = backboneLength + distance(f.backbone[0].LatLng, candidate.LatLng)
				} else if numFields == bestNumFields {
					newBackboneLength := backboneLength + distance(f.backbone[0].LatLng, candidate.LatLng)
					if newBackboneLength < bestBackboneLength {
						bestNumFlipPortals = numFlipPortals
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
			backboneLength = bestBackboneLength
			tq := newTriangleWedgeQuery(f.backbone[len(f.backbone)-1].LatLng, f.backbone[0].LatLng, f.backbone[1].LatLng)
			for _, portal := range f.portals {
				if portal.Index != f.backbone[0].Index &&
					portal.Index != f.backbone[1].Index &&
					portal.Index != f.backbone[len(f.backbone)-1].Index &&
					tq.ContainsPoint(portal.LatLng) {
					f.candidates = append(f.candidates, flipFieldCandidatePortal{
						isFlipPortal:       false,
						visitedInThisRound: false,
						index:              portal.Index,
						latLng:             portal.LatLng,
					})
				}
			}
			var zeroOneCCW ccwQuery
			if ccw {
				zeroOneCCW = newCCWQuery(f.backbone[0].LatLng, f.backbone[1].LatLng)
			} else {
				zeroOneCCW = newCCWQuery(f.backbone[1].LatLng, f.backbone[0].LatLng)
			}
			for i := range f.candidates {
				if !zeroOneCCW.IsCCW(f.candidates[i].latLng) {
					f.candidates[i].isFlipPortal = false
				}
			}
		} else if bestInsertPosition < len(f.backbone) {
			f.backbone = append(f.backbone, portalData{})
			copy(f.backbone[bestInsertPosition+1:], f.backbone[bestInsertPosition:])
			f.backbone[bestInsertPosition] = portalData{
				LatLng: f.candidates[bestCandidate].latLng,
				Index:  f.candidates[bestCandidate].index,
			}
			backboneLength = bestBackboneLength
			f.candidates[bestCandidate], f.candidates[len(f.candidates)-1] =
				f.candidates[len(f.candidates)-1], f.candidates[bestCandidate]
			f.candidates = f.candidates[:len(f.candidates)-1]
			var seg1CCW, seg2CCW ccwQuery
			if ccw {
				seg1CCW = newCCWQuery(f.backbone[bestInsertPosition-1].LatLng, f.backbone[bestInsertPosition].LatLng)
				seg2CCW = newCCWQuery(f.backbone[bestInsertPosition].LatLng, f.backbone[bestInsertPosition+1].LatLng)
			} else {
				seg1CCW = newCCWQuery(f.backbone[bestInsertPosition+1].LatLng, f.backbone[bestInsertPosition].LatLng)
				seg2CCW = newCCWQuery(f.backbone[bestInsertPosition].LatLng, f.backbone[bestInsertPosition-1].LatLng)
			}
			for i := range f.candidates {
				if !seg1CCW.IsCCW(f.candidates[i].latLng) || !seg2CCW.IsCCW(f.candidates[i].latLng) {
					f.candidates[i].isFlipPortal = false
				}
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
					f.candidates = append(f.candidates, flipFieldCandidatePortal{
						isFlipPortal:       false,
						visitedInThisRound: false,
						index:              portal.Index,
						latLng:             portal.LatLng,
					})
				}
			}
			var lastTwoCCW ccwQuery
			var baseCCW ccwQuery
			if ccw {
				lastTwoCCW = newCCWQuery(f.backbone[len(f.backbone)-2].LatLng, f.backbone[len(f.backbone)-1].LatLng)
				baseCCW = newCCWQuery(f.backbone[0].LatLng, f.backbone[len(f.backbone)-1].LatLng)
			} else {
				lastTwoCCW = newCCWQuery(f.backbone[len(f.backbone)-1].LatLng, f.backbone[len(f.backbone)-2].LatLng)
				baseCCW = newCCWQuery(f.backbone[len(f.backbone)-1].LatLng, f.backbone[0].LatLng)
			}
			for i := range f.candidates {
				if !lastTwoCCW.IsCCW(f.candidates[i].latLng) {
					f.candidates[i].isFlipPortal = false
				}
			}
			for i := 0; i < len(f.candidates); {
				if !baseCCW.IsCCW(f.candidates[i].latLng) {
					f.candidates[i], f.candidates[len(f.candidates)-1] = f.candidates[len(f.candidates)-1], f.candidates[i]
					f.candidates = f.candidates[:len(f.candidates)-1]
				} else {
					i++
				}
			}
		}
	}
	if f.numPortalLimit != EQUAL || len(f.backbone) == f.maxBackbonePortals {
		numFlipPortals := bestNumFlipPortals
		if f.maxFlipPortals > 0 && numFlipPortals > f.maxFlipPortals {
			numFlipPortals = f.maxFlipPortals
		}
		numFields := numFlipPortals * (2*len(f.backbone) - 1)
		if numFields > f.bestSolution {
			f.bestSolution = numFields
		}
	}
	f.flipPortals = f.flipPortals[:0]
	for _, portal := range f.candidates {
		if portal.isFlipPortal {
			f.flipPortals = append(f.flipPortals, portalData{
				Index:  portal.index,
				LatLng: portal.latLng,
			})
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
