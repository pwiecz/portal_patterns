package lib
import "fmt"
type bestThreeCornersQuery struct {
	portals0           []portalData
	numPortals0        portalIndex
	portals1           []portalData
	numPortals1        portalIndex
	portals2           []portalData
	numPortals2        uint
	numPortals1x2      uint
	index              []bestSolution
	numCornerChanges   []uint16
	onIndexEntryFilled func()
	// preallocated storage for lists of portals within triangles at consecutive recursion depths
	portalsInTriangle0 [][]portalData
	portalsInTriangle1 [][]portalData
	portalsInTriangle2 [][]portalData
	// current recursion depth
	depth uint16
}

func newBestThreeCornersQuery(portals0, portals1, portals2 []portalData, onIndexEntryFilled func()) *bestThreeCornersQuery {
	numPortals0x1x2 := uint(len(portals0)) * uint(len(portals1)) * uint(len(portals2))
	index := make([]bestSolution, numPortals0x1x2)
	numCornerChanges := make([]uint16, numPortals0x1x2)
	for i := 0; i < len(index); i++ {
		index[i].Length = invalidLength
	}
	return &bestThreeCornersQuery{
		portals0:           append(make([]portalData, 0, len(portals0)), portals0...),
		numPortals0:        portalIndex(len(portals0)),
		portals1:           append(make([]portalData, 0, len(portals1)), portals1...),
		numPortals1:        portalIndex(len(portals1)),
		portals2:           append(make([]portalData, 0, len(portals2)), portals2...),
		numPortals2:        uint(len(portals2)),
		numPortals1x2:      uint(len(portals1)) * uint(len(portals2)),
		index:              index,
		numCornerChanges:   numCornerChanges,
		onIndexEntryFilled: onIndexEntryFilled,
		portalsInTriangle0: make([][]portalData, len(portals0)+len(portals1)+len(portals2)),
		portalsInTriangle1: make([][]portalData, len(portals0)+len(portals1)+len(portals2)),
		portalsInTriangle2: make([][]portalData, len(portals0)+len(portals1)+len(portals2)),
	}
}

func (q *bestThreeCornersQuery) getIndex(i0, i1, i2 portalIndex) bestSolution {
	return q.index[uint(i0)*q.numPortals1x2+uint(i1)*q.numPortals2+uint(i2)]
}
func (q *bestThreeCornersQuery) setIndex(i0, i1, i2 portalIndex, s bestSolution) {
	q.index[uint(i0)*q.numPortals1x2+uint(i1)*q.numPortals2+uint(i2)] = s
}
func (q *bestThreeCornersQuery) getNumCornerChanges(i0, i1, i2 portalIndex) uint16 {
	return q.numCornerChanges[uint(i0)*q.numPortals1x2+uint(i1)*q.numPortals2+uint(i2)]
}
func (q *bestThreeCornersQuery) setNumCornerChanges(i0, i1, i2 portalIndex, n uint16) {
	q.numCornerChanges[uint(i0)*q.numPortals1x2+uint(i1)*q.numPortals2+uint(i2)] = n
}
func (q *bestThreeCornersQuery) findBestThreeCorner(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index).Length != invalidLength {
		return
	}
	q.portalsInTriangle0[0] = portalsInsideTriangle(q.portals0, p0, p1, p2, q.portalsInTriangle0[0])
	q.portalsInTriangle1[0] = portalsInsideTriangle(q.portals1, p0, p1, p2, q.portalsInTriangle1[0])
	q.portalsInTriangle2[0] = portalsInsideTriangle(q.portals2, p0, p1, p2, q.portalsInTriangle2[0])
	q.findBestThreeCornerAux(p0, p1, p2, q.portalsInTriangle0[0], q.portalsInTriangle1[0], q.portalsInTriangle2[0])
}
func (q *bestThreeCornersQuery) findBestThreeCornerAux(p0, p1, p2 portalData, candidates0, candidates1, candidates2 []portalData) (bestSolution, uint16) {
	q.depth++
	q.portalsInTriangle0[q.depth] = append(q.portalsInTriangle0[q.depth][:0], candidates0...)
	q.portalsInTriangle1[q.depth] = append(q.portalsInTriangle1[q.depth][:0], candidates1...)
	q.portalsInTriangle2[q.depth] = append(q.portalsInTriangle2[q.depth][:0], candidates2...)
	var bestTC bestSolution
	var bestNumCornerChanges uint16
	for _, portal := range q.portalsInTriangle0[q.depth] {
		candidate := q.getIndex(portal.Index, p1.Index, p2.Index)
		numCornerChanges := q.getNumCornerChanges(portal.Index, p1.Index, p2.Index)
		if candidate.Length == invalidLength {
			candidatesInWedge0 := partitionPortalsInsideWedge(candidates0, portal, p1, p2)
			candidatesInWedge1 := partitionPortalsInsideWedge(candidates1, portal, p1, p2)
			candidatesInWedge2 := partitionPortalsInsideWedge(candidates2, portal, p1, p2)
			candidate, numCornerChanges = q.findBestThreeCornerAux(portal, p1, p2, candidatesInWedge0, candidatesInWedge1, candidatesInWedge2)
			if candidate.Length > 0 && candidate.Index >= q.numPortals0 {
				numCornerChanges = numCornerChanges + 1
			}
		}
		candidate.Length = candidate.Length + 1
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && numCornerChanges < bestNumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index
			bestNumCornerChanges = numCornerChanges
		}
	}
	for _, portal := range q.portalsInTriangle1[q.depth] {
		candidate := q.getIndex(p0.Index, portal.Index, p2.Index)
		numCornerChanges := q.getNumCornerChanges(p0.Index, portal.Index, p2.Index)
		if candidate.Length == invalidLength {
			candidatesInWedge0 := partitionPortalsInsideWedge(candidates0, portal, p0, p2)
			candidatesInWedge1 := partitionPortalsInsideWedge(candidates1, portal, p0, p2)
			candidatesInWedge2 := partitionPortalsInsideWedge(candidates2, portal, p0, p2)
			candidate, numCornerChanges = q.findBestThreeCornerAux(p0, portal, p2, candidatesInWedge0, candidatesInWedge1, candidatesInWedge2)
			if candidate.Length > 0 && (candidate.Index < q.numPortals0 || candidate.Index >= q.numPortals0+q.numPortals1) {
				numCornerChanges = numCornerChanges + 1
			}
		}
		candidate.Length = candidate.Length + 1
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && numCornerChanges < bestNumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index + q.numPortals0
			bestNumCornerChanges = numCornerChanges
		}
	}
	for _, portal := range q.portalsInTriangle2[q.depth] {
		candidate := q.getIndex(p0.Index, p1.Index, portal.Index)
		numCornerChanges := q.getNumCornerChanges(p0.Index, p1.Index, portal.Index)
		if candidate.Length == invalidLength {
			candidatesInWedge0 := partitionPortalsInsideWedge(candidates0, portal, p0, p1)
			candidatesInWedge1 := partitionPortalsInsideWedge(candidates1, portal, p0, p1)
			candidatesInWedge2 := partitionPortalsInsideWedge(candidates2, portal, p0, p1)
			candidate, numCornerChanges = q.findBestThreeCornerAux(p0, p1, portal, candidatesInWedge0, candidatesInWedge1, candidatesInWedge2)
			if candidate.Length > 0 && candidate.Index < q.numPortals0+q.numPortals1 {
				numCornerChanges = numCornerChanges + 1
			}
		}
		candidate.Length = candidate.Length + 1
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && numCornerChanges < bestNumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index + q.numPortals0 + q.numPortals1
			bestNumCornerChanges = numCornerChanges
		}
	}
	q.setIndex(p0.Index, p1.Index, p2.Index, bestTC)
	q.setNumCornerChanges(p0.Index, p1.Index, p2.Index, bestNumCornerChanges)
	q.onIndexEntryFilled()
	q.depth--
	return bestTC, bestNumCornerChanges
}

// LargestThreeCorner - Find best way to connect three groups of portals
func LargestThreeCorner(portals0, portals1, portals2 []Portal, progressFunc func(int, int)) []IndexedPortal {
	portalsData0 := portalsToPortalData(portals0)
	portalsData1 := portalsToPortalData(portals1)
	portalsData2 := portalsToPortalData(portals2)

	numIndexEntries := len(portals0) * len(portals1) * len(portals2)
	everyNth := numIndexEntries / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	indexEntriesFilled := 0
	indexEntriesFilledModN := 0
	onFillIndexEntry := func() {
		indexEntriesFilled++
		indexEntriesFilledModN++
		if indexEntriesFilledModN == everyNth {
			indexEntriesFilledModN = 0
			progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}
	progressFunc(0, numIndexEntries)
	q := newBestThreeCornersQuery(portalsData0, portalsData1, portalsData2, onFillIndexEntry)
	for _, p0 := range portalsData0 {
		for _, p1 := range portalsData1 {
			for _, p2 := range portalsData2 {
				q.findBestThreeCorner(p0, p1, p2)
			}
		}
	}
	progressFunc(numIndexEntries, numIndexEntries)

	var largestTC bestSolution
	var bestNumCornerChanges uint16
	var bestP0, bestP1, bestP2 portalData
	for _, p0 := range portalsData0 {
		for _, p1 := range portalsData1 {
			for _, p2 := range portalsData2 {
				solution := q.getIndex(p0.Index, p1.Index, p2.Index)
				numCornerChanges := q.getNumCornerChanges(p0.Index, p1.Index, p2.Index)
				if solution.Length > largestTC.Length || (solution.Length == largestTC.Length && numCornerChanges < bestNumCornerChanges) {
					largestTC = solution
					bestNumCornerChanges = numCornerChanges
					bestP0, bestP1, bestP2 = p0, p1, p2
				}
			}
		}
	}
	numPortals0 := portalIndex(len(portals0))
	numPortals1 := portalIndex(len(portals1))
	k0, k1, k2 := bestP0.Index, bestP1.Index, bestP2.Index
	result := append(make([]IndexedPortal, 0, largestTC.Length+3),
		IndexedPortal{0, portals0[k0]},
		IndexedPortal{1, portals1[k1]},
		IndexedPortal{2, portals2[k2]})
	for {
		sol := q.getIndex(k0, k1, k2)
		if sol.Length == 0 {
			break
		}
		if sol.Index < numPortals0 {
			result = append(result, IndexedPortal{0, portals0[sol.Index]})
			k0 = sol.Index
		} else {
			sol.Index = sol.Index - numPortals0
			if sol.Index < numPortals1 {
				result = append(result, IndexedPortal{1, portals1[sol.Index]})
				k1 = sol.Index
			} else {
				sol.Index = sol.Index - numPortals1
				result = append(result, IndexedPortal{2, portals2[sol.Index]})
				k2 = sol.Index
			}
		}
	}
	return result
}
