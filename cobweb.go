package main

import "fmt"

func findBestCobWeb(p0, p1, p2 portalData, candidates []portalData, index [][][]bestSolution, onFilledIndexEntry func()) bestSolution {
	localCandidates := append(make([]portalData, 0, len(candidates)), candidates...)
	var bestCobWeb bestSolution
	for _, portal := range localCandidates {
		candidate := index[p1.Index][p2.Index][portal.Index]
		if candidate.Length < 0 {
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
			candidate = findBestCobWeb(p1, p2, portal, candidatesInWedge, index, onFilledIndexEntry)
		}
		if candidate.Length+1 > bestCobWeb.Length {
			bestCobWeb.Length = candidate.Length + 1
			bestCobWeb.Index = portal.Index
		}
	}
	if index[p0.Index][p1.Index][p2.Index].Length < 0 {
		onFilledIndexEntry()
	}

	index[p0.Index][p1.Index][p2.Index] = bestCobWeb
	return bestCobWeb
}

// LargestCobWeb - Find largest possible CobWeb of portals to be made
func LargestCobWeb(portals []Portal) []Portal {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)
	index := make([][][]bestSolution, 0, len(portals))
	for i := 0; i < len(portals); i++ {
		index = append(index, make([][]bestSolution, 0, len(portals)))
		for j := 0; j < len(portals); j++ {
			index[i] = append(index[i], make([]bestSolution, len(portals)))
			for k := 0; k < len(portals); k++ {
				index[i][j][k].Length = -1
			}
		}
	}

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
	var portalsInTriangle []portalData
	for i, p0 := range portalsData {
		for j, p1 := range portalsData {
			if i == j {
				continue
			}
			for k, p2 := range portalsData {
				if i == k || j == k {
					continue
				}
				if index[p0.Index][p1.Index][p2.Index].Length >= 0 {
					continue
				}
				triangle := newTriangleQuery(p1.LatLng, p0.LatLng, p2.LatLng)
				portalsInTriangle = portalsInTriangle[:0]
				for _, p := range portalsData {
					if p.Index != p0.Index && p.Index != p1.Index && p.Index != p2.Index &&
						triangle.ContainsPoint(p.LatLng) {
						portalsInTriangle = append(portalsInTriangle, p)
					}
				}
				findBestCobWeb(p0, p1, p2, portalsInTriangle, index, onFilledIndexEntry)
			}
		}
	}
	printProgressBar(numIndexEntries, numIndexEntries)
	fmt.Println("")

	var bestP0, bestP1, bestP2 portalData
	bestLength := 0
	for i, p0 := range portalsData {
		for j, p1 := range portalsData {
			if i == j {
				continue
			}
			for k, p2 := range portalsData {
				if i == k || j == k {
					continue
				}
				candidate := index[p0.Index][p1.Index][p2.Index]
				if candidate.Length+3 > bestLength {
					bestP0, bestP1, bestP2 = p0, p1, p2
					bestLength = candidate.Length + 3
				}
			}
		}
	}

	largestCobweb := append(make([]int, 0, bestLength), bestP0.Index, bestP1.Index, bestP2.Index)
	k0, k1, k2 := bestP0.Index, bestP1.Index, bestP2.Index
	for {
		sol := index[k0][k1][k2]
		if sol.Length <= 0 {
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
