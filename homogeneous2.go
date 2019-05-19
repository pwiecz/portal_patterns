package main

import "fmt"

type homogeneous2TriangleScorer interface {
	score(p portalData, level int) float32
}

type homogeneous2Scorer interface {
	newTriangleScorer(a, b, c portalData) homogeneous2TriangleScorer
	setTriangleScore(a, b, c uint16, score [6]float32)
}

type homogeneous2TopLevelScorer interface {
	scoreTriangle(a, b, c portalData) float32
}

type bestHomogeneous2Query struct {
	portals            []portalData
	index              [][][]uint16
	onFilledIndexEntry func()
	portalsInTriangle  []portalData
	maxDepth           int
	scorer             homogeneous2Scorer
}

func newBestHomogeneous2Query(portals []portalData, scorer homogeneous2Scorer, maxDepth int, onFilledIndexEntry func()) *bestHomogeneous2Query {
	if maxDepth > 6 {
		panic("Max depth too high")
	}
	if maxDepth <= 0 {
		panic("Max depth too low")
	}
	index := make([][][]uint16, 0, len(portals))
	for i := 0; i < len(portals); i++ {
		index = append(index, make([][]uint16, 0, len(portals)))
		for j := 0; j < len(portals); j++ {
			index[i] = append(index[i], make([]uint16, len(portals)))
			for k := 0; k < len(portals); k++ {
				index[i][j][k] = invalidPortalIndex
			}
		}
	}
	return &bestHomogeneous2Query{
		portals:            portals,
		index:              index,
		onFilledIndexEntry: onFilledIndexEntry,
		portalsInTriangle:  make([]portalData, 0, len(portals)),
		maxDepth:           maxDepth,
		scorer:             scorer,
	}
}

func (q *bestHomogeneous2Query) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.index[p0.Index][p1.Index][p2.Index] != invalidLength {
		return
	}
	q.portalsInTriangle = portalsInsideTriangle(q.portals, p0, p1, p2, q.portalsInTriangle)
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle)
}

func sorted(a, b, c portalData) (portalData, portalData, portalData) {
	if a.Index < b.Index {
		if a.Index < c.Index {
			if b.Index < c.Index {
				return a, b, c
			}
			return a, c, b
		}
		return c, a, b
	}
	if a.Index < c.Index {
		return b, a, c
	}
	if b.Index < c.Index {
		return b, c, a
	}
	return c, b, a
}

func sortedIndices(a, b, c uint16) (uint16, uint16, uint16) {
	if a < b {
		if a < c {
			if b < c {
				return a, b, c
			}
			return a, c, b
		}
		return c, a, b
	}
	if a < c {
		return b, a, c
	}
	if b < c {
		return b, c, a
	}
	return c, b, a
}

func ordering(p0, p1, p2 portalData, index int) (portalData, portalData, portalData) {
	if index == 0 {
		return p0, p1, p2
	}
	if index == 1 {
		return p0, p2, p1
	}
	if index == 2 {
		return p1, p0, p2
	}
	if index == 3 {
		return p1, p2, p0
	}
	if index == 4 {
		return p2, p0, p1
	}
	return p2, p1, p0
}
func indexOrdering(p0, p1, p2 uint16, index int) (uint16, uint16, uint16) {
	if index == 0 {
		return p0, p1, p2
	}
	if index == 1 {
		return p0, p2, p1
	}
	if index == 2 {
		return p1, p0, p2
	}
	if index == 3 {
		return p1, p2, p0
	}
	if index == 4 {
		return p2, p0, p1
	}
	return p2, p1, p0
}

func (q *bestHomogeneous2Query) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) {
	localCandidates := append(make([]portalData, 0, len(candidates)), candidates...)
	bestHomogeneous := [6]uint16{
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
		invalidPortalIndex - 1,
	}
	var bestScore [6]float32
	triangleScorer := q.scorer.newTriangleScorer(p0, p1, p2)
	for _, portal := range localCandidates {
		if q.index[portal.Index][p1.Index][p2.Index] == invalidLength {
			candidatesInWedge := portalsInsideWedge(localCandidates, portal, p1, p2, q.portalsInTriangle)
			q.findBestHomogeneousAux(portal, p1, p2, candidatesInWedge)
		}
		if q.index[portal.Index][p0.Index][p2.Index] == invalidLength {
			candidatesInWedge := portalsInsideWedge(localCandidates, portal, p0, p2, q.portalsInTriangle)
			q.findBestHomogeneousAux(portal, p0, p2, candidatesInWedge)
		}
		if q.index[portal.Index][p0.Index][p1.Index] == invalidLength {
			candidatesInWedge := portalsInsideWedge(localCandidates, portal, p0, p1, q.portalsInTriangle)
			q.findBestHomogeneousAux(portal, p0, p1, candidatesInWedge)
		}
		for i := 0; i < q.maxDepth; i++ {
			score := triangleScorer.score(portal, i)
			if i == 0 && score == 0 {
				panic("zero score")
			}
			if score == 0 {
				break
			}
			if score > bestScore[i] {
				bestScore[i] = score
				bestHomogeneous[i] = portal.Index
			}
		}
	}
	q.onFilledIndexEntry()
	s0, s1, s2 := sortedIndices(p0.Index, p1.Index, p2.Index)
	q.index[s0][s1][s2] = bestHomogeneous[0]
	q.index[s0][s2][s1] = bestHomogeneous[1]
	q.index[s1][s0][s2] = bestHomogeneous[2]
	q.index[s1][s2][s0] = bestHomogeneous[3]
	q.index[s2][s0][s1] = bestHomogeneous[4]
	q.index[s2][s1][s0] = bestHomogeneous[5]
	q.scorer.setTriangleScore(s0, s1, s2, bestScore)
}

// DeepestHomogeneous2 - Find deepest homogeneous field that can be made out of portals
func DeepestHomogeneous2(portals []Portal, maxDepth int) ([]Portal, int) {
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

	scorer := newAvoidThinTriangles2Scorer(portalsData)
	printProgressBar(0, numIndexEntries)
	q := newBestHomogeneous2Query(portalsData, scorer, maxDepth, onFilledIndexEntry)
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

	topLevelScorer := scorer
	var bestDepth int
	var bestP0, bestP1, bestP2 portalData
	var bestScore float32
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				for depth := q.maxDepth - 1; depth >= bestDepth; depth-- {
					s0, s1, s2 := ordering(p0, p1, p2, depth)
					if depth > 0 && q.index[s0.Index][s1.Index][s2.Index] >= invalidPortalIndex-1 {
						continue
					}
					score := topLevelScorer.scoreTriangle(s0, s1, s2)
					if depth > bestDepth || (depth == bestDepth && score > bestScore) {
						bestP0, bestP1, bestP2 = s0, s1, s2
						bestDepth = depth
						bestScore = score
					}
				}
			}
		}
	}
	resultIndices := []uint16{bestP0.Index, bestP1.Index, bestP2.Index}
	resultIndices = appendHomogeneous2Result(bestP0.Index, bestP1.Index, bestP2.Index, bestDepth+1, resultIndices, q.index)
	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, bestDepth + 1
}

func appendHomogeneous2Result(p0, p1, p2 uint16, maxDepth int, result []uint16, index [][][]uint16) []uint16 {
	if maxDepth == 0 {
		return result
	}
	s0, s1, s2 := sortedIndices(p0, p1, p2)
	s0, s1, s2 = indexOrdering(s0, s1, s2, maxDepth-1)
	bestP := index[s0][s1][s2]
	result = append(result, bestP)
	result = appendHomogeneous2Result(bestP, p1, p2, maxDepth-1, result, index)
	result = appendHomogeneous2Result(p0, bestP, p2, maxDepth-1, result, index)
	result = appendHomogeneous2Result(p0, p1, bestP, maxDepth-1, result, index)
	return result
}
