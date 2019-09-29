package lib

// LargestDoubleHerringbone - Find largest possible multilayer of portals to be made
func LargestDoubleHerringbone(portals []Portal, numWorkers int, progressFunc func(int, int)) (Portal, Portal, []Portal, []Portal) {
	if numWorkers == 1 {
		return LargestDoubleHerringboneST(portals, progressFunc)
	}
	return LargestDoubleHerringboneMT(portals, numWorkers, progressFunc)
}

// LargestDoubleHerringboneST - Find largest possible multilayer of portals to be made, using a single thread
func LargestDoubleHerringboneST(portals []Portal, progressFunc func(int, int)) (Portal, Portal, []Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{Index: portalIndex(i), LatLng: portal.LatLng})
	}
	portalList := make([]portalData, 0, len(portals))
	for _, p := range portalsData {
		portalList = append(portalList, p)
	}

	var largestCCW, largestCW []portalIndex
	var bestB0, bestB1 portalIndex
	resultCacheCCW := make([]portalIndex, 0, len(portals))
	resultCacheCW := make([]portalIndex, 0, len(portals))
	numPairs := len(portals) * (len(portals) - 1) / 2
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	numProcessedPairs := 0
	progressFunc(0, numPairs)
	q := newBestHerringboneQuery(portalsData)
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
				progressFunc(numProcessedPairs, numPairs)
			}
		}
	}
	progressFunc(numPairs, numPairs)
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
