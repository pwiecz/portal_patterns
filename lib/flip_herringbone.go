package lib

type PortalLimit int

const (
	EQUAL      PortalLimit = 0
	LESS_EQUAL PortalLimit = 1
)

type bestFlipHerringboneQuery struct {
	maxBackbonePortals int
	numPortalLimit     PortalLimit
	backbone           []portalData
	visiblePortals     []portalData
	candidates         []portalData
}

func (f *bestFlipHerringboneQuery) solve(bestSolution int) {
	for {
		if len(f.backbone) >= f.maxBackbonePortals {
			return
		}
		if f.maxBackbonePortals * len(f.visiblePortals) < bestSolution {
			return
		}
		bestNumFields := len(f.visiblePortals) * (2*len(f.backbone) - 1)
		if f.numPortalLimit == EQUAL {
			bestNumFields = 0
		}
		bestCandidate := -1
		bestInsertPosition := -1
		for i, candidate := range f.candidates {
			for pos := 1; pos < len(f.backbone); pos++ {
				numVisiblePortals := numPortalsLeftOfTwoLines(f.visiblePortals, f.backbone[pos-1], candidate, f.backbone[pos])
				numFields := numVisiblePortals * (2*len(f.backbone) + 1)
				if numFields > bestNumFields {
					bestNumFields = numFields
					bestCandidate = i
					bestInsertPosition = pos
				}
			}
		}
		if bestCandidate < 0 {
			return
		}
		f.backbone = append(f.backbone, portalData{})
		copy(f.backbone[bestInsertPosition+1:], f.backbone[bestInsertPosition:])
		f.backbone[bestInsertPosition] = f.candidates[bestCandidate]
		f.candidates[bestCandidate], f.candidates[len(f.candidates)-1] =
			f.candidates[len(f.candidates)-1], f.candidates[bestCandidate]
		f.candidates = f.candidates[:len(f.candidates)-1]
		f.visiblePortals = partitionPortalsLeftOfLine(f.visiblePortals, f.backbone[bestInsertPosition-1], f.backbone[bestInsertPosition])
		f.visiblePortals = partitionPortalsLeftOfLine(f.visiblePortals, f.backbone[bestInsertPosition], f.backbone[bestInsertPosition+1])
	}
}

func findBestFlipHerringbone(p0, p1 portalData, portals []portalData, maxBackbonePortals int, numPortalLimit PortalLimit, bestSolution int) ([]portalData, []portalData) {
	filteredPortals := []portalData{}
	for _, portal := range portals {
		if portal.Index == p0.Index || portal.Index == p1.Index {
			continue
		}
		if Sign(p0.LatLng, p1.LatLng, portal.LatLng) <= 0 {
			continue
		}
		filteredPortals = append(filteredPortals, portal)
	}
	q := bestFlipHerringboneQuery{
		maxBackbonePortals: maxBackbonePortals,
		numPortalLimit:     numPortalLimit,
		backbone:           []portalData{p0, p1},
		visiblePortals:     filteredPortals,
		candidates:         append([]portalData(nil), filteredPortals...),
	}
	q.solve(bestSolution)
	return q.backbone, q.visiblePortals
}

func LargestFlipHerringbone(portals []Portal, maxBackbonePortals int, numPortalLimit PortalLimit, progressFunc func(int, int)) ([]Portal, []Portal) {
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
	for _, p0 := range portalsData {
		for _, p1 := range portalsData {
			if p0.Index == p1.Index {
				continue
			}
			b, f := findBestFlipHerringbone(p0, p1, portalsData, maxBackbonePortals, numPortalLimit, bestNumFields)
			if numPortalLimit != EQUAL || len(b) == maxBackbonePortals {
				numFields := len(f)*(2*len(b)-1)
				distanceSq := DistanceSq(p0.LatLng, p1.LatLng)
				if  numFields > bestNumFields || (numFields == bestNumFields && distanceSq < bestDistanceSq) {
					bestNumFields = numFields
					bestBackbone = b
					bestFlipPortals = f
					bestDistanceSq = distanceSq
				}
			}
			numProcessedPairs++
			numProcessedPairsModN++
			if numProcessedPairsModN == everyNth {
				numProcessedPairsModN = 0
				progressFunc(numProcessedPairs, numPairs)
				//				}
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
