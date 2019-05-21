package main

import "fmt"
import "math"

type homogeneousTriangleScorer interface {
	scoreCandidate(p portalData)
	bestMidpoints() [6]portalIndex
}

type homogeneousScorer interface {
	newTriangleScorer(a, b, c portalData, maxDepth int) homogeneousTriangleScorer
}

type homogeneousTopLevelScorer interface {
	scoreTriangle(a, b, c portalData) float32
}

type bestHomogeneous2Query struct {
	portals            []portalData
	index              [][][]portalIndex
	onFilledIndexEntry func()
	portalsInTriangle  []portalData
	maxDepth           int
	scorer             homogeneousScorer
}

func newBestHomogeneous2Query(portals []portalData, scorer homogeneousScorer, maxDepth int, onFilledIndexEntry func()) *bestHomogeneous2Query {
	index := make([][][]portalIndex, 0, len(portals))
	for i := 0; i < len(portals); i++ {
		index = append(index, make([][]portalIndex, 0, len(portals)))
		for j := 0; j < len(portals); j++ {
			index[i] = append(index[i], make([]portalIndex, len(portals)))
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
	if q.index[p0.Index][p1.Index][p2.Index] != invalidPortalIndex {
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

func sortedIndices(a, b, c portalIndex) (portalIndex, portalIndex, portalIndex) {
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
	switch index {
	case 2:
		return p0, p1, p2
	case 3:
		return p0, p2, p1
	case 4:
		return p1, p0, p2
	case 5:
		return p1, p2, p0
	case 6:
		return p2, p0, p1
	default:
		return p2, p1, p0
	}
}
func indexOrdering(p0, p1, p2 portalIndex, index int) (portalIndex, portalIndex, portalIndex) {
	switch index {
	case 2:
		return p0, p1, p2
	case 3:
		return p0, p2, p1
	case 4:
		return p1, p0, p2
	case 5:
		return p1, p2, p0
	case 6:
		return p2, p0, p1
	default:
		return p2, p1, p0
	}
}

func (q *bestHomogeneous2Query) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) {
	localCandidates := append(make([]portalData, 0, len(candidates)), candidates...)
	triangleScorer := q.scorer.newTriangleScorer(p0, p1, p2, q.maxDepth)
	for _, portal := range localCandidates {
		if q.index[portal.Index][p1.Index][p2.Index] == invalidPortalIndex {
			candidatesInWedge := portalsInsideWedge(localCandidates, portal, p1, p2, q.portalsInTriangle)
			q.findBestHomogeneousAux(portal, p1, p2, candidatesInWedge)
		}
		if q.index[portal.Index][p0.Index][p2.Index] == invalidPortalIndex {
			candidatesInWedge := portalsInsideWedge(localCandidates, portal, p0, p2, q.portalsInTriangle)
			q.findBestHomogeneousAux(portal, p0, p2, candidatesInWedge)
		}
		if q.index[portal.Index][p0.Index][p1.Index] == invalidPortalIndex {
			candidatesInWedge := portalsInsideWedge(localCandidates, portal, p0, p1, q.portalsInTriangle)
			q.findBestHomogeneousAux(portal, p0, p1, candidatesInWedge)
		}
		triangleScorer.scoreCandidate(portal)
	}
	q.onFilledIndexEntry()
	bestMidpoints := triangleScorer.bestMidpoints()
	s0, s1, s2 := sortedIndices(p0.Index, p1.Index, p2.Index)
	q.index[s0][s1][s2] = bestMidpoints[0]
	q.index[s0][s2][s1] = bestMidpoints[1]
	q.index[s1][s0][s2] = bestMidpoints[2]
	q.index[s1][s2][s0] = bestMidpoints[3]
	q.index[s2][s0][s1] = bestMidpoints[4]
	q.index[s2][s1][s0] = bestMidpoints[5]
}

// DeepestHomogeneous2 - Find deepest homogeneous field that can be made out of portals
func DeepestHomogeneous2(portals []Portal, maxDepth int, scorer homogeneousScorer, topLevelScorer homogeneousTopLevelScorer) ([]Portal, uint16) {
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

	bestDepth := 1
	var bestP0, bestP1, bestP2 portalData
	bestScore := float32(-math.MaxFloat32)
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				for depth := q.maxDepth; depth >= bestDepth; depth-- {
					s0, s1, s2 := p0, p1, p2
					if depth >= 2 {
						s0, s1, s2 = ordering(p0, p1, p2, depth)
						if q.index[s0.Index][s1.Index][s2.Index] >= invalidPortalIndex-1 {
							continue
						}
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

	resultIndices := []portalIndex{bestP0.Index, bestP1.Index, bestP2.Index}
	resultIndices = appendHomogeneous2Result(bestP0.Index, bestP1.Index, bestP2.Index, bestDepth, resultIndices, q.index)

	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, (uint16)(bestDepth)
}

func appendHomogeneous2Result(p0, p1, p2 portalIndex, maxDepth int, result []portalIndex, index [][][]portalIndex) []portalIndex {
	if maxDepth == 1 {
		return result
	}
	s0, s1, s2 := sortedIndices(p0, p1, p2)
	s0, s1, s2 = indexOrdering(s0, s1, s2, maxDepth)
	bestP := index[s0][s1][s2]
	result = append(result, bestP)
	result = appendHomogeneous2Result(bestP, p1, p2, maxDepth-1, result, index)
	result = appendHomogeneous2Result(p0, bestP, p2, maxDepth-1, result, index)
	result = appendHomogeneous2Result(p0, p1, bestP, maxDepth-1, result, index)
	return result
}
