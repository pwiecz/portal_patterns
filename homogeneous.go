package main

import "fmt"
import "math"

func findBestHomogenous(p0, p1, p2 portalData, candidates []portalData, index [][][]bestSolution, onFilledIndexEntry func()) bestSolution {
	var bestHomogeneous bestSolution
	for _, portal := range candidates {
		minDepth := math.MaxInt32
		{
			candidate0 := index[portal.Index][p1.Index][p2.Index]
			if candidate0.Length < 0 {
				candidatesInWedge := portalsInsideWedge(candidates, portal, p1, p2)
				candidate0 = findBestHomogenous(portal, p1, p2, candidatesInWedge, index, onFilledIndexEntry)
			}
			if candidate0.Length < minDepth {
				minDepth = candidate0.Length
			}
		}
		{
			candidate1 := index[portal.Index][p0.Index][p2.Index]
			if candidate1.Length < 0 {
				candidatesInWedge := portalsInsideWedge(candidates, portal, p0, p2)
				candidate1 = findBestHomogenous(portal, p0, p2, candidatesInWedge, index, onFilledIndexEntry)
			}
			if candidate1.Length < minDepth {
				minDepth = candidate1.Length
			}
		}
		{
			candidate2 := index[portal.Index][p0.Index][p1.Index]
			if candidate2.Length < 0 {
				candidatesInWedge := portalsInsideWedge(candidates, portal, p0, p1)
				candidate2 = findBestHomogenous(portal, p0, p1, candidatesInWedge, index, onFilledIndexEntry)
			}
			if candidate2.Length < minDepth {
				minDepth = candidate2.Length
			}
		}
		if minDepth != math.MaxInt32 && minDepth+1 > bestHomogeneous.Length {
			bestHomogeneous.Index = portal.Index
			bestHomogeneous.Length = minDepth + 1
		}
	}
	if index[p0.Index][p1.Index][p2.Index].Length < 0 {
		onFilledIndexEntry()
	}
	index[p0.Index][p1.Index][p2.Index] = bestHomogeneous
	index[p0.Index][p2.Index][p1.Index] = bestHomogeneous
	index[p1.Index][p0.Index][p2.Index] = bestHomogeneous
	index[p1.Index][p2.Index][p0.Index] = bestHomogeneous
	index[p2.Index][p0.Index][p1.Index] = bestHomogeneous
	index[p2.Index][p1.Index][p0.Index] = bestHomogeneous
	return bestHomogeneous
}

// DeepestHomogeneous - Find deepest homogeneous field that can be made out of portals
func DeepestHomogeneous(portals []Portal) ([]Portal, int) {
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

	numIndexEntries := len(portals) * (len(portals) - 1) * (len(portals) - 2) / 6
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
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
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
				findBestHomogenous(p0, p1, p2, portalsInTriangle, index, onFilledIndexEntry)
			}
		}
	}
	printProgressBar(numIndexEntries, numIndexEntries)
	fmt.Println("")

	bestDepth := -1
	var bestP0, bestP1, bestP2 portalData
	var bestArea float64
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				candidate := index[p0.Index][p1.Index][p2.Index]
				if candidate.Length > bestDepth || (candidate.Length == bestDepth && triangleArea(p0, p1, p2) < bestArea) {
					bestP0, bestP1, bestP2 = p0, p1, p2
					bestDepth = candidate.Length
					bestArea = triangleArea(p0, p1, p2)
				}
			}
		}
	}

	resultIndices := []int{bestP0.Index, bestP1.Index, bestP2.Index}
	resultIndices = appendHomogeneousResult(bestP0.Index, bestP1.Index, bestP2.Index, bestDepth, resultIndices, index)
	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, bestDepth
}

func appendHomogeneousResult(p0, p1, p2 int, maxDepth int, result []int, index [][][]bestSolution) []int {
	if maxDepth == 0 {
		return result
	}
	bestP := index[p0][p1][p2].Index
	result = append(result, bestP)
	result = appendHomogeneousResult(bestP, p1, p2, maxDepth-1, result, index)
	result = appendHomogeneousResult(p0, bestP, p2, maxDepth-1, result, index)
	result = appendHomogeneousResult(p0, p1, bestP, maxDepth-1, result, index)
	return result
}

func appendHomogeneousPolylines(p0, p1, p2 Portal, maxDepth int, result []string, portals []Portal) ([]string, []Portal) {
	if maxDepth == 0 {
		return result, portals
	}
	portal := portals[0]
	result = append(result,
		polylineFromPortalList([]Portal{p0, portal}),
		polylineFromPortalList([]Portal{p1, portal}),
		polylineFromPortalList([]Portal{p2, portal}))
	result, portals = appendHomogeneousPolylines(portal, p1, p2, maxDepth-1, result, portals[1:])
	result, portals = appendHomogeneousPolylines(p0, portal, p2, maxDepth-1, result, portals)
	result, portals = appendHomogeneousPolylines(p0, p1, portal, maxDepth-1, result, portals)
	return result, portals
}
