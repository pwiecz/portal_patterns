package main

import "fmt"

type bestCobwebQuery struct {
	portals            []portalData
	index              []bestSolution
	numPortals         int64
	numPortalsSq       int64
	onFilledIndexEntry func()
	filteredPortals    []portalData
}

func newBestCobwebQuery(portals []portalData, onFilledIndexEntry func()) *bestCobwebQuery {
	numPortals := int64(len(portals))
	index := make([]bestSolution, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i].Length = invalidLength
	}
	return &bestCobwebQuery{
		portals:            portals,
		numPortals:         numPortals,
		numPortalsSq:       numPortals * numPortals,
		index:              index,
		onFilledIndexEntry: onFilledIndexEntry,
		filteredPortals:    make([]portalData, 0, len(portals)),
	}
}
func (q *bestCobwebQuery) getIndex(i, j, k portalIndex) bestSolution {
	return q.index[int64(i)*q.numPortalsSq+int64(j)*q.numPortals+int64(k)]
}
func (q *bestCobwebQuery) setIndex(i, j, k portalIndex, s bestSolution) {
	q.index[int64(i)*q.numPortalsSq+int64(j)*q.numPortals+int64(k)] = s
}
func (q *bestCobwebQuery) findBestCobweb(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index).Length != invalidLength {
		return
	}
	q.filteredPortals = portalsInsideTriangle(q.portals, p0, p1, p2, q.filteredPortals)
	q.findBestCobwebAux(p0, p1, p2, q.filteredPortals)
}

func (q *bestCobwebQuery) findBestCobwebAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
	localCandidates := append(make([]portalData, 0, len(candidates)), candidates...)
	var bestCobweb bestSolution
	for _, portal := range localCandidates {
		candidate := q.getIndex(p1.Index, p2.Index, portal.Index)
		if candidate.Length == invalidLength {
			candidatesInWedge := portalsInsideWedge(localCandidates, portal, p1, p2, q.filteredPortals)
			candidate = q.findBestCobwebAux(p1, p2, portal, candidatesInWedge)
		}
		if candidate.Length+1 > bestCobweb.Length {
			bestCobweb.Length = candidate.Length + 1
			bestCobweb.Index = portal.Index
		}
	}
	q.onFilledIndexEntry()

	q.setIndex(p0.Index, p1.Index, p2.Index, bestCobweb)
	return bestCobweb
}

// LargestCobweb - Find largest possible cobweb of portals to be made
func LargestCobweb(portals []Portal) []Portal {
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
	onFilledIndexEntry := func() {
		indexEntriesFilled++
		if indexEntriesFilled%everyNth == 0 {
			printProgressBar(indexEntriesFilled, numIndexEntries)
		}
	}
	printProgressBar(0, numIndexEntries)
	q := newBestCobwebQuery(portalsData, onFilledIndexEntry)
	for i, p0 := range portalsData {
		for j, p1 := range portalsData {
			if i == j {
				continue
			}
			for k, p2 := range portalsData {
				if i == k || j == k {
					continue
				}
				q.findBestCobweb(p0, p1, p2)
			}
		}
	}
	printProgressBar(numIndexEntries, numIndexEntries)
	fmt.Println("")

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
