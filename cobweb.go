package main

import "fmt"

type bestCobwebQuery struct {
	portals            []portalData
	index              [][][]bestSolution
	onFilledIndexEntry func()
}

func newBestCobwebQuery(portals []portalData, onFilledIndexEntry func()) *bestCobwebQuery {
	numPortals := len(portals)
	index := make([][][]bestSolution, 0, numPortals)
	for i := 0; i < numPortals; i++ {
		index = append(index, make([][]bestSolution, 0, numPortals))
		for j := 0; j < numPortals; j++ {
			index[i] = append(index[i], make([]bestSolution, numPortals))
			for k := 0; k < numPortals; k++ {
				index[i][j][k].Length = invalidLength
			}
		}
	}
	return &bestCobwebQuery{
		portals:            append(make([]portalData, 0, len(portals)), portals...),
		index:              index,
		onFilledIndexEntry: onFilledIndexEntry,
	}
}

func (q *bestCobwebQuery) findBestCobweb(p0, p1, p2 portalData) {
	if q.index[p0.Index][p1.Index][p2.Index].Length != invalidLength {
		return
	}
	filteredPortals := portalsInsideTriangle(q.portals, p0, p1, p2)
	q.findBestCobwebAux(p0, p1, p2, filteredPortals)
}

func (q *bestCobwebQuery) findBestCobwebAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
	localCandidates := append(make([]portalData, 0, len(candidates)), candidates...)
	var bestCobweb bestSolution
	for _, portal := range localCandidates {
		candidate := q.index[p1.Index][p2.Index][portal.Index]
		if candidate.Length == invalidLength {
			wedge := newTriangleWedgeQuery(portal.LatLng, p1.LatLng, p2.LatLng)
			candidatesInWedge := candidates
			for i := 0; i < len(candidatesInWedge); {
				cand := candidatesInWedge[i]
				if cand.Index != portal.Index && wedge.ContainsPoint(cand.LatLng) {
					i++
				} else {
					candidatesInWedge[i], candidatesInWedge[len(candidatesInWedge)-1] = candidatesInWedge[len(candidatesInWedge)-1], cand
					candidatesInWedge = candidatesInWedge[:len(candidatesInWedge)-1]
				}
			}
			candidate = q.findBestCobwebAux(p1, p2, portal, candidatesInWedge)
		}
		if candidate.Length+1 > bestCobweb.Length {
			bestCobweb.Length = candidate.Length + 1
			bestCobweb.Index = portal.Index
		}
	}
	if q.index[p0.Index][p1.Index][p2.Index].Length == invalidLength {
		q.onFilledIndexEntry()
	}

	q.index[p0.Index][p1.Index][p2.Index] = bestCobweb
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
				candidate := q.index[p0.Index][p1.Index][p2.Index]
				if candidate.Length+3 > bestLength {
					bestP0, bestP1, bestP2 = p0, p1, p2
					bestLength = candidate.Length + 3
				}
			}
		}
	}

	largestCobweb := append(make([]uint16, 0, bestLength), bestP0.Index, bestP1.Index, bestP2.Index)
	k0, k1, k2 := bestP0.Index, bestP1.Index, bestP2.Index
	for {
		sol := q.index[k0][k1][k2]
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
