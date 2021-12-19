package lib

import (
	"fmt"
	"math"
	"sync"

	"github.com/golang/geo/r3"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

type homogeneousPureNode struct {
	index      portalIndex
	start, end float64
	distance   s1.ChordAngle
}

// specialize a simple sorting function, to limit overhead of a std sort package.
func sortHomogeneousPureNodesByDistance(nodes []homogeneousPureNode) {
	if len(nodes) < 2 {
		return
	}

	left, right := uint(0), uint(len(nodes)-1)

	// Pick a pivot
	pivotIndex := (left + right) / 2

	// Move the pivot to the right
	nodes[pivotIndex], nodes[right] = nodes[right], nodes[pivotIndex]

	// Pile elements smaller than the pivot on the left
	for i := range nodes {
		if nodes[i].distance < nodes[right].distance {
			nodes[i], nodes[left] = nodes[left], nodes[i]
			left++
		}
	}

	// Place the pivot after the last smaller element
	nodes[left], nodes[right] = nodes[right], nodes[left]

	// Go down the rabbit hole
	sortHomogeneousPureNodesByDistance(nodes[:left])
	sortHomogeneousPureNodesByDistance(nodes[left+1:])
}

type lvlNTriangleQuery struct {
	portals                      []portalData
	disabledPortals              []portalData
	expectedNumPortalsInTriangle int
}

func newLvlNTriangleQuery(portals []portalData, disabledPortals []portalData, level int) *lvlNTriangleQuery {
	expectedNumPortalsInTriangle := 0
	for i := 1; i < level; i++ {
		expectedNumPortalsInTriangle = expectedNumPortalsInTriangle*3 + 1
	}
	return &lvlNTriangleQuery{
		portals:                      portals,
		disabledPortals:              disabledPortals,
		expectedNumPortalsInTriangle: expectedNumPortalsInTriangle,
	}
}

type lvlNTriangleRequest struct {
	third []portalIndex
	p0    portalData
	p1    portalData
}

func lvlNTriangleWorker(
	q *lvlNTriangleQuery,
	requestChannel, responseChannel chan lvlNTriangleRequest,
	wg *sync.WaitGroup) {
	normalizedVector := func(b0, b1 s2.Point) r3.Vector {
		// Let's care about memory consumption and not precompute
		// the norms for each portal pair.
		return b1.Cross(b0.Vector).Normalize()
	}

	portalsLeftOfLine := []homogeneousPureNode{}
	disabledPortalsLeftOfLine := []homogeneousPureNode{}
	portalsInTriangle := []portalIndex{}
	for req := range requestChannel {
		req.third = req.third[:0]
		p0, p1 := req.p0, req.p1
		portalsLeftOfLine = portalsLeftOfLine[:0]
		disabledPortalsLeftOfLine = disabledPortalsLeftOfLine[:0]
		p01, p10 := normalizedVector(p0.LatLng, p1.LatLng), normalizedVector(p1.LatLng, p0.LatLng)
		distQuery := newDistanceQuery(p0.LatLng, p1.LatLng)
		for _, p2 := range q.portals {
			if p2.Index == p0.Index || p2.Index == p1.Index {
				continue
			}
			if !s2.Sign(p2.LatLng, p0.LatLng, p1.LatLng) {
				continue
			}

			a0 := p01.Dot(normalizedVector(p1.LatLng, p2.LatLng)) // acos of angle p0,p1,p2
			a1 := p10.Dot(normalizedVector(p0.LatLng, p2.LatLng)) // acos of angle p1,p0,p2
			dist := distQuery.ChordAngle(p2.LatLng)
			portalsLeftOfLine = append(portalsLeftOfLine, homogeneousPureNode{
				p2.Index, a0, a1, dist})
		}
		sortHomogeneousPureNodesByDistance(portalsLeftOfLine)
		for _, dp := range q.disabledPortals {
			if !s2.Sign(dp.LatLng, p0.LatLng, p1.LatLng) {
				continue
			}

			a0 := p01.Dot(normalizedVector(p1.LatLng, dp.LatLng)) // acos of angle p0,p1,dp
			a1 := p10.Dot(normalizedVector(p0.LatLng, dp.LatLng)) // acos of angle p1,p0,dp
			dist := distQuery.ChordAngle(dp.LatLng)
			disabledPortalsLeftOfLine = append(disabledPortalsLeftOfLine, homogeneousPureNode{
				dp.Index, a0, a1, dist})
		}
	thirdPortalLoop:
		for k, node := range portalsLeftOfLine {
			// Emit each triangle only once to make sure we have consistent data,
			// even in the face of duplicate or colinear portals.
			// So emit triangle only if p0 is the smallest of its vertices.
			if node.index <= p0.Index {
				continue
			}
			for _, disabledPortal := range disabledPortalsLeftOfLine {
				// Triangle contains a disabled portal so cannot make a pure field.
				if disabledPortal.start <= node.start && disabledPortal.end <= node.end && disabledPortal.distance <= node.distance {
					break thirdPortalLoop
				}
			}
			portalsInTriangle = portalsInTriangle[:0]
			numPortalsInTriangle := 0
			for j := 0; j < k; j++ {
				if portalsLeftOfLine[j].start <= node.start && portalsLeftOfLine[j].end <= node.end {
					portalsInTriangle = append(portalsInTriangle, portalsLeftOfLine[j].index)
					numPortalsInTriangle++
					if numPortalsInTriangle > q.expectedNumPortalsInTriangle {
						break
					}
				}
			}
			if numPortalsInTriangle == q.expectedNumPortalsInTriangle && areValidPureHomogeneousPortals(p0.Index, p1.Index, node.index, portalsInTriangle, q.portals) {
				req.third = append(req.third, node.index)
			}
		}
		responseChannel <- req
	}
	wg.Done()
}

type edge struct {
	p0, p1 portalIndex
}
type triangle struct {
	p0, p1, p2 portalIndex
}
type mergeTrianglesRequest struct {
	triangles []triangle
	p0        portalIndex
	p1        portalIndex
}

func mergeTrianglesWorker(
	portals []portalData,
	triangles [][]portalIndex,
	requestChannel, responseChannel chan mergeTrianglesRequest,
	wg *sync.WaitGroup) {
	numPortals := uint32(len(portals))
	for req := range requestChannel {
		req.triangles = req.triangles[:0]
		// p0 is the central portal of the triangle, p1 is one of the corners.
		// Find two remaining corners.
		p0, p1 := req.p0, req.p1
		edgeIndex := uint32(p0)*numPortals + uint32(p1)
		revEdgeIndex := uint32(p1)*numPortals + uint32(p0)
		for _, third0 := range triangles[edgeIndex] {
			// Emit each triangle only once to make sure we have consistent
			// data, even in the face of duplicate or colinear portals.
			// So emit triangle only if third0 is the smallest of its vertices.
			if third0 >= p1 {
				continue
			}
			for _, third1 := range triangles[revEdgeIndex] {
				// See comment above.
				if third0 >= third1 || !s2.Sign(portals[p0].LatLng, portals[third0].LatLng, portals[third1].LatLng) {
					continue
				}
				thirdEdgeIndex := uint32(third0)*numPortals + uint32(third1)
				for _, third2 := range triangles[thirdEdgeIndex] {
					if third2 == p0 {
						req.triangles = append(req.triangles, triangle{p1, third0, third1})
					}
				}
			}
		}
		responseChannel <- req
	}
	wg.Done()
}

func findAllLvlNTriangles(portals []portalData, params homogeneousPureParams, level int) ([][]portalIndex, []edge) {
	resultCache := sync.Pool{
		New: func() interface{} {
			return []portalIndex{}
		},
	}

	requestChannel := make(chan lvlNTriangleRequest, params.numWorkers)
	responseChannel := make(chan lvlNTriangleRequest, params.numWorkers)
	var wg sync.WaitGroup
	wg.Add(params.numWorkers)
	q := newLvlNTriangleQuery(portals, params.disabledPortals, level)
	for i := 0; i < params.numWorkers; i++ {
		go lvlNTriangleWorker(q, requestChannel, responseChannel, &wg)
	}
	go func() {
		for i, p0 := range portals {
			for _, p1 := range portals[i+1:] {
				requestChannel <- lvlNTriangleRequest{
					p0:    p0,
					p1:    p1,
					third: resultCache.Get().([]portalIndex),
				}
			}
		}
		close(requestChannel)
	}()
	go func() {
		wg.Wait()
		close(responseChannel)
	}()
	lvlNTriangles := make([][]portalIndex, len(portals)*len(portals))
	lvlNEdges := []edge{}

	numPairs := len(portals) * (len(portals) - 1) / 2
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}

	params.progressFunc(0, numPairs)
	numProcessedPairs := 0
	numProcessedPairsModN := 0

	numPortals := uint32(len(portals))
	numLvlNTriangles := 0
	for resp := range responseChannel {
		if len(resp.third) > 0 {
			edge0Index := uint32(resp.p0.Index)*numPortals + uint32(resp.p1.Index)
			if len(lvlNTriangles[edge0Index]) == 0 {
				lvlNEdges = append(lvlNEdges, edge{resp.p0.Index, resp.p1.Index})
			}
			lvlNTriangles[edge0Index] = append(lvlNTriangles[edge0Index], resp.third...)
			for _, third := range resp.third {
				edge1Index := uint32(resp.p1.Index)*numPortals + uint32(third)
				if len(lvlNTriangles[edge1Index]) == 0 {
					lvlNEdges = append(lvlNEdges, edge{resp.p1.Index, third})
				}
				lvlNTriangles[edge1Index] = append(lvlNTriangles[edge1Index], resp.p0.Index)
				edge2Index := uint32(third)*numPortals + uint32(resp.p0.Index)
				if len(lvlNTriangles[edge2Index]) == 0 {
					lvlNEdges = append(lvlNEdges, edge{third, resp.p0.Index})
				}
				lvlNTriangles[edge2Index] = append(lvlNTriangles[edge2Index], resp.p1.Index)
			}
			numLvlNTriangles += len(resp.third)
		}
		resultCache.Put(resp.third[:0])
		numProcessedPairs++
		numProcessedPairsModN++
		if numProcessedPairsModN == everyNth {
			numProcessedPairsModN = 0
			params.progressFunc(numProcessedPairs, numPairs)
		}
	}
	params.progressFunc(numPairs, numPairs)

	return lvlNTriangles, lvlNEdges
}

func deepestPureHomogeneous(portals []portalData, params homogeneousPureParams) ([]portalIndex, int) {
	var prevTriangles [][]portalIndex
	var prevEdges []edge
	initialLevel := 4
	for {
		prevTriangles, prevEdges = findAllLvlNTriangles(portals, params, initialLevel)
		if len(prevEdges) > 0 || initialLevel <= 1 {
			break
		}
		initialLevel--
	}

	resultCache := sync.Pool{
		New: func() interface{} {
			return make([]triangle, 0, len(portals))
		},
	}

	bestDepth := initialLevel
	for depth := initialLevel + 1; depth < params.maxDepth; depth++ {
		requestChannel := make(chan mergeTrianglesRequest, params.numWorkers)
		responseChannel := make(chan mergeTrianglesRequest, params.numWorkers)
		var wg sync.WaitGroup
		wg.Add(params.numWorkers)
		for i := 0; i < params.numWorkers; i++ {
			go mergeTrianglesWorker(portals, prevTriangles, requestChannel, responseChannel, &wg)
		}

		newTriangles := 0

		go func() {
			for _, commonEdge := range prevEdges {
				requestChannel <- mergeTrianglesRequest{
					p0:        commonEdge.p0,
					p1:        commonEdge.p1,
					triangles: resultCache.Get().([]triangle),
				}
			}
			close(requestChannel)
		}()
		go func() {
			wg.Wait()
			close(responseChannel)
		}()

		numEdges := len(prevEdges)
		everyNth := numEdges / 1000
		if everyNth < 50 {
			everyNth = 2
		}

		params.progressFunc(0, numEdges)
		numProcessedEdges := 0
		numProcessedEdgesModN := 0

		lvlNTriangles := make([][]portalIndex, len(portals)*len(portals))
		lvlNEdges := []edge{}
		numPortals := uint32(len(portals))

		for resp := range responseChannel {
			for _, triangle := range resp.triangles {
				newTriangles++
				edgeIndex0 := uint32(triangle.p0)*numPortals + uint32(triangle.p1)
				lvlNTriangles[edgeIndex0] = append(lvlNTriangles[edgeIndex0], triangle.p2)
				if len(lvlNTriangles[edgeIndex0]) == 1 {
					lvlNEdges = append(lvlNEdges, edge{triangle.p0, triangle.p1})
				}
				edgeIndex1 := uint32(triangle.p1)*numPortals + uint32(triangle.p2)
				lvlNTriangles[edgeIndex1] = append(lvlNTriangles[edgeIndex1], triangle.p0)
				if len(lvlNTriangles[edgeIndex1]) == 1 {
					lvlNEdges = append(lvlNEdges, edge{triangle.p1, triangle.p2})
				}
				edgeIndex2 := uint32(triangle.p2)*numPortals + uint32(triangle.p0)
				lvlNTriangles[edgeIndex2] = append(lvlNTriangles[edgeIndex2], triangle.p1)
				if len(lvlNTriangles[edgeIndex2]) == 1 {
					lvlNEdges = append(lvlNEdges, edge{triangle.p2, triangle.p0})
				}
			}
			resultCache.Put(resp.triangles)

			numProcessedEdges++
			numProcessedEdgesModN++
			if numProcessedEdgesModN == everyNth {
				numProcessedEdgesModN = 0
				params.progressFunc(numProcessedEdges, numEdges)
			}
		}
		params.progressFunc(numEdges, numEdges)

		if len(lvlNEdges) == 0 {
			break
		}
		prevTriangles = lvlNTriangles
		prevEdges = lvlNEdges
		bestDepth = depth
	}

	var bestP0, bestP1, bestP2 int
	foundSolution := false
	bestTriangleScore := float32(-math.MaxFloat32)
	for edge, edgeTriangles := range prevTriangles {
		if len(edgeTriangles) == 0 {
			continue
		}
		p0 := edge / len(portals)
		p1 := edge % len(portals)
		// Every triangle is stored three times on the list pick only one representative.
		if p0 >= p1 {
			continue
		}
		for _, p2 := range edgeTriangles {
			if p0 >= int(p2) {
				continue
			}
			if !hasAllIndicesInTheTriple(params.fixedCornerIndices, p0, p1, int(p2)) {
				continue
			}
			score := params.scorer.scoreTrianglePure(
				portals[p0], portals[p1], portals[p2], bestDepth, portals)
			if score > bestTriangleScore {
				bestTriangleScore = score
				bestP0, bestP1, bestP2 = p0, p1, int(p2)
				foundSolution = true
			}
		}
	}

	if !foundSolution {
		return []portalIndex{}, 0
	}

	var triangleVertices func(p0, p1, p2 portalData, depth int, portals []portalData) []portalIndex
	triangleVertices = func(p0, p1, p2 portalData, depth int, portals []portalData) []portalIndex {
		if depth == 1 {
			return []portalIndex{}
		}
		portalsInTriangle := portalsInsideTriangle(portals, p0, p1, p2, nil)
		center := findHomogeneousCenterPortal(p0, p1, p2, portalsInTriangle)
		result := []portalIndex{portalIndex(center.Index)}
		result = append(result, triangleVertices(center, p1, p2, depth-1, portalsInTriangle)...)
		result = append(result, triangleVertices(p0, center, p2, depth-1, portalsInTriangle)...)
		result = append(result, triangleVertices(p0, p1, center, depth-1, portalsInTriangle)...)
		return result

	}
	return append([]portalIndex{portalIndex(bestP0), portalIndex(bestP1), portalIndex(bestP2)},
		triangleVertices(portals[bestP0], portals[bestP1], portals[bestP2], bestDepth, portals)...), bestDepth
}

// Assuming p0, p1, p2 are corners of a pure homogeneous field, find its center portal.
// Panic if no suitable center portal found.
func findHomogeneousCenterPortal(p0, p1, p2 portalData, portalsInTriangle []portalData) portalData {
	for _, candidate := range portalsInTriangle {
		q0 := newTriangleQuery(p0.LatLng, p1.LatLng, candidate.LatLng)
		q1 := newTriangleQuery(p1.LatLng, p2.LatLng, candidate.LatLng)
		q2 := newTriangleQuery(p2.LatLng, p0.LatLng, candidate.LatLng)
		c0, c1, c2 := 0, 0, 0
		for _, p := range portalsInTriangle {
			if p.Index == candidate.Index {
				continue
			}
			if q0.ContainsPoint(p.LatLng) {
				c0++
			}
			if q1.ContainsPoint(p.LatLng) {
				c1++
			}
			if q2.ContainsPoint(p.LatLng) {
				c2++
			}
		}
		if c0 == c1 && c0 == c2 {
			return candidate
		}
	}
	panic("Could not find center portal")
}

func areValidPureHomogeneousPortals(p0, p1, p2 portalIndex, inside []portalIndex, portals []portalData) bool {
	if len(inside) <= 1 {
		return true
	}
	insideCopy := make([]portalIndex, len(inside)-1)
	for candidate := 0; candidate < len(inside); candidate++ {
		insideCopy[0] = inside[candidate]
		q0 := newTriangleQuery(portals[p0].LatLng, portals[p1].LatLng, portals[insideCopy[0]].LatLng)
		q1 := newTriangleQuery(portals[p1].LatLng, portals[p2].LatLng, portals[insideCopy[0]].LatLng)
		q2 := newTriangleQuery(portals[p2].LatLng, portals[p0].LatLng, portals[insideCopy[0]].LatLng)
		c0, c1, c2 := 0, 0, 0
		for i, pi := range inside {
			if i == candidate {
				continue
			}
			p := portals[pi].LatLng
			if q0.ContainsPoint(p) {
				insideCopy[c0+c1] = insideCopy[c0]
				insideCopy[c0] = pi
				c0++
			} else if q1.ContainsPoint(p) {
				insideCopy[c0+c1] = pi
				c1++
			} else if q2.ContainsPoint(p) {
				insideCopy[len(insideCopy)-c2-1] = pi
				c2++
			} else {
				// We're hitting some collinear points or other numeric accuracy issues.
				// Better ignore this set of points or they may be causing issues later on.
				return false
			}

		}
		if c0+c1+c2+1 != len(inside) {
			panic(fmt.Errorf("%d,%d,%d,%d", c0, c1, c2, len(inside)))
		}
		if c0 == c2 && c0 == c1 &&
			areValidPureHomogeneousPortals(p0, p1, inside[candidate], insideCopy[:c0], portals) &&
			areValidPureHomogeneousPortals(p1, p2, inside[candidate], insideCopy[c0:c0+c1], portals) &&
			areValidPureHomogeneousPortals(p2, p0, inside[candidate], insideCopy[c0+c1:], portals) {
			return true
		}
		inside[0], inside[candidate] = inside[candidate], inside[0]
	}
	return false
}
