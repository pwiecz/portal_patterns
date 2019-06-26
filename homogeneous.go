package main

import "math"

type topLevelTriangleScorer interface {
	scoreTriangle(a, b, c portalData) float32
}

type bestHomogeneousQuery struct {
	portals            []portalData
	index              []bestSolution
	numPortals         int64
	numPortalsSq       int64
	onFilledIndexEntry func()
	portalsInTriangle  []portalData
	maxDepth           uint16
}

func newBestHomogeneousQuery(portals []portalData, maxDepth int, onFilledIndexEntry func()) *bestHomogeneousQuery {
	numPortals := int64(len(portals))
	index := make([]bestSolution, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i].Length = invalidLength
	}
	return &bestHomogeneousQuery{
		portals:            portals,
		index:              index,
		numPortals:         numPortals,
		numPortalsSq:       numPortals * numPortals,
		onFilledIndexEntry: onFilledIndexEntry,
		portalsInTriangle:  make([]portalData, 0, len(portals)),
		maxDepth:           uint16(maxDepth),
	}
}

func (q *bestHomogeneousQuery) getIndex(i, j, k portalIndex) bestSolution {
	return q.index[int64(i)*q.numPortalsSq+int64(j)*q.numPortals+int64(k)]
}
func (q *bestHomogeneousQuery) setIndex(i, j, k portalIndex, s bestSolution) {
	q.index[int64(i)*q.numPortalsSq+int64(j)*q.numPortals+int64(k)] = s
}
func (q *bestHomogeneousQuery) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index).Length != invalidLength {
		return
	}
	q.portalsInTriangle = portalsInsideTriangle(q.portals, p0, p1, p2, q.portalsInTriangle)
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle)
}

func (q *bestHomogeneousQuery) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
	localCandidates := append(make([]portalData, 0, len(candidates)), candidates...)
	bestMidpoint := bestSolution{Index: invalidPortalIndex, Length: 1}
	for _, portal := range localCandidates {
		minDepth := invalidLength
		{
			candidate0 := q.getIndex(portal.Index, p1.Index, p2.Index)
			if candidate0.Length == invalidLength {
				candidatesInWedge := portalsInsideWedge(localCandidates, portal, p1, p2, q.portalsInTriangle)
				candidate0 = q.findBestHomogeneousAux(portal, p1, p2, candidatesInWedge)
			}
			if candidate0.Length < minDepth {
				minDepth = candidate0.Length
			}
		}
		{
			candidate1 := q.getIndex(portal.Index, p0.Index, p2.Index)
			if candidate1.Length == invalidLength {
				candidatesInWedge := portalsInsideWedge(localCandidates, portal, p0, p2, q.portalsInTriangle)
				candidate1 = q.findBestHomogeneousAux(portal, p0, p2, candidatesInWedge)
			}
			if candidate1.Length < minDepth {
				minDepth = candidate1.Length
			}
		}
		{
			candidate2 := q.getIndex(portal.Index, p0.Index, p1.Index)
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
			if minDepth+1 > bestMidpoint.Length {
				bestMidpoint.Index = portal.Index
				bestMidpoint.Length = minDepth + 1
			}
		}
	}
	q.onFilledIndexEntry()
	q.setIndex(p0.Index, p1.Index, p2.Index, bestMidpoint)
	q.setIndex(p0.Index, p2.Index, p1.Index, bestMidpoint)
	q.setIndex(p1.Index, p0.Index, p2.Index, bestMidpoint)
	q.setIndex(p1.Index, p2.Index, p0.Index, bestMidpoint)
	q.setIndex(p2.Index, p0.Index, p1.Index, bestMidpoint)
	q.setIndex(p2.Index, p1.Index, p0.Index, bestMidpoint)
	return bestMidpoint
}

// DeepestHomogeneous - Find deepest homogeneous field that can be made out of portals
func DeepestHomogeneous(portals []Portal, maxDepth int, topLevelScorer topLevelTriangleScorer, progressFunc func(int, int)) ([]Portal, uint16) {
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
			progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}

	progressFunc(0, numIndexEntries)
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
	progressFunc(numIndexEntries, numIndexEntries)

	var bestDepth uint16
	var bestP0, bestP1, bestP2 portalData
	bestScore := float32(-math.MaxFloat32)
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				candidate := q.getIndex(p0.Index, p1.Index, p2.Index)
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
	resultIndices = q.appendHomogeneousResult(bestP0.Index, bestP1.Index, bestP2.Index, bestDepth, resultIndices)
	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, bestDepth
}

func (q *bestHomogeneousQuery) appendHomogeneousResult(p0, p1, p2 portalIndex, maxDepth uint16, result []portalIndex) []portalIndex {
	if maxDepth == 1 {
		return result
	}
	bestP := q.getIndex(p0, p1, p2).Index
	result = append(result, bestP)
	result = q.appendHomogeneousResult(bestP, p1, p2, maxDepth-1, result)
	result = q.appendHomogeneousResult(p0, bestP, p2, maxDepth-1, result)
	result = q.appendHomogeneousResult(p0, p1, bestP, maxDepth-1, result)
	return result
}

func appendHomogeneousPolylines(p0, p1, p2 Portal, maxDepth uint16, result []string, portals []Portal) ([]string, []Portal) {
	if maxDepth == 1 {
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
