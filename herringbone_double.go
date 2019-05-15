package main

import "fmt"

// LargestDoubleHerringbone - Find largest possible multilayer of portals to be made
func LargestDoubleHerringbone(portals []Portal) (Portal, Portal, []Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{Index: uint16(i), LatLng: portal.LatLng})
	}
	portalList := make([]portalData, 0, len(portals))
	for _, p := range portalsData {
		portalList = append(portalList, p)
	}

	var largestCCW, largestCW []uint16
	var bestB0, bestB1 uint16
	resultCacheCCW := make([]uint16, 0, len(portals))
	resultCacheCW := make([]uint16, 0, len(portals))
	numPairs := len(portals) * (len(portals) - 1) / 2
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	numProcessedPairs := 0
	printProgressBar(0, numPairs)
	q := newBestHerringBoneQuery(portalsData)
	for i, b0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			b1 := portalsData[j]
			bestCCW := q.findBestHerringbone(b0, b1, resultCacheCCW)
			bestCW := q.findBestHerringbone(b1, b0, resultCacheCW)
			if len(bestCCW)+len(bestCW) > len(largestCCW)+len(largestCW) {
				largestCCW = append(largestCCW[:0], bestCCW...)
				largestCW = append(largestCW[:0], bestCW...)
				bestB0, bestB1 = b0.Index, b1.Index
			}
			numProcessedPairs++
			if numProcessedPairs%everyNth == 0 {
				printProgressBar(numProcessedPairs, numPairs)
			}
		}
	}
	printProgressBar(numPairs, numPairs)
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
