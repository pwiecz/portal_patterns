package lib

import "math"

type homogeneousTriangleScorer interface {
	// resets scorer to compute scores for this triangle
	reset(a, b, c portalData, numCandidates int)
	scoreCandidate(p portalData)
	bestMidpoints() [6]portalIndex
}
type homogeneousDepthTriangleScorer interface {
	// resets scorer to compute scores for this triangle
	reset(a, b, c portalData)
	scoreCandidate(p portalData)
	bestMidpoint() portalIndex
}

type homogeneousScorer interface {
	newTriangleScorer(maxDepth int, perfect bool) homogeneousTriangleScorer
	scoreTriangle(a, b, c portalData) float32
}

type bestHomogeneous2Query struct {
	// all the portals
	portals []portalData
	// index of triple of portals to a solution
	// each permutations of the three portals stores the best solution
	// for different depth - 2..7
	index []portalIndex
	// count of portals (used to compute a solution index from indices of three portals)
	numPortals uint
	// callback to be called whenever solution for new triple of portals is found
	onFilledIndexEntry func()
	// preallocated storage for lists of portals within triangles at consecutive recursion depths
	portalsInTriangle [][]portalData
	// preallocated storage for triangle scorers at consecutive recursion depths
	triangleScorers []homogeneousTriangleScorer
	// current recursion depth
	depth uint16
	// maxDepth of solution to be found
	maxDepth int
	// accept only candidates that use all the portals within the top level triangle
	perfect bool
	// a scorer for picking best of the possible solutions of the same depth
	scorer homogeneousScorer
}

func newBestHomogeneous2Query(portals []portalData, scorer homogeneousScorer, maxDepth int, perfect bool, onFilledIndexEntry func()) *bestHomogeneous2Query {
	numPortals := uint(len(portals))
	index := make([]portalIndex, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i] = invalidPortalIndex
	}
	triangleScorers := make([]homogeneousTriangleScorer, len(portals))
	for i := 0; i < len(portals); i++ {
		triangleScorers[i] = scorer.newTriangleScorer(maxDepth, perfect)
	}
	return &bestHomogeneous2Query{
		portals:            portals,
		index:              index,
		numPortals:         numPortals,
		onFilledIndexEntry: onFilledIndexEntry,
		triangleScorers:    triangleScorers,
		portalsInTriangle:  make([][]portalData, len(portals)),
		maxDepth:           maxDepth,
		perfect:            perfect,
		scorer:             scorer,
	}
}

func (q *bestHomogeneous2Query) getIndex(i, j, k portalIndex) portalIndex {
	return q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)]
}
func (q *bestHomogeneous2Query) setIndex(i, j, k portalIndex, index portalIndex) {
	q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)] = index
}

func (q *bestHomogeneous2Query) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index) != invalidPortalIndex {
		return
	}
	q.portalsInTriangle[0] = portalsInsideTriangle(q.portals, p0, p1, p2, q.portalsInTriangle[0])
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle[0])
}

func (q *bestHomogeneous2Query) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) {
	q.depth++
	q.portalsInTriangle[q.depth] = append(q.portalsInTriangle[q.depth][:0], candidates...)
	triangleScorer := q.triangleScorers[q.depth]
	triangleScorer.reset(p0, p1, p2, len(candidates))
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

// DeepestHomogeneous2 - Find deepest homogeneous field that can be made out of portals - single-threaded
func DeepestHomogeneous2(portals []Portal, options ...HomogeneousOption) ([]Portal, uint16) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	params := defaultHomogeneous2Params(len(portals))
	for _, option := range options {
		option.apply2(&params)
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
			params.progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}

	params.progressFunc(0, numIndexEntries)
	q := newBestHomogeneous2Query(portalsData, params.scorer, params.maxDepth, params.perfect, onFilledIndexEntry)
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				if !hasAllIndicesInTheTriple(params.fixedCornerIndices, i, j, k) {
					continue
				}
				q.findBestHomogeneous(p0, p1, p2)
			}
		}
	}
	q.portalsInTriangle = nil
	params.progressFunc(numIndexEntries, numIndexEntries)

	bestDepth := 1
	var bestP0, bestP1, bestP2 portalData
	var bestScore float32 = -math.MaxFloat32
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				if !hasAllIndicesInTheTriple(params.fixedCornerIndices, i, j, k) {
					continue
				}
				p2 := portalsData[k]
				for depth := q.maxDepth; depth >= bestDepth; depth-- {
					s0, s1, s2 := p0, p1, p2
					if depth >= 2 {
						s0, s1, s2 = ordering(p0, p1, p2, depth)
						if q.getIndex(s0.Index, s1.Index, s2.Index) >= invalidPortalIndex-1 {
							continue
						}
					}
					score := params.topLevelScorer.scoreTriangle(s0, s1, s2)
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
