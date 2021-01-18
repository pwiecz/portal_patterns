package lib

import (
	"math"
	"strings"
)

type bestHomogeneousQuery interface {
	findBestHomogeneous(p0, p1, p2 portalData)
	bestMidpointAtDepth(i, j, k portalIndex, depth int) portalIndex
}

type bestHomogeneousNonPureQuery struct {
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
}

func newBestHomogeneousQuery(portals []portalData, maxDepth int, onFilledIndexEntry func()) bestHomogeneousQuery {
	numPortals := uint(len(portals))
	index := make([]bestSolution, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i].Index = invalidPortalIndex
		index[i].Length = invalidLength
	}
	return &bestHomogeneousNonPureQuery{
		portals:            portals,
		index:              index,
		numPortals:         numPortals,
		onFilledIndexEntry: onFilledIndexEntry,
		portalsInTriangle:  make([][]portalData, len(portals)),
		maxDepth:           uint16(maxDepth),
	}
}

func (q *bestHomogeneousNonPureQuery) getIndex(i, j, k portalIndex) bestSolution {
	return q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)]
}
func (q *bestHomogeneousNonPureQuery) bestMidpointAtDepth(i, j, k portalIndex, depth int) portalIndex {
	solution := q.getIndex(i, j, k)
	if int(solution.Length) < depth {
		return invalidPortalIndex
	}
	return solution.Index
}
func (q *bestHomogeneousNonPureQuery) setIndex(i, j, k portalIndex, s bestSolution) {
	q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)] = s
}
func (q *bestHomogeneousNonPureQuery) findBestHomogeneous(p0, p1, p2 portalData) {
	if q.getIndex(p0.Index, p1.Index, p2.Index).Length != invalidLength {
		return
	}
	q.portalsInTriangle[0] = portalsInsideTriangle(q.portals, p0, p1, p2, q.portalsInTriangle[0])
	q.findBestHomogeneousAux(p0, p1, p2, q.portalsInTriangle[0])
}

func (q *bestHomogeneousNonPureQuery) findBestHomogeneousAux(p0, p1, p2 portalData, candidates []portalData) bestSolution {
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
		if candidate1.Length < minDepth {
			minDepth = candidate1.Length
		}

		candidate2 := q.getIndex(portal.Index, p0.Index, p1.Index)
		if candidate2.Length == invalidLength {
			candidatesInWedge := partitionPortalsInsideWedge(candidates, portal, p0, p1)
			candidate2 = q.findBestHomogeneousAux(portal, p0, p1, candidatesInWedge)
		}
		if candidate2.Length < minDepth {
			minDepth = candidate2.Length
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
	requires2 := false
	for _, option := range options {
		if option.requires2() {
			requires2 = true
		} else {
			option.apply(&params)
		}
	}
	portalsData := portalsToPortalData(portals)
	if len(params.fixedCornerIndices) == 3 {
		fixedPortals := []portalData{
			portalsData[params.fixedCornerIndices[0]],
			portalsData[params.fixedCornerIndices[1]],
			portalsData[params.fixedCornerIndices[2]],
		}
		filteredPortalsData := append(fixedPortals,
			portalsInsideTriangle(portalsData,
				fixedPortals[0], fixedPortals[1], fixedPortals[2],
				[]portalData{})...)
		filteredPortals := make([]Portal, 0, len(filteredPortalsData))
		for _, p := range filteredPortalsData {
			filteredPortals = append(filteredPortals, portals[p.Index])
		}
		portals = filteredPortals
		portalsData = portalsToPortalData(portals)
		params.fixedCornerIndices = []int{0, 1, 2}
		for i, option := range options {
			if _, ok := option.(HomogeneousFixedCornerIndices); ok {
				options[i] = (HomogeneousFixedCornerIndices)([]int{0, 1, 2})
			}
		}
	}

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
	var q bestHomogeneousQuery
	if requires2 {
		params2 := defaultHomogeneous2Params(len(portals))
		for _, option := range options {
			option.apply2(&params2)
		}
		params = params2.homogeneousParams
		q = newBestHomogeneous2Query(portalsData, params2.scorer, params2.maxDepth, params2.pure, onFilledIndexEntry)
	} else if params.pure {
		resultIndices, bestDepth := deepestPureHomogeneous(portalsData, params)
		result := []Portal{}
		for _, index := range resultIndices {
			result = append(result, portals[index])
		}

		return result, uint16(bestDepth)
	} else {
		q = newBestHomogeneousQuery(portalsData, params.maxDepth, onFilledIndexEntry)
	}
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
	params.progressFunc(numIndexEntries, numIndexEntries)

	bestP, bestDepth := pickBestTopLevelTriangle(portalsData, params, q)
	resultIndices := []portalIndex{bestP[0].Index, bestP[1].Index, bestP[2].Index}
	resultIndices = append(resultIndices, homogeneousResultIndices(bestP[0].Index, bestP[1].Index, bestP[2].Index, bestDepth, q)...)
	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, uint16(bestDepth)
}

func pickBestTopLevelTriangle(portalsData []portalData, params homogeneousParams, q bestHomogeneousQuery) ([3]portalData, int) {
	bestDepth := 1
	bestTriangle := [3]portalData{}
	bestScore := float32(-math.MaxFloat32)
	for i, p0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			p1 := portalsData[j]
			for k := j + 1; k < len(portalsData); k++ {
				if !hasAllIndicesInTheTriple(params.fixedCornerIndices, i, j, k) {
					continue
				}
				p2 := portalsData[k]
				for depth := params.maxDepth; depth >= bestDepth; depth-- {
					if depth >= 2 {
						if q.bestMidpointAtDepth(p0.Index, p1.Index, p2.Index, depth) >= invalidPortalIndex-1 {
							continue
						}
					}
					score := params.topLevelScorer.scoreTriangle(p0, p1, p2)
					if depth > bestDepth || (depth == bestDepth && score > bestScore) {
						bestTriangle = [3]portalData{p0, p1, p2}
						bestDepth = depth
						bestScore = score
					}
				}
			}
		}
	}
	return bestTriangle, bestDepth

}

func homogeneousResultIndices(p0, p1, p2 portalIndex, depth int, q bestHomogeneousQuery) []portalIndex {
	if depth == 1 {
		return nil
	}
	bestP := q.bestMidpointAtDepth(p0, p1, p2, depth)
	result := []portalIndex{bestP}
	result = append(result, homogeneousResultIndices(bestP, p1, p2, depth-1, q)...)
	result = append(result, homogeneousResultIndices(p0, bestP, p2, depth-1, q)...)
	result = append(result, homogeneousResultIndices(p0, p1, bestP, depth-1, q)...)
	return result
}

func AppendHomogeneousPolylines(p0, p1, p2 Portal, maxDepth uint16, result [][]Portal, portals []Portal) ([][]Portal, []Portal) {
	if maxDepth == 1 {
		return result, portals
	}
	portal := portals[0]
	result = append(result,
		[]Portal{p0, portal},
		[]Portal{p1, portal},
		[]Portal{p2, portal})
	result, portals = AppendHomogeneousPolylines(portal, p1, p2, maxDepth-1, result, portals[1:])
	result, portals = AppendHomogeneousPolylines(p0, portal, p2, maxDepth-1, result, portals)
	result, portals = AppendHomogeneousPolylines(p0, p1, portal, maxDepth-1, result, portals)
	return result, portals
}

func HomogeneousPolylines(depth uint16, result []Portal) [][]Portal {
	if len(result) == 0 {
		return ([][]Portal)(nil)
	}
	polylines := [][]Portal{{result[0], result[1], result[2], result[0]}}
	polylines, _ = AppendHomogeneousPolylines(result[0], result[1], result[2], uint16(depth), polylines, result[3:])
	return polylines
}
func HomogeneousDrawToolsString(depth uint16, result []Portal) string {
	polylines := HomogeneousPolylines(depth, result)
	polylineStrings := make([]string, 0, len(polylines))
	for _, polyline := range polylines {
		polylineStrings = append(polylineStrings, PolylineFromPortalList(polyline))
	}
	return "[" + strings.Join(polylineStrings, ",") + "]"
}
