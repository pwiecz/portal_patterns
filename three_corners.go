package main

import "fmt"

type bestTCSolution struct {
	Index            int
	Length           int
	NumCornerChanges int
}

func findBestThreeCorner(p0, p1, p2 portalData, portals0, portals1, portals2 []portalData, index [][][]bestTCSolution, numP0, numP1 int, onIndexEntryFilled func()) bestTCSolution {
	if index[p0.Index][p1.Index][p2.Index].Length >= 0 {
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

	index := make([][][]bestTCSolution, 0, len(portals0))
	for i := 0; i < len(portals0); i++ {
		index = append(index, make([][]bestTCSolution, 0, len(portals1)))
		for j := 0; j < len(portals1); j++ {
			index[i] = append(index[i], make([]bestTCSolution, len(portals2)))
			for k := 0; k < len(portals2); k++ {
				index[i][j][k].Length = -1
			}
		}
	}

	numIndexEntries := len(portals0) * len(portals1) * len(portals2)
	indexEntriesFilled := 0
	onFillIndexEntry := func() {
		indexEntriesFilled++
		everyNth := numIndexEntries / 1000
		if everyNth < 2 {
			everyNth = 2
		}
		if indexEntriesFilled%everyNth == 1 {
			printProgressBar(indexEntriesFilled, numIndexEntries)
		}
	}
	for _, p0 := range portalsData0 {
		for _, p1 := range portalsData1 {
			for _, p2 := range portalsData2 {
				if index[p0.Index][p1.Index][p2.Index].Length >= 0 {
					continue
				}
				findBestThreeCorner(p0, p1, p2, portalsData0, portalsData1, portalsData2, index, len(portals0), len(portals1), onFillIndexEntry)
			}
		}
	}
	printProgressBar(numIndexEntries, numIndexEntries)
	fmt.Println("")

	var largestTC bestTCSolution
	var bestP0, bestP1, bestP2 portalData
	for _, p0 := range portalsData0 {
		for _, p1 := range portalsData1 {
			for _, p2 := range portalsData2 {
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
		if sol.Length <= 0 {
			break
		}
		if sol.Index < len(portals0) {
			result = append(result, indexedPortal{0, portals0[sol.Index]})
			k0 = sol.Index
		} else {
			sol.Index = sol.Index - len(portals0)
			if sol.Index < len(portals1) {
				result = append(result, indexedPortal{1, portals1[sol.Index]})
				k1 = sol.Index
			} else {
				sol.Index = sol.Index - len(portals1)
				result = append(result, indexedPortal{2, portals2[sol.Index]})
				k2 = sol.Index
			}
		}
	}
	return result
}
