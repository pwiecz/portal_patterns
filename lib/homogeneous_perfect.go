package lib

type bestHomogeneousPerfectQuery struct {
	// all the portals
	portals []portalData
	// index of triple of portals to a solution
	// solution for every triple is store six times - for each of the permutations of portals
	index []bestSolution
	// count of portals (used to compute a solution index from indices of three portals)
	numPortals uint
	// callback to be called whenever solution for new triple of portals is found
	onFilledIndexEntry func()
	// preallocated storage for lists of portals within triangles at consecutive recursion depths
	portalsInTriangle [][]portalData
	// current recursion depth
	depth uint16
	// maxDepth of solution to be found
	maxDepth uint16
	// Possible number of portals in a homogeneous field at different depths up to maxDepth
	legalNumPortals []int
}

func newBestHomogeneousPerfectQuery(portals []portalData, maxDepth int, onFilledIndexEntry func()) bestHomogeneousQuery {
	numPortals := uint(len(portals))
	index := make([]bestSolution, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i].Index = invalidPortalIndex
		index[i].Length = invalidLength
	}
	legalNumPortals := make([]int, maxDepth)
	legalNumPortals[0] = 0
	for i := 1; i < maxDepth; i++ {
		legalNumPortals[i] = (legalNumPortals[i-1]+1)*3 - 2
	}
	return &bestHomogeneousPerfectQuery{
		portals:            portals,
		index:              index,
		numPortals:         numPortals,
		onFilledIndexEntry: onFilledIndexEntry,
		portalsInTriangle:  make([][]portalData, len(portals)),
		maxDepth:           uint16(maxDepth),
		legalNumPortals:    legalNumPortals,
	}
}

func (q *bestHomogeneousPerfectQuery) getIndex(i, j, k portalIndex) bestSolution {
	return q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)]
}
func (q *bestHomogeneousPerfectQuery) bestMidpointAtDepth(i, j, k portalIndex, depth int) portalIndex {
	solution := q.getIndex(i, j, k)
	if int(solution.Length) < depth {
		return invalidPortalIndex
	}
	return solution.Index
}
func (q *bestHomogeneousPerfectQuery) setIndex(i, j, k portalIndex, s bestSolution) {
	q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)] = s
}
func (q *bestHomogeneousPerfectQuery) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index).Length != invalidLength {
		return
	}
	q.portalsInTriangle[0] = portalsInsideTriangle(q.portals, p0, p1, p2, q.portalsInTriangle[0])
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle[0])
}

func (q *bestHomogeneousPerfectQuery) isNumberOfPortalsOk(numPortals int) bool {
	for _, num := range q.legalNumPortals {
		if num == numPortals {
			return true
		}
	}
	return false
}
func (q *bestHomogeneousPerfectQuery) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
	q.depth++
	// make a copy of input slice to slice we'll be iterating over,
	// as we're going to keep modifying the input slice by calling
	//  partitionPortalsInsideWedge().
	q.portalsInTriangle[q.depth] = append(q.portalsInTriangle[q.depth][:0], candidates...)
	bestMidpoint := bestSolution{Index: invalidPortalIndex, Length: 1}
	for _, portal := range q.portalsInTriangle[q.depth] {
		candidate0 := q.getIndex(portal.Index, p1.Index, p2.Index)
		if candidate0.Length == invalidLength {
			candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p1, p2)
			candidate0 = q.findBestHomogeneousAux(portal, p1, p2, candidatesInWedge)
		}

		minDepth := candidate0.Length

		candidate1 := q.getIndex(portal.Index, p0.Index, p2.Index)
		if candidate1.Length == invalidLength {
			candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p0, p2)
			candidate1 = q.findBestHomogeneousAux(portal, p0, p2, candidatesInWedge)
		}
		if candidate1.Length != minDepth {
			// it's faster not to break here, but to keep using already partitioned
			// portal list.
			minDepth = 0
		}

		candidate2 := q.getIndex(portal.Index, p0.Index, p1.Index)
		if candidate2.Length == invalidLength {
			candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p0, p1)
			candidate2 = q.findBestHomogeneousAux(portal, p0, p1, candidatesInWedge)
		}
		if candidate2.Length != minDepth {
			minDepth = 0
		}

		if minDepth+1 > q.maxDepth {
			minDepth = q.maxDepth - 1
		}
		if minDepth+1 > bestMidpoint.Length {
			bestMidpoint.Index = portal.Index
			bestMidpoint.Length = minDepth + 1
		}
	}
	q.onFilledIndexEntry()
	if bestMidpoint.Length == 1 && len(candidates) != 0 {
		bestMidpoint.Length = 0
	}
	q.setIndex(p0.Index, p1.Index, p2.Index, bestMidpoint)
	q.setIndex(p0.Index, p2.Index, p1.Index, bestMidpoint)
	q.setIndex(p1.Index, p0.Index, p2.Index, bestMidpoint)
	q.setIndex(p1.Index, p2.Index, p0.Index, bestMidpoint)
	q.setIndex(p2.Index, p0.Index, p1.Index, bestMidpoint)
	q.setIndex(p2.Index, p1.Index, p0.Index, bestMidpoint)
	q.depth--
	return bestMidpoint
}
