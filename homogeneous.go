package main

import "fmt"
import "math"

type topLevelTriangleScorer interface {
	scoreTriangle(a, b, c portalData) float32
}

type bestHomogeneousQuery struct {
	portals            []portalData
	index              [][][]bestSolution
	onFilledIndexEntry func()
	portalsInTriangle  []portalData
	maxDepth           uint16
}

func newBestHomogeneousQuery(portals []portalData, maxDepth int, onFilledIndexEntry func()) *bestHomogeneousQuery {
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
		portalsInTriangle:  make([]portalData, 0, len(portals)),
		maxDepth:           uint16(maxDepth),
	}
}

func (q *bestHomogeneousQuery) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.index[p0.Index][p1.Index][p2.Index].Length != invalidLength {
		return
	}
	q.portalsInTriangle = portalsInsideTriangle(q.portals, p0, p1, p2, q.portalsInTriangle)
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle)
}

func (q *bestHomogeneousQuery) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
	localCandidates := append(make([]portalData, 0, len(candidates)), candidates...)
	var bestHomogeneous bestSolution
	for _, portal := range localCandidates {
		minDepth := invalidLength
		{
			candidate0 := q.index[portal.Index][p1.Index][p2.Index]
			if candidate0.Length == invalidLength {
				candidatesInWedge := portalsInsideWedge(localCandidates, portal, p1, p2, q.portalsInTriangle)
				candidate0 = q.findBestHomogeneousAux(portal, p1, p2, candidatesInWedge)
			}
			if candidate0.Length < minDepth {
				minDepth = candidate0.Length
			}
		}
		{
			candidate1 := q.index[portal.Index][p0.Index][p2.Index]
			if candidate1.Length == invalidLength {
				candidatesInWedge := portalsInsideWedge(localCandidates, portal, p0, p2, q.portalsInTriangle)
				candidate1 = q.findBestHomogeneousAux(portal, p0, p2, candidatesInWedge)
			}
			if candidate1.Length < minDepth {
				minDepth = candidate1.Length
			}
		}
		{
			candidate2 := q.index[portal.Index][p0.Index][p1.Index]
			if candidate2.Length == invalidLength {
				candidatesInWedge := portalsInsideWedge(localCandidates, portal, p0, p1, q.portalsInTriangle)
				candidate2 = q.findBestHomogeneousAux(portal, p0, p1, candidatesInWedge)
			}
			if candidate2.Length < minDepth {
				minDepth = candidate2.Length
			}
		}
		if minDepth != invalidLength {
			if minDepth+1 > q.maxDepth {
				minDepth = q.maxDepth - 1
			}
			if minDepth+1 > bestHomogeneous.Length {
				bestHomogeneous.Index = portal.Index
				bestHomogeneous.Length = minDepth + 1
			}
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
func DeepestHomogeneous(portals []Portal, maxDepth int, topLevelScorer topLevelTriangleScorer) ([]Portal, uint16) {
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
	q := newBestHomogeneousQuery(portalsData, maxDepth, onFilledIndexEntry)
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
	bestScore := float32(-math.MaxFloat32)
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				candidate := q.index[p0.Index][p1.Index][p2.Index]
				score := topLevelScorer.scoreTriangle(p0, p1, p2)
				if candidate.Length > bestDepth || (candidate.Length == bestDepth && score > bestScore) {
					bestP0, bestP1, bestP2 = p0, p1, p2
					bestDepth = candidate.Length
					bestScore = score
				}
			}
		}
	}

	resultIndices := []portalIndex{bestP0.Index, bestP1.Index, bestP2.Index}
	resultIndices = appendHomogeneousResult(bestP0.Index, bestP1.Index, bestP2.Index, bestDepth, resultIndices, q.index)
	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, bestDepth
}

func appendHomogeneousResult(p0, p1, p2 portalIndex, maxDepth uint16, result []portalIndex, index [][][]bestSolution) []portalIndex {
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
