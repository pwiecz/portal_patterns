package lib

type bestCobwebQuery struct {
	onFilledIndexEntry func()
	portals            []portalData
	index              []bestSolution
	filteredPortals    [][]portalData
	numPortals         uint
	depth              uint16
}

func newBestCobwebQuery(portals []portalData, onFilledIndexEntry func()) *bestCobwebQuery {
	numPortals := uint(len(portals))
	index := make([]bestSolution, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i].Length = invalidLength
	}
	return &bestCobwebQuery{
		portals:            portals,
		numPortals:         numPortals,
		index:              index,
		onFilledIndexEntry: onFilledIndexEntry,
		filteredPortals:    make([][]portalData, len(portals)),
		depth:              0,
	}
}
func (q *bestCobwebQuery) getIndex(i, j, k portalIndex) bestSolution {
	return q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)]
}
func (q *bestCobwebQuery) setIndex(i, j, k portalIndex, s bestSolution) {
	q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)] = s
}
func (q *bestCobwebQuery) findBestCobweb(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index).Length != invalidLength {
		return
	}
	q.filteredPortals[0] = portalsInsideTriangle(q.portals, p0, p1, p2, q.filteredPortals[0])
	q.findBestCobwebAux(p0, p1, p2, q.filteredPortals[0])
	q.findBestCobwebAux(p0, p2, p1, q.filteredPortals[0])
	q.findBestCobwebAux(p1, p0, p2, q.filteredPortals[0])
	q.findBestCobwebAux(p1, p2, p0, q.filteredPortals[0])
	q.findBestCobwebAux(p2, p0, p1, q.filteredPortals[0])
	q.findBestCobwebAux(p2, p1, p0, q.filteredPortals[0])
}

func (q *bestCobwebQuery) findBestCobwebAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
	q.depth++
	q.filteredPortals[q.depth] = append(q.filteredPortals[q.depth][:0], candidates...)
	var bestCobweb bestSolution
	for _, portal := range q.filteredPortals[q.depth] {
		if q.getIndex(portal.Index, p1.Index, p2.Index).Length == invalidLength {
			candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p1, p2)
			q.findBestCobwebAux(portal, p1, p2, candidatesInWedge)
			q.findBestCobwebAux(portal, p2, p1, candidatesInWedge)
			q.findBestCobwebAux(p1, portal, p2, candidatesInWedge)
			q.findBestCobwebAux(p1, p2, portal, candidatesInWedge)
			q.findBestCobwebAux(p2, portal, p1, candidatesInWedge)
			q.findBestCobwebAux(p2, p1, portal, candidatesInWedge)
		}

		candidate := q.getIndex(p1.Index, p2.Index, portal.Index)
		if candidate.Length+1 > bestCobweb.Length {
			bestCobweb.Length = candidate.Length + 1
			bestCobweb.Index = portal.Index
		}
	}
	q.onFilledIndexEntry()

	q.setIndex(p0.Index, p1.Index, p2.Index, bestCobweb)
	q.depth--
	return bestCobweb
}

// LargestCobweb - Find largest possible cobweb of portals to be made
func LargestCobweb(portals []Portal, fixedCornerIndices []int, progressFunc func(int, int)) []Portal {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

	numIndexEntries := len(portals) * len(portals) * len(portals)
	everyNth := numIndexEntries / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	indexEntriesFilled := 0
	indexEntriesFilledModN := 0
	onFilledIndexEntry := func() {
		indexEntriesFilled++
		indexEntriesFilledModN++
		if indexEntriesFilledModN == everyNth {
			indexEntriesFilledModN = 0
			progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}
	progressFunc(0, numIndexEntries)
	q := newBestCobwebQuery(portalsData, onFilledIndexEntry)
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				if !hasAllIndicesInTheTriple(fixedCornerIndices, i, j, k) {
					continue
				}
				q.findBestCobweb(p0, p1, p2)
			}
		}
	}
	q.filteredPortals = nil
	progressFunc(numIndexEntries, numIndexEntries)

	var bestP0, bestP1, bestP2 portalData
	var bestLength uint16
	for i, p0 := range portalsData {
		for j, p1 := range portalsData {
			if i == j {
				continue
			}
			for k, p2 := range portalsData {
				if i == k || j == k {
					continue
				}
				if !hasAllIndicesInTheTriple(fixedCornerIndices, i, j, k) {
					continue
				}
				candidate := q.getIndex(p0.Index, p1.Index, p2.Index)
				if candidate.Length+3 > bestLength {
					bestP0, bestP1, bestP2 = p0, p1, p2
					bestLength = candidate.Length + 3
				}
			}
		}
	}

	largestCobweb := append(make([]portalIndex, 0, bestLength), bestP0.Index, bestP1.Index, bestP2.Index)
	k0, k1, k2 := bestP0.Index, bestP1.Index, bestP2.Index
	for {
		sol := q.getIndex(k0, k1, k2)
		if sol.Length == 0 {
			break
		}
		largestCobweb = append(largestCobweb, sol.Index)
		k0, k1, k2 = k1, k2, sol.Index
	}
	result := make([]Portal, 0, len(largestCobweb))
	for _, portalIx := range largestCobweb {
		result = append(result, portals[portalIx])
	}
	return result
}

func CobwebPolyline(result []Portal) []Portal {
	if len(result) < 3 {
		return []Portal{}
	}
	portalList := []Portal{result[1], result[0]}
	for _, portal := range result[2:] {
		portalList = append(portalList, portal, portalList[len(portalList)-2])
	}
	return portalList
}
func CobwebDrawToolsString(result []Portal) string {
	return "[\n" + PolylineFromPortalList(CobwebPolyline(result)) + "\n]"
}
