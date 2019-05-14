package main

import "fmt"

type bestTCSolution struct {
	Index            uint16
	Length           uint16
	NumCornerChanges uint16
}

func findBestThreeCorner(p0, p1, p2 portalData, portals0, portals1, portals2 []portalData, index [][][]bestTCSolution, numP0, numP1 uint16, onIndexEntryFilled func()) bestTCSolution {
	if index[p0.Index][p1.Index][p2.Index].Length != invalidLength {
		return index[p0.Index][p1.Index][p2.Index]
	}
	triangle := newTriangleQuery(p0.LatLng, p1.LatLng, p2.LatLng)
	filteredPortals0 := keepOnlyContainedPortals(portals0, triangle)
	filteredPortals1 := keepOnlyContainedPortals(portals1, triangle)
	filteredPortals2 := keepOnlyContainedPortals(portals2, triangle)
	localPortals0 := append(make([]portalData, 0, len(filteredPortals0)), filteredPortals0...)
	localPortals1 := append(make([]portalData, 0, len(filteredPortals1)), filteredPortals1...)
	localPortals2 := append(make([]portalData, 0, len(filteredPortals2)), filteredPortals2...)
	var bestTC bestTCSolution
	for i := 0; i < len(localPortals0); i++ {
		portal := localPortals0[i]
		candidate := findBestThreeCorner(portal, p1, p2, filteredPortals0, filteredPortals1, filteredPortals2, index, numP0, numP1, onIndexEntryFilled)
		candidate.Length = candidate.Length + 1
		if candidate.Length > 0 && candidate.Index >= numP0 {
			candidate.NumCornerChanges = candidate.NumCornerChanges + 1
		}
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && candidate.NumCornerChanges < bestTC.NumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index
			bestTC.NumCornerChanges = candidate.NumCornerChanges
		}
	}
	for i := 0; i < len(localPortals1); i++ {
		portal := localPortals1[i]
		candidate := findBestThreeCorner(p0, portal, p2, filteredPortals0, filteredPortals1, filteredPortals2, index, numP0, numP1, onIndexEntryFilled)
		candidate.Length = candidate.Length + 1
		if candidate.Length > 0 && (candidate.Index < numP0 || candidate.Index >= numP0+numP1) {
			candidate.NumCornerChanges = candidate.NumCornerChanges + 1
		}
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && candidate.NumCornerChanges < bestTC.NumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index + numP0
			bestTC.NumCornerChanges = candidate.NumCornerChanges
		}
	}
	for i := 0; i < len(localPortals2); i++ {
		portal := localPortals2[i]
		candidate := findBestThreeCorner(p0, p1, portal, filteredPortals0, filteredPortals1, filteredPortals2, index, numP0, numP1, onIndexEntryFilled)
		candidate.Length = candidate.Length + 1
		if candidate.Length > 0 && candidate.Index < numP0+numP1 {
			candidate.NumCornerChanges = candidate.NumCornerChanges + 1
		}
		if candidate.Length > bestTC.Length || (candidate.Length == bestTC.Length && candidate.NumCornerChanges < bestTC.NumCornerChanges) {
			bestTC.Length = candidate.Length
			bestTC.Index = portal.Index + numP0 + numP1
			bestTC.NumCornerChanges = candidate.NumCornerChanges
		}
	}
	index[p0.Index][p1.Index][p2.Index] = bestTC
	onIndexEntryFilled()
	return bestTC
}

// LargestThreeCorner - Find best way to connect three groups of portals
func LargestThreeCorner(portals0, portals1, portals2 []Portal) []indexedPortal {
	portalsData0 := portalsToPortalData(portals0)
	portalsData1 := portalsToPortalData(portals1)
	portalsData2 := portalsToPortalData(portals2)
	localPortalsData0 := append(make([]portalData, 0, len(portalsData0)), portalsData0...)
	localPortalsData1 := append(make([]portalData, 0, len(portalsData1)), portalsData1...)
	localPortalsData2 := append(make([]portalData, 0, len(portalsData2)), portalsData2...)
	numPortals0 := uint16(len(portals0))
	numPortals1 := uint16(len(portals1))
	numPortals2 := uint16(len(portals2))
	index := make([][][]bestTCSolution, 0, numPortals0)
	for i := 0; i < len(portals0); i++ {
		index = append(index, make([][]bestTCSolution, 0, numPortals1))
		for j := 0; j < len(portals1); j++ {
			index[i] = append(index[i], make([]bestTCSolution, numPortals2))
			for k := 0; k < len(portals2); k++ {
				index[i][j][k].Length = invalidLength
			}
		}
	}

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
	for _, p0 := range localPortalsData0 {
		for _, p1 := range localPortalsData1 {
			for _, p2 := range localPortalsData2 {
				if index[p0.Index][p1.Index][p2.Index].Length != invalidLength {
					continue
				}
				findBestThreeCorner(p0, p1, p2, portalsData0, portalsData1, portalsData2, index, numPortals0, numPortals1, onFillIndexEntry)
			}
		}
	}
	printProgressBar(numIndexEntries, numIndexEntries)
	fmt.Println("")

	var largestTC bestTCSolution
	var bestP0, bestP1, bestP2 portalData
	for _, p0 := range localPortalsData0 {
		for _, p1 := range localPortalsData1 {
			for _, p2 := range localPortalsData2 {
				solution := index[p0.Index][p1.Index][p2.Index]
				if solution.Length > largestTC.Length || (solution.Length == largestTC.Length && solution.NumCornerChanges < largestTC.NumCornerChanges) {
					largestTC = solution
					bestP0, bestP1, bestP2 = p0, p1, p2
				}
			}
		}
	}
	k0, k1, k2 := bestP0.Index, bestP1.Index, bestP2.Index
	result := append(make([]indexedPortal, 0, largestTC.Length+3),
		indexedPortal{0, portals0[k0]},
		indexedPortal{1, portals1[k1]},
		indexedPortal{2, portals2[k2]})
	for {
		sol := index[k0][k1][k2]
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
