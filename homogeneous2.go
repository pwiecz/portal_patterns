package main

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
	index              []portalIndex
	numPortals         uint
	numPortalsSq       uint
	onFilledIndexEntry func()
	portalsInTriangle  [][]portalData
	depth              uint16
	maxDepth           int
	scorer             homogeneousScorer
}

func newBestHomogeneous2Query(portals []portalData, scorer homogeneousScorer, maxDepth int, onFilledIndexEntry func()) *bestHomogeneous2Query {
	numPortals := uint(len(portals))
	index := make([]portalIndex, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i] = invalidPortalIndex
	}
	return &bestHomogeneous2Query{
		portals:            portals,
		index:              index,
		numPortals:         numPortals,
		numPortalsSq:       numPortals * numPortals,
		onFilledIndexEntry: onFilledIndexEntry,
		portalsInTriangle:  make([][]portalData, len(portals)),
		maxDepth:           maxDepth,
		scorer:             scorer,
	}
}

func (q *bestHomogeneous2Query) getIndex(i, j, k portalIndex) portalIndex {
	return q.index[uint(i)*q.numPortalsSq+uint(j)*q.numPortals+uint(k)]
}
func (q *bestHomogeneous2Query) setIndex(i, j, k portalIndex, index portalIndex) {
	q.index[uint(i)*q.numPortalsSq+uint(j)*q.numPortals+uint(k)] = index
}

func (q *bestHomogeneous2Query) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index) != invalidPortalIndex {
		return
	}
	q.portalsInTriangle[0] = portalsInsideTriangle(q.portals, p0, p1, p2, q.portalsInTriangle[0])
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle[0])
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
	q.depth++
	q.portalsInTriangle[q.depth] = append(q.portalsInTriangle[q.depth][:0], candidates...)
	triangleScorer := q.scorer.newTriangleScorer(p0, p1, p2, q.maxDepth)
	for _, portal := range q.portalsInTriangle[q.depth] {
		if q.getIndex(portal.Index, p1.Index, p2.Index) == invalidPortalIndex {
			candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p1, p2)
			q.findBestHomogeneousAux(portal, p1, p2, candidatesInWedge)
		}
		if q.getIndex(portal.Index, p0.Index, p2.Index) == invalidPortalIndex {
			candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p0, p2)
			q.findBestHomogeneousAux(portal, p0, p2, candidatesInWedge)
		}
		if q.getIndex(portal.Index, p0.Index, p1.Index) == invalidPortalIndex {
			candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p0, p1)
			q.findBestHomogeneousAux(portal, p0, p1, candidatesInWedge)
		}
		triangleScorer.scoreCandidate(portal)
	}
	q.onFilledIndexEntry()
	bestMidpoints := triangleScorer.bestMidpoints()
	s0, s1, s2 := sortedIndices(p0.Index, p1.Index, p2.Index)
	q.setIndex(s0, s1, s2, bestMidpoints[0])
	q.setIndex(s0, s2, s1, bestMidpoints[1])
	q.setIndex(s1, s0, s2, bestMidpoints[2])
	q.setIndex(s1, s2, s0, bestMidpoints[3])
	q.setIndex(s2, s0, s1, bestMidpoints[4])
	q.setIndex(s2, s1, s0, bestMidpoints[5])
	q.depth--
}

// DeepestHomogeneous2 - Find deepest homogeneous field that can be made out of portals
func DeepestHomogeneous2(portals []Portal, maxDepth int, scorer homogeneousScorer, topLevelScorer homogeneousTopLevelScorer, progressFunc func(int, int)) ([]Portal, uint16) {
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
	indexEntriesFilledModN := 0
	onFilledIndexEntry := func() {
		indexEntriesFilled++
		indexEntriesFilledModN++
		if indexEntriesFilledModN == everyNth {
			indexEntriesFilledModN = 0
			progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}

	progressFunc(0, numIndexEntries)
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
	q.portalsInTriangle = nil
	progressFunc(numIndexEntries, numIndexEntries)

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
						if q.getIndex(s0.Index, s1.Index, s2.Index) >= invalidPortalIndex-1 {
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
	resultIndices = q.appendHomogeneous2Result(bestP0.Index, bestP1.Index, bestP2.Index, bestDepth, resultIndices)

	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, (uint16)(bestDepth)
}

func (q *bestHomogeneous2Query) appendHomogeneous2Result(p0, p1, p2 portalIndex, maxDepth int, result []portalIndex) []portalIndex {
	if maxDepth == 1 {
		return result
	}
	s0, s1, s2 := sortedIndices(p0, p1, p2)
	s0, s1, s2 = indexOrdering(s0, s1, s2, maxDepth)
	bestP := q.getIndex(s0, s1, s2)
	result = append(result, bestP)
	result = q.appendHomogeneous2Result(bestP, p1, p2, maxDepth-1, result)
	result = q.appendHomogeneous2Result(p0, bestP, p2, maxDepth-1, result)
	result = q.appendHomogeneous2Result(p0, p1, bestP, maxDepth-1, result)
	return result
}
