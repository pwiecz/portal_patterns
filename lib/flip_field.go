package lib

import "github.com/golang/geo/s2"

// LargestFlipField -
func LargestFlipField(portals []Portal, maxBackbonePortals int, numPortalLimit PortalLimit, numWorkers int, progressFunc func(int, int)) ([]Portal, []Portal) {
	if numWorkers == 1 {
		return LargestFlipFieldST(portals, maxBackbonePortals, numPortalLimit, progressFunc)
	}
	return LargestFlipFieldMT(portals, maxBackbonePortals, numPortalLimit, numWorkers, progressFunc)
}

type PortalLimit int

const (
	EQUAL      PortalLimit = 0
	LESS_EQUAL PortalLimit = 1
)

type bestFlipFieldQuery struct {
	maxBackbonePortals int
	numPortalLimit     PortalLimit
	bestSolution       int // best solution found so far
	portals            []portalData
	backbone           []portalData
	candidates         []portalData
}

func newBestFlipFieldQuery(portals []portalData, maxBackbonePortals int, numPortalLimit PortalLimit) bestFlipFieldQuery {
	return bestFlipFieldQuery{
		maxBackbonePortals: maxBackbonePortals,
		numPortalLimit:     numPortalLimit,
		portals:            portals,
		backbone:           make([]portalData, 0, maxBackbonePortals),
		candidates:         make([]portalData, 0, len(portals)),
	}
}

func (f *bestFlipFieldQuery) solve(bestSolution int) {
}

func (f *bestFlipFieldQuery) findBestFlipField(p0, p1 portalData) ([]portalData, []portalData) {
	f.candidates = f.candidates[:0]
	for _, portal := range f.portals {
		if portal.Index == p0.Index || portal.Index == p1.Index {
			continue
		}
		if !s2.Sign(p0.LatLng, p1.LatLng, portal.LatLng) {
			continue
		}
		f.candidates = append(f.candidates, portal)
	}
	flipPortals := f.candidates
	f.backbone = append(f.backbone[:0], p0, p1)
	for {
		if len(f.backbone) >= f.maxBackbonePortals {
			break
		}
		if len(flipPortals)*(2*f.maxBackbonePortals-1) < f.bestSolution {
			break
		}
		bestNumFields := len(flipPortals) * (2*len(f.backbone) - 1)
		if f.numPortalLimit == EQUAL {
			bestNumFields = 0
		}
		bestCandidate := -1
		bestInsertPosition := -1
		for i, candidate := range f.candidates {
			for pos := 1; pos < len(f.backbone); pos++ {
				numFlipPortals := numPortalsLeftOfTwoLines(flipPortals, f.backbone[pos-1], candidate, f.backbone[pos])
				numFields := numFlipPortals * (2*len(f.backbone) + 1)
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
		f.backbone = append(f.backbone, portalData{})
		copy(f.backbone[bestInsertPosition+1:], f.backbone[bestInsertPosition:])
		f.backbone[bestInsertPosition] = f.candidates[bestCandidate]
		f.candidates[bestCandidate], f.candidates[len(f.candidates)-1] =
			f.candidates[len(f.candidates)-1], f.candidates[bestCandidate]
		f.candidates = f.candidates[:len(f.candidates)-1]
		flipPortals = partitionPortalsLeftOfLine(flipPortals, f.backbone[bestInsertPosition-1], f.backbone[bestInsertPosition])
		flipPortals = partitionPortalsLeftOfLine(flipPortals, f.backbone[bestInsertPosition], f.backbone[bestInsertPosition+1])
	}
	if f.numPortalLimit != EQUAL || len(f.backbone) == f.maxBackbonePortals {
		numFields := len(flipPortals) * (2*len(f.backbone) - 1)
		if numFields > f.bestSolution {
			f.bestSolution = numFields
		}
	}
	return f.backbone, flipPortals
}

func LargestFlipFieldST(portals []Portal, maxBackbonePortals int, numPortalLimit PortalLimit, progressFunc func(int, int)) ([]Portal, []Portal) {
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
	progressFunc(0, numPairs)

	var bestNumFields int
	bestBackbone, bestFlipPortals := []portalData(nil), []portalData(nil)
	var bestDistanceSq float64
	q := newBestFlipFieldQuery(portalsData, maxBackbonePortals, numPortalLimit)
	for _, p0 := range portalsData {
		for _, p1 := range portalsData {
			if p0.Index == p1.Index {
				continue
			}
			b, f := q.findBestFlipField(p0, p1)
			if numPortalLimit != EQUAL || len(b) == maxBackbonePortals {
				numFields := len(f) * (2*len(b) - 1)
				distanceSq := distanceSq(p0, p1)
				if numFields > bestNumFields || (numFields == bestNumFields && distanceSq < bestDistanceSq) {
					bestNumFields = numFields
					bestBackbone = append(bestBackbone[:0], b...)
					bestFlipPortals = append(bestFlipPortals[:0], f...)
					bestDistanceSq = distanceSq
				}
			}
			numProcessedPairs++
			numProcessedPairsModN++
			if numProcessedPairsModN == everyNth {
				numProcessedPairsModN = 0
				progressFunc(numProcessedPairs, numPairs)
			}
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
