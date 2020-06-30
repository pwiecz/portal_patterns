package lib

type homogeneousTriangleScorer interface {
	// resets scorer to compute scores for this triangle
	reset(a, b, c portalData, numCandidates int)
	scoreCandidate(p portalData)
	bestMidpoints() [6]portalIndex
}

type homogeneousScorer interface {
	newTriangleScorer(maxDepth int, pure bool) homogeneousTriangleScorer
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
	// a scorer for picking best of the possible solutions of the same depth
	scorer homogeneousScorer
}

func newBestHomogeneous2Query(portals []portalData, scorer homogeneousScorer, maxDepth int, pure bool, onFilledIndexEntry func()) *bestHomogeneous2Query {
	numPortals := uint(len(portals))
	index := make([]portalIndex, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i] = invalidPortalIndex
	}
	triangleScorers := make([]homogeneousTriangleScorer, len(portals))
	for i := 0; i < len(portals); i++ {
		triangleScorers[i] = scorer.newTriangleScorer(maxDepth, pure)
	}
	return &bestHomogeneous2Query{
		portals:            portals,
		index:              index,
		numPortals:         numPortals,
		onFilledIndexEntry: onFilledIndexEntry,
		triangleScorers:    triangleScorers,
		portalsInTriangle:  make([][]portalData, len(portals)),
		maxDepth:           maxDepth,
		scorer:             scorer,
	}
}

func (q *bestHomogeneous2Query) getIndex(i, j, k portalIndex) portalIndex {
	return q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)]
}
func (q *bestHomogeneous2Query) bestMidpointAtDepth(i, j, k portalIndex, depth int) portalIndex {
	s0, s1, s2 := sortedIndices(i, j, k)
	s0, s1, s2 = indexOrdering(s0, s1, s2, depth)
	return q.getIndex(s0, s1, s2)
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
