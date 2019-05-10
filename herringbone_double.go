package main

import "fmt"

// LargestDoubleHerringbone - Find largest possible multilayer of portals to be made
func LargestDoubleHerringbone(portals []Portal) (Portal, Portal, []Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{Index: i, LatLng: portal.LatLng})
	}
	portalList := make([]portalData, 0, len(portals))
	for _, p := range portalsData {
		portalList = append(portalList, p)
	}

	var largestCCW, largestCW []int
	var bestB0, bestB1 int
	nodesCache := make([]node, 0, len(portals))
	resultCacheCCW := make([]int, 0, len(portals))
	resultCacheCW := make([]int, 0, len(portals))
	for i, b0 := range portalsData {
		printProgressBar(i, len(portals))
		for j := i + 1; j < len(portalsData); j++ {
			b1 := portalsData[j]
			bestCCW := findBestHerringbone(b0, b1, portalsData, nodesCache, resultCacheCCW)
			bestCW := findBestHerringbone(b1, b0, portalsData, nodesCache, resultCacheCW)
			if len(bestCCW)+len(bestCW) > len(largestCCW)+len(largestCW) {
				largestCCW = append(largestCCW[:0], bestCCW...)
				largestCW = append(largestCW[:0], bestCW...)
				bestB0, bestB1 = b0.Index, b1.Index
			}
		}
	}
	fmt.Println("")
	resultCCW := make([]Portal, 0, len(largestCCW))
	for _, portalIx := range largestCCW {
		resultCCW = append(resultCCW, portals[portalIx])
	}
	resultCW := make([]Portal, 0, len(largestCW))
	for _, portalIx := range largestCW {
		resultCW = append(resultCW, portals[portalIx])
	}

	return portals[bestB0], portals[bestB1], resultCCW, resultCW
}
