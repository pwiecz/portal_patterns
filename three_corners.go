package main

import "fmt"

type bestThreeCornersQuery struct {
	portals0           []portalData
	numPortals0        portalIndex
	portals1           []portalData
	numPortals1        portalIndex
	portals2           []portalData
	index              [][][]bestSolution
	numCornerChanges   [][][]uint16
	onIndexEntryFilled func()
	filteredPortals0   []portalData
	filteredPortals1   []portalData
	filteredPortals2   []portalData
}

func newBestThreeCornersQuery(portals0, portals1, portals2 []portalData, onIndexEntryFilled func()) *bestThreeCornersQuery {
	fmt.Println("Lengths:", len(portals0), ",", len(portals1), ",", len(portals2))
	index := make([][][]bestSolution, 0, len(portals0))
	numCornerChanges := make([][][]uint16, 0, len(portals0))
	for i := 0; i < len(portals0); i++ {
		index = append(index, make([][]bestSolution, 0, len(portals1)))
		numCornerChanges = append(numCornerChanges, make([][]uint16, 0, len(portals1)))
		for j := 0; j < len(portals1); j++ {
			index[i] = append(index[i], make([]bestSolution, len(portals2)))
			numCornerChanges[i] = append(numCornerChanges[i], make([]uint16, len(portals2)))
			for k := 0; k < len(portals2); k++ {
				index[i][j][k].Length = invalidLength
			}
		}
	}
	return &bestThreeCornersQuery{
		portals0:           append(make([]portalData, 0, len(portals0)), portals0...),
		numPortals0:        portalIndex(len(portals0)),
		portals1:           append(make([]portalData, 0, len(portals1)), portals1...),
		numPortals1:        portalIndex(len(portals1)),
		portals2:           append(make([]portalData, 0, len(portals2)), portals2...),
		index:              index,
		numCornerChanges:   numCornerChanges,
		onIndexEntryFilled: onIndexEntryFilled,
		filteredPortals0:   make([]portalData, 0, len(portals0)),
		filteredPortals1:   make([]portalData, 0, len(portals1)),
		filteredPortals2:   make([]portalData, 0, len(portals2)),
	}
}

func (q *bestThreeCornersQuery) findBestThreeCorner(p0, p1, p2 portalData) {
	if q.index[p0.Index][p1.Index][p2.Index].Length != invalidLength {
		return
	}
	q.filteredPortals0 = portalsInsideTriangle(q.portals0, p0, p1, p2, q.filteredPortals0)
	q.filteredPortals1 = portalsInsideTriangle(q.portals1, p0, p1, p2, q.filteredPortals1)
	q.filteredPortals2 = portalsInsideTriangle(q.portals2, p0, p1, p2, q.filteredPortals2)
	q.findBestThreeCornerAux(p0, p1, p2, q.filteredPortals0, q.filteredPortals1, q.filteredPortals2)
}
func (q *bestThreeCornersQuery) findBestThreeCornerAux(p0, p1, p2 portalData, portals0, portals1, portals2 []portalData) (bestSolution, uint16) {
	fmt.Println("Local lengths:", len(portals0), ",", len(portals1), ",", len(portals2))
	localPortals0 := append(make([]portalData, 0, len(portals0)), portals0...)
	localPortals1 := append(make([]portalData, 0, len(portals1)), portals1...)
	localPortals2 := append(make([]portalData, 0, len(portals2)), portals2...)
	var bestTC bestSolution
	var bestNumCornerChanges uint16
	for i := 0; i < len(localPortals0); i++ {
		portal := localPortals0[i]
		candidate := q.index[portal.Index][p1.Index][p2.Index]
		numCornerChanges := q.numCornerChanges[portal.Index][p1.Index][p2.Index]
		if candidate.Length == invalidLength {
			filteredPortals0 := portalsInsideWedge(localPortals0, portal, p1, p2, q.filteredPortals0)
			filteredPortals1 := portalsInsideWedge(localPortals1, portal, p1, p2, q.filteredPortals1)
			filteredPortals2 := portalsInsideWedge(localPortals2, portal, p1, p2, q.filteredPortals2)
			candidate, numCornerChanges = q.findBestThreeCornerAux(portal, p1, p2, filteredPortals0, filteredPortals1, filteredPortals2)
			candidate.Length = candidate.Length + 1
			if candidate.Length > 0 && candidate.Index >= q.numPortals0 {
				numCornerChanges = numCornerChanges + 1
			}
		}
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && numCornerChanges < bestNumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index
			bestNumCornerChanges = numCornerChanges
		}
	}
	for i := 0; i < len(localPortals1); i++ {
		portal := localPortals1[i]
		candidate := q.index[p0.Index][portal.Index][p2.Index]
		numCornerChanges := q.numCornerChanges[p0.Index][portal.Index][p2.Index]
		if candidate.Length == invalidLength {
			filteredPortals0 := portalsInsideWedge(localPortals0, portal, p0, p2, q.filteredPortals0)
			filteredPortals1 := portalsInsideWedge(localPortals1, portal, p0, p2, q.filteredPortals1)
			filteredPortals2 := portalsInsideWedge(localPortals2, portal, p0, p2, q.filteredPortals2)
			candidate, numCornerChanges = q.findBestThreeCornerAux(p0, portal, p2, filteredPortals0, filteredPortals1, filteredPortals2)
			candidate.Length = candidate.Length + 1
			if candidate.Length > 0 && (candidate.Index < q.numPortals0 || candidate.Index >= q.numPortals0+q.numPortals1) {
				numCornerChanges = numCornerChanges + 1
			}
		}
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && numCornerChanges < bestNumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index + q.numPortals0
			bestNumCornerChanges = numCornerChanges
		}
	}
	for i := 0; i < len(localPortals2); i++ {
		portal := localPortals2[i]
		fmt.Println(p0.Index, ",", p1.Index, ",", portal.Index)
		candidate := q.index[p0.Index][p1.Index][portal.Index]
		numCornerChanges := q.numCornerChanges[p0.Index][p1.Index][portal.Index]
		if candidate.Length == invalidLength {
			filteredPortals0 := portalsInsideWedge(localPortals0, portal, p0, p1, q.filteredPortals0)
			filteredPortals1 := portalsInsideWedge(localPortals1, portal, p0, p1, q.filteredPortals1)
			filteredPortals2 := portalsInsideWedge(localPortals2, portal, p0, p1, q.filteredPortals2)
			candidate, numCornerChanges = q.findBestThreeCornerAux(p0, p1, portal, filteredPortals0, filteredPortals1, filteredPortals2)
			candidate.Length = candidate.Length + 1
			if candidate.Length > 0 && candidate.Index < q.numPortals0+q.numPortals1 {
				numCornerChanges = numCornerChanges + 1
			}
		}
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && numCornerChanges < bestNumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index + q.numPortals0 + q.numPortals1
			bestNumCornerChanges = numCornerChanges
		}
	}
	q.index[p0.Index][p1.Index][p2.Index] = bestTC
	q.onIndexEntryFilled()
	return bestTC, bestNumCornerChanges
}

// LargestThreeCorner - Find best way to connect three groups of portals
func LargestThreeCorner(portals0, portals1, portals2 []Portal) []indexedPortal {
	portalsData0 := portalsToPortalData(portals0)
	portalsData1 := portalsToPortalData(portals1)
	portalsData2 := portalsToPortalData(portals2)

	numIndexEntries := len(portals0) * len(portals1) * len(portals2)
	everyNth := numIndexEntries / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	indexEntriesFilled := 0
	onFillIndexEntry := func() {
		indexEntriesFilled++
		if indexEntriesFilled%everyNth == 0 {
			printProgressBar(indexEntriesFilled, numIndexEntries)
		}
	}
	printProgressBar(0, numIndexEntries)
	q := newBestThreeCornersQuery(portalsData0, portalsData1, portalsData2, onFillIndexEntry)
	for _, p0 := range portalsData0 {
		for _, p1 := range portalsData1 {
			for _, p2 := range portalsData2 {
				q.findBestThreeCorner(p0, p1, p2)
			}
		}
	}
	printProgressBar(numIndexEntries, numIndexEntries)
	fmt.Println("")

	var largestTC bestSolution
	var bestNumCornerChanges uint16
	var bestP0, bestP1, bestP2 portalData
	for _, p0 := range portalsData0 {
		for _, p1 := range portalsData1 {
			for _, p2 := range portalsData2 {
				solution := q.index[p0.Index][p1.Index][p2.Index]
				numCornerChanges := q.numCornerChanges[p0.Index][p1.Index][p2.Index]
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
	result := append(make([]indexedPortal, 0, largestTC.Length+3),
		indexedPortal{0, portals0[k0]},
		indexedPortal{1, portals1[k1]},
		indexedPortal{2, portals2[k2]})
	for {
		sol := q.index[k0][k1][k2]
		if sol.Length == 0 {
			break
		}
		if sol.Index < numPortals0 {
			result = append(result, indexedPortal{0, portals0[sol.Index]})
			k0 = sol.Index
		} else {
			sol.Index = sol.Index - numPortals0
			if sol.Index < numPortals1 {
				result = append(result, indexedPortal{1, portals1[sol.Index]})
				k1 = sol.Index
			} else {
				sol.Index = sol.Index - numPortals1
				result = append(result, indexedPortal{2, portals2[sol.Index]})
				k2 = sol.Index
			}
		}
	}
	return result
}
