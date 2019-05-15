package main

import "fmt"
import "math"

type bestHomogeneousQuery struct {
	portals            []portalData
	index              [][][]bestSolution
	onFilledIndexEntry func()
	portalsInTriangle  []portalData
}

func newBestHomogeneousQuery(portals []portalData, onFilledIndexEntry func()) *bestHomogeneousQuery{
	index := make([][][]bestSolution, 0, len(portals))
	for i := 0; i < len(portals); i++ {
		index = append(index, make([][]bestSolution, 0, len(portals)))
		for j := 0; j < len(portals); j++ {
			index[i] = append(index[i], make([]bestSolution, len(portals)))
			for k := 0; k < len(portals); k++ {
				index[i][j][k].Length = invalidLength
			}
		}
	}
	return &bestHomogeneousQuery{
		portals:            portals,
		index:              index,
		onFilledIndexEntry: onFilledIndexEntry,
	}
}

func (q *bestHomogeneousQuery) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.index[p0.Index][p1.Index][p2.Index].Length != invalidLength {
		return
	}
	triangle := newTriangleQuery(p1.LatLng, p0.LatLng, p2.LatLng)
	q.portalsInTriangle = q.portalsInTriangle[:0]
	for _, p := range q.portals {
		if p.Index != p0.Index && p.Index != p1.Index && p.Index != p2.Index &&
			triangle.ContainsPoint(p.LatLng) {
			q.portalsInTriangle = append(q.portalsInTriangle, p)
		}
	}
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle)
}

func (q *bestHomogeneousQuery) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
	localCandidates := append(make([]portalData, 0, len(candidates)), candidates...)
	var bestHomogeneous bestSolution
	for _, portal := range localCandidates {
		minDepth := uint16(math.MaxUint16)
		{
			candidate0 := q.index[portal.Index][p1.Index][p2.Index]
			if candidate0.Length == invalidLength {
				candidatesInWedge := portalsInsideWedge(candidates, portal, p1, p2)
				candidate0 = q.findBestHomogeneousAux(portal, p1, p2, candidatesInWedge)
			}
			if candidate0.Length < minDepth {
				minDepth = candidate0.Length
			}
		}
		{
			candidate1 := q.index[portal.Index][p0.Index][p2.Index]
			if candidate1.Length == invalidLength {
				candidatesInWedge := portalsInsideWedge(candidates, portal, p0, p2)
				candidate1 = q.findBestHomogeneousAux(portal, p0, p2, candidatesInWedge)
			}
			if candidate1.Length < minDepth {
				minDepth = candidate1.Length
			}
		}
		{
			candidate2 := q.index[portal.Index][p0.Index][p1.Index]
			if candidate2.Length == invalidLength {
				candidatesInWedge := portalsInsideWedge(candidates, portal, p0, p1)
				candidate2 = q.findBestHomogeneousAux(portal, p0, p1, candidatesInWedge)
			}
			if candidate2.Length < minDepth {
				minDepth = candidate2.Length
			}
		}
		if minDepth != uint16(math.MaxUint16) && minDepth+1 > bestHomogeneous.Length {
			bestHomogeneous.Index = portal.Index
			bestHomogeneous.Length = minDepth + 1
		}
	}
	if q.index[p0.Index][p1.Index][p2.Index].Length == invalidLength {
		q.onFilledIndexEntry()
	}
	q.index[p0.Index][p1.Index][p2.Index] = bestHomogeneous
	q.index[p0.Index][p2.Index][p1.Index] = bestHomogeneous
	q.index[p1.Index][p0.Index][p2.Index] = bestHomogeneous
	q.index[p1.Index][p2.Index][p0.Index] = bestHomogeneous
	q.index[p2.Index][p0.Index][p1.Index] = bestHomogeneous
	q.index[p2.Index][p1.Index][p0.Index] = bestHomogeneous
	return bestHomogeneous
}

// DeepestHomogeneous - Find deepest homogeneous field that can be made out of portals
func DeepestHomogeneous(portals []Portal) ([]Portal, uint16) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

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
	q := newBestHomogeneousQuery(portalsData, onFilledIndexEntry)
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				q.findBestHomogeneous(p0, p1, p2)
			}
		}
	}
	printProgressBar(numIndexEntries, numIndexEntries)
	fmt.Println("")

	var bestDepth uint16
	var bestP0, bestP1, bestP2 portalData
	var bestArea float64
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				candidate := q.index[p0.Index][p1.Index][p2.Index]
				if candidate.Length > bestDepth || (candidate.Length == bestDepth && triangleArea(p0, p1, p2) < bestArea) {
					bestP0, bestP1, bestP2 = p0, p1, p2
					bestDepth = candidate.Length
					bestArea = triangleArea(p0, p1, p2)
				}
			}
		}
	}

	resultIndices := []uint16{bestP0.Index, bestP1.Index, bestP2.Index}
	resultIndices = appendHomogeneousResult(bestP0.Index, bestP1.Index, bestP2.Index, bestDepth, resultIndices, q.index)
	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, bestDepth
}

func appendHomogeneousResult(p0, p1, p2 uint16, maxDepth uint16, result []uint16, index [][][]bestSolution) []uint16 {
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

func appendHomogeneousPolylines(p0, p1, p2 Portal, maxDepth uint16, result []string, portals []Portal) ([]string, []Portal) {
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
