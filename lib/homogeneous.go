package lib

import "math"

type bestHomogeneousQuery struct {
	// all the portals
	portals            []portalData
	// index of triple of portals to a solution
	// solution for every triple is store six times - for each of the permutations of portals
	index              []bestSolution
	// count of portals (used to compute a solution index from indices of three portals)
	numPortals         uint
	// callback to be called whenever solution for new triple of portals is found
	onFilledIndexEntry func()
	// preallocated storage for lists of portals within triangles at consecutive recursion depths
	portalsInTriangle  [][]portalData
	// current recursion depth
	depth              uint16
	// maxDepth of solution to be found
	maxDepth           uint16
	// accept only candidates that use all the portals within the top level triangle
	perfect            bool
}

func newBestHomogeneousQuery(portals []portalData, maxDepth int, perfect bool, onFilledIndexEntry func()) *bestHomogeneousQuery {
	numPortals := uint(len(portals))
	index := make([]bestSolution, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i].Index = invalidPortalIndex
		index[i].Length = invalidLength
	}
	return &bestHomogeneousQuery{
		portals:            portals,
		index:              index,
		numPortals:         numPortals,
		onFilledIndexEntry: onFilledIndexEntry,
		portalsInTriangle:  make([][]portalData, len(portals)),
		maxDepth:           uint16(maxDepth),
		perfect:            perfect,
	}
}

func (q *bestHomogeneousQuery) getIndex(i, j, k portalIndex) bestSolution {
	return q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)]
}
func (q *bestHomogeneousQuery) setIndex(i, j, k portalIndex, s bestSolution) {
	q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)] = s
}
func (q *bestHomogeneousQuery) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index).Length != invalidLength {
		return
	}
	q.portalsInTriangle[0] = portalsInsideTriangle(q.portals, p0, p1, p2, q.portalsInTriangle[0])
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle[0])
}

func (q *bestHomogeneousQuery) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
	q.depth++
	// make a copy of input slice to slice we'll be iterating over,
	// as we're going to keep modifying the input slice by calling
	//  partitionPortalsInsideWedge().
	q.portalsInTriangle[q.depth] = append(q.portalsInTriangle[q.depth][:0], candidates...)
	bestMidpoint := bestSolution{Index: invalidPortalIndex, Length: 1}
	for _, portal := range q.portalsInTriangle[q.depth] {
		var minDepth uint16
		{
			candidate0 := q.getIndex(portal.Index, p1.Index, p2.Index)
			if candidate0.Length == invalidLength {
				candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p1, p2)
				candidate0 = q.findBestHomogeneousAux(portal, p1, p2, candidatesInWedge)
			}
			minDepth = candidate0.Length
		}
		{
			candidate1 := q.getIndex(portal.Index, p0.Index, p2.Index)
			if candidate1.Length == invalidLength {
				candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p0, p2)
				candidate1 = q.findBestHomogeneousAux(portal, p0, p2, candidatesInWedge)
			}
			if !q.perfect && candidate1.Length < minDepth {
				minDepth = candidate1.Length
			} else if q.perfect && candidate1.Length != minDepth {
				continue
			}
		}
		{
			candidate2 := q.getIndex(portal.Index, p0.Index, p1.Index)
			if candidate2.Length == invalidLength {
				candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p0, p1)
				candidate2 = q.findBestHomogeneousAux(portal, p0, p1, candidatesInWedge)
			}
			if !q.perfect && candidate2.Length < minDepth {
				minDepth = candidate2.Length
			} else if q.perfect && candidate2.Length != minDepth {
				continue
			}
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
	if q.perfect && bestMidpoint.Length == 1 && len(candidates) != 0 {
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

// DeepestHomogeneous - Find deepest homogeneous field that can be made out of portals
func DeepestHomogeneous(portals []Portal, options ...HomogeneousOption) ([]Portal, uint16) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	params := defaultHomogeneousParams()
	for _, option := range options {
		option.apply(&params)
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
	q := newBestHomogeneousQuery(portalsData, params.maxDepth, params.perfect, onFilledIndexEntry)
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

	var bestDepth uint16
	var bestP0, bestP1, bestP2 portalData
	bestScore := float32(-math.MaxFloat32)
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				p2 := portalsData[k]
				if !hasAllIndicesInTheTriple(params.fixedCornerIndices, i, j, k) {
					continue
				}
				candidate := q.getIndex(p0.Index, p1.Index, p2.Index)
				score := params.topLevelScorer.scoreTriangle(p0, p1, p2)
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

func AppendHomogeneousPolylines(p0, p1, p2 Portal, maxDepth uint16, result []string, portals []Portal) ([]string, []Portal) {
	if maxDepth == 1 {
		return result, portals
	}
	portal := portals[0]
	result = append(result,
		PolylineFromPortalList([]Portal{p0, portal}),
		PolylineFromPortalList([]Portal{p1, portal}),
		PolylineFromPortalList([]Portal{p2, portal}))
	result, portals = AppendHomogeneousPolylines(portal, p1, p2, maxDepth-1, result, portals[1:])
	result, portals = AppendHomogeneousPolylines(p0, portal, p2, maxDepth-1, result, portals)
	result, portals = AppendHomogeneousPolylines(p0, p1, portal, maxDepth-1, result, portals)
	return result, portals
}
