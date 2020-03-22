package lib

import "math"
import "sync"

type bestHomogeneous2MtQuery struct {
	// all the portals
	portals []portalData
	// index of triple of portals to a solution
	// each permutations of the three portals stores the best solution
	// for different depth - 2..7
	index []portalIndex
	// count of portals (used to compute a solution index from indices of three portals)
	numPortals uint
	maxDepth   int
	// depth of solution to be found
	depth uint16
	// accept only candidates that use all the portals within the top level triangle
	perfect bool
}

func newBestHomogeneous2MtQuery(portals []portalData, index []portalIndex, maxDepth, depth int, perfect bool) *bestHomogeneous2MtQuery {
	return &bestHomogeneous2MtQuery{
		portals:    portals,
		index:      index,
		numPortals: uint(len(portals)),
		maxDepth:   maxDepth,
		depth:      uint16(depth),
		perfect:    perfect,
	}
}

func (q *bestHomogeneous2MtQuery) setIndex(i, j, k portalIndex, index portalIndex) {
	q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)] = index
}

type homogeneous2Request struct {
	p0, p1, p2 portalData
}

func bestHomogeneous2Worker(q *bestHomogeneous2MtQuery,
	scorer homogeneousScorer,
	requestChannel chan int,
	responseChannel chan empty,
	wg *sync.WaitGroup) {
	triangleScorer := scorer.newTriangleScorerForDepth(q.depth)
	for req := range requestChannel {
		p0 := q.portals[req]
		for j := req + 1; j < len(q.portals); j++ {
			p1 := q.portals[j]
			for k := j + 1; k < len(q.portals); k++ {
				p2 := q.portals[k]
				q.findBestHomogeneousAux(p0, p1, p2, triangleScorer)
			}
		}
		responseChannel <- empty{}
	}
	wg.Done()
}

func (q *bestHomogeneous2MtQuery) getIndex(i, j, k portalIndex) portalIndex {
	return q.index[(uint(i)*q.numPortals+uint(j))*q.numPortals+uint(k)]
}

func (q *bestHomogeneous2MtQuery) findBestHomogeneousAux(p0, p1, p2 portalData, triangleScorer homogeneousDepthTriangleScorer) {
	s0, s1, s2 := sortedIndices(p0.Index, p1.Index, p2.Index)
	if !q.perfect && q.depth > 2 {
		i0, i1, i2 := indexOrdering(s0, s1, s2, int(q.depth-1))
		if q.getIndex(i0, i1, i2) >= invalidPortalIndex-1 {
			return
		}
	}

	triangleScorer.reset(p0, p1, p2)
	tq := newTriangleQuery(p0.LatLng, p1.LatLng, p2.LatLng)
	if !q.perfect || q.depth > 2 {
		for _, portal := range q.portals {
			if portal.Index == p0.Index || portal.Index == p1.Index || portal.Index == p2.Index || !tq.ContainsPoint(portal.LatLng) {
				continue
			}
			triangleScorer.scoreCandidate(portal)
		}
	} else {
		foundPortal := false
		var onlyPortal portalData
		for _, portal := range q.portals {
			if portal.Index == p0.Index || portal.Index == p1.Index || portal.Index == p2.Index || !tq.ContainsPoint(portal.LatLng) {
				continue
			}
			if foundPortal {
				return
			}
			foundPortal = true
			onlyPortal = portal
		}
		if !foundPortal {
			return
		}
		triangleScorer.scoreCandidate(onlyPortal)
	}

	bestMidpoint := triangleScorer.bestMidpoint()
	if bestMidpoint >= invalidPortalIndex-1 {
		return
	}
	i0, i1, i2 := indexOrdering(s0, s1, s2, int(q.depth))
	q.setIndex(i0, i1, i2, bestMidpoint)
}

type homogeneous2Result struct {
	index      []portalIndex
	portals    []Portal
	numPortals uint
}

func (r *homogeneous2Result) getIndex(i, j, k portalIndex) portalIndex {
	return r.index[(uint(i)*r.numPortals+uint(j))*r.numPortals+uint(k)]
}

type empty struct{}

// DeepestHomogeneous2MT - Find deepest homogeneous field that can be made out of portals
func DeepestHomogeneous2MT(portals []Portal, params homogeneous2Params) ([]Portal, uint16) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}

	portalsData := portalsToPortalData(portals)

	numIndexEntries := len(portals) * (params.maxDepth - 1)
	everyNth := numIndexEntries / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	indexEntriesFilled := 0
	indexEntriesFilledModN := 0

	numPortals := uint(len(portals))
	index := make([]portalIndex, numPortals*numPortals*numPortals)
	for i := 0; i < len(index); i++ {
		index[i] = invalidPortalIndex
	}
	for depth := 2; depth <= params.maxDepth; depth++ {
		requestChannel := make(chan int, params.numWorkers*2)
		responseChannel := make(chan empty, params.numWorkers*2)
		var wg sync.WaitGroup
		wg.Add(params.numWorkers)
		q := newBestHomogeneous2MtQuery(portalsData, index, params.maxDepth, depth, params.perfect)

		for i := 0; i < params.numWorkers; i++ {
			go bestHomogeneous2Worker(q, params.scorer, requestChannel, responseChannel, &wg)
		}
		go func() {
			for i, _ := range portalsData {
				requestChannel <- i
			}
			close(requestChannel)
		}()
		go func() {
			wg.Wait()
			close(responseChannel)
		}()
		for _ = range responseChannel {
			indexEntriesFilled++
			indexEntriesFilledModN++
			if indexEntriesFilledModN == everyNth {
				indexEntriesFilledModN = 0
				params.progressFunc(indexEntriesFilled, numIndexEntries)
			}
		}
	}
	params.progressFunc(numIndexEntries, numIndexEntries)

	r := &homogeneous2Result{
		index:      index,
		portals:    portals,
		numPortals: numPortals,
	}
	bestDepth := 1
	var bestP0, bestP1, bestP2 portalData
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
					s0, s1, s2 := p0, p1, p2
					if depth >= 2 {
						s0, s1, s2 = ordering(p0, p1, p2, depth)
						if r.getIndex(s0.Index, s1.Index, s2.Index) >= invalidPortalIndex-1 {
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
	resultIndices = r.appendHomogeneousResult(bestP0.Index, bestP1.Index, bestP2.Index, bestDepth, resultIndices)
	result := []Portal{}
	for _, index := range resultIndices {
		result = append(result, portals[index])
	}

	return result, uint16(bestDepth)
}

func (r *homogeneous2Result) appendHomogeneousResult(p0, p1, p2 portalIndex, maxDepth int, result []portalIndex) []portalIndex {
	if maxDepth == 1 {
		return result
	}
	s0, s1, s2 := sortedIndices(p0, p1, p2)
	s0, s1, s2 = indexOrdering(s0, s1, s2, maxDepth)
	bestP := r.getIndex(s0, s1, s2)
	result = append(result, bestP)
	result = r.appendHomogeneousResult(bestP, p1, p2, maxDepth-1, result)
	result = r.appendHomogeneousResult(p0, bestP, p2, maxDepth-1, result)
	result = r.appendHomogeneousResult(p0, p1, bestP, maxDepth-1, result)
	return result
}
