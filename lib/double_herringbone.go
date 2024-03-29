package lib

// LargestDoubleHerringbone - Find largest possible multilayer of portals to be made
func LargestDoubleHerringbone(portals []Portal, fixedBaseIndices []int, numWorkers int, progressFunc func(int, int)) (Portal, Portal, []Portal, []Portal) {
	if numWorkers == 1 {
		return LargestDoubleHerringboneST(portals, fixedBaseIndices, progressFunc)
	}
	return LargestDoubleHerringboneMT(portals, fixedBaseIndices, numWorkers, progressFunc)
}

// LargestDoubleHerringboneST - Find largest possible multilayer of portals to be made, using a single thread
func LargestDoubleHerringboneST(portals []Portal, fixedBaseIndices []int, progressFunc func(int, int)) (Portal, Portal, []Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

	var largestCCW, largestCW []portalIndex
	var bestB0, bestB1 portalIndex
	resultCacheCCW := make([]portalIndex, 0, len(portals))
	resultCacheCW := make([]portalIndex, 0, len(portals))
	numPairs := len(portals) * (len(portals) - 1) / 2
	if len(fixedBaseIndices) == 1 {
		numPairs = len(portals) - 1
	} else if len(fixedBaseIndices) == 2 {
		numPairs = 1
	}
	everyNth := numPairs / 1000
	if everyNth < 1 {
		everyNth = 1
	}
	numProcessedPairs := 0
	progressFunc(0, numPairs)
	q := newBestHerringboneQuery(portalsData)
	for i, b0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			b1 := portalsData[j]
			if !hasAllElementsInThePair(fixedBaseIndices, i, j) {
				continue
			}
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

func DoubleHerringbonePolyline(b0, b1 Portal, result0, result1 []Portal) []Portal {
	portalList := []Portal{b0, b1}
	atIndex := 0
	for _, portal := range result0 {
		portalList = append(portalList, portal, portalList[atIndex])
		atIndex = 1 - atIndex
	}
	for _, portal := range result1 {
		portalList = append(portalList, portal, portalList[atIndex])
		atIndex = 1 - atIndex
	}
	return portalList
}
func DoubleHerringboneDrawToolsString(b0, b1 Portal, result0, result1 []Portal) string {
	return "[\n" + PolylineFromPortalList(DoubleHerringbonePolyline(b0, b1, result0, result1)) + "\n]"
}
