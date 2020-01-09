package lib

import "fmt"
import "sort"
import "github.com/golang/geo/r3"
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
	//	candidates   []portalData
	//	flipPortals  []portalData

}

func newBestFlipFieldQuery(portals []portalData, maxBackbonePortals int, numPortalLimit PortalLimit, maxFlipPortals int, simpleBackbone bool) bestFlipFieldQuery {
	return bestFlipFieldQuery{
		maxBackbonePortals: maxBackbonePortals,
		numPortalLimit:     numPortalLimit,
		maxFlipPortals:     maxFlipPortals,
		simpleBackbone:     simpleBackbone,
		portals:            portals,
		backbone:           make([]portalData, 0, maxBackbonePortals),
		//		candidates:         make([]portalData, 0, len(portals)),
		//		flipPortals:        make([]portalData, 0, len(portals)),
		candidates: make([]flipFieldCandidatePortal, 0, len(portals)),
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

func newByAngleFrom(p0, p1, s portalData, portals []portalData) *byAngleFrom {
	return &byAngleFrom{
		portals: portals,
		p01:     p1.LatLng.Cross(p0.LatLng.Vector).Normalize(),
		s:       s.LatLng.Vector,
	}
}

type byAngleFrom struct {
	portals []portalData
	p01     r3.Vector
	s       r3.Vector
}

func (b *byAngleFrom) angleTo(p portalData) float64 {
	return b.p01.Dot(p.LatLng.Cross(b.s).Normalize())
}
func (b *byAngleFrom) Len() int {
	return len(b.portals)
}
func (b *byAngleFrom) Swap(i, j int) {
	b.portals[i], b.portals[j] = b.portals[j], b.portals[i]
}
func (b *byAngleFrom) Less(i, j int) bool {
	return b.p01.Dot(b.portals[i].LatLng.Cross(b.s).Normalize()) <
		b.p01.Dot(b.portals[j].LatLng.Cross(b.s).Normalize())
}
func (b *byAngleFrom) PortalLess(p portalData, i int) bool {
	return b.p01.Dot(p.LatLng.Cross(b.s).Normalize()) >
		b.p01.Dot(b.portals[i].LatLng.Cross(b.s).Normalize())
}

type byDistFrom struct {
	candidates []flipFieldCandidatePortal
	//p0, p1 s2.Point
	distQuery distanceQuery
}

func (b *byDistFrom) Len() int {
	return len(b.candidates)
}
func (b *byDistFrom) Swap(i, j int) {
	b.candidates[i], b.candidates[j] = b.candidates[j], b.candidates[i]
}
func (b *byDistFrom) Less(i, j int) bool {
	return b.distQuery.ChordAngle(b.candidates[i].latLng) < b.distQuery.ChordAngle(b.canidates[j].latLng)
}

func (f *bestFlipFieldQuery) findBestFlipField(p0, p1 portalData, ccw bool) ([]portalData, []portalData, float64) {
	var candidateCCWQuery ccwQuery
	if ccw {
		candidateCCWQuery = newCCWQuery(p0.LatLng, p1.LatLng)
		//f.candidates = portalsLeftOfLine(f.portals, p0, p1, f.candidates[:0])
	} else {
		candidateCCWQuery = newCCWQuery(p0.LatLng, p1.LatLng)

		//f.candidates = portalsLeftOfLine(f.portals, p1, p0, f.candidates[:0])
	}
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
	//	f.flipPortals = append(f.flipPortals[:0], f.candidates...)
	f.backbone = append(f.backbone[:0], p0, p1)
	backboneLength := distance(p0, p1)
	//	flipPortals0 := make([]portalData, 0, len(f.flipPortals))
	//	flipPortals1 := make([]portalData, 0, len(f.flipPortals))
	for {
		if len(f.backbone) >= f.maxBackbonePortals {
			breakp
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
		//flipPortals0 = append(flipPortals0[:0], f.flipPortals...)
		//flipPortals1 = append(flipPortals1[:0], f.flipPortals...)
		for pos := 1; pos < len(f.backbone); pos++ {
			//byAngle0 := newByAngleFrom(f.backbone[pos-1], f.backbone[pos], f.backbone[pos], flipPortals0)
			//byAngle1 := newByAngleFrom(f.backbone[pos-1], f.backbone[pos], f.backbone[pos-1], flipPortals1)
			byDist := byDistFrom{
				candidates: f.candidates,
				distQuery:  newDistanceQuery(f.backbone[pos-1].LatLng, f.backbone[pos].LatLng),
			}
			sort.Sort(byDist)
			//sort.Sort(byAngle0)
			//sort.Sort(byAngle1)
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
					if ccw != posCCW.IsCCW(candidate.LatLng) {
						continue
					}
					if pos > 1 && ccw == prevPosCCW.IsCCW(candidate.LatLng) {
						continue
					}
				}
				if ccw != segCCW.IsCCW(candidate.LatLng) {
					continue
				}
				var visQ1, visQ2 ccwQuery
				if ccw {
					visQ1 = newCCWQuery(f.backbone[pos-1].LatLng, candidate.LatLng)
					visQ2 = newCCWQuery(candidate.LatLng, f.backbone[pos].LatLng)
					//					numFlipPortals = numPortalsLeftOfTwoLines(f.flipPortals, f.backbone[pos-1], candidate, f.backbone[pos])
					//						n0 := sort.Search(len(flipPortals0), func(i int) bool { return byAngle0.PortalLess(candidate, i) })
					//						n1 := sort.Search(len(flipPortals1), func(i int) bool { return byAngle1.PortalLess(candidate, i) })
					//							fmt.Println("ccw", len(f.flipPortals), n0, n1, "vs", numFlipPortals)
				} else {
					visQ1 = newCCWQuery(f.backbone[pos].LatLng, candidate.LatLng)
					visQ2 = newCCWQuery(candidate.LatLng, f.backbone[pos-1].LatLng)
					//					numFlipPortals = numPortalsLeftOfTwoLines(f.flipPortals, f.backbone[pos], candidate, f.backbone[pos-1])
					//						n0 := sort.Search(len(flipPortals0), func(i int) bool { return byAngle0.PortalLess(candidate, i) })
					//						n1 := sort.Search(len(flipPortals1), func(i int) bool { return byAngle1.PortalLess(candidate, i) })
					//							fmt.Println("cw", len(f.flipPortals), n0, n1, "vs", numFlipPortals)
					//					}
				}
				numFlipPortals := 0
				for j := i + 1; j < len(f.candidates); j++ {
					if !f.candidates[j].isFlipPortal {
						continue
					}
					if visQ1.IsCCW(f.candidates[j].latLng) && visQ2.IsCCW(f.candidates[j].latLng) {
						numFlipPortals++
					}
				}
				if numFlipPortals*(2*f.maxBackbonePortals-1) < f.bestSolution {
					continue
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
				if numFlipPortals*(2*f.maxBackbonePortals-1) < f.bestSolution {
					continue
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
				var numFlipPortals int
				if ccw {
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, candidate, f.backbone[0])
				} else {
					numFlipPortals = numPortalsLeftOfLine(f.flipPortals, f.backbone[0], candidate)
				}
				if numFlipPortals*(2*f.maxBackbonePortals-1) < f.bestSolution {
					continue
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
						portalIndex:        portal.Index,
						latLng:             portal.LatLng,
					})
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
