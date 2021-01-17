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

type lvl2TriangleQuery struct {
	portals []portalData
}

func newLvl2TriangleQuery(portals []portalData) *lvl2TriangleQuery {
	return &lvl2TriangleQuery{
		portals: portals,
	}
}

type lvl2TriangleRequest struct {
	p0, p1 portalData
	third  []portalIndex
}

func lvl2TriangleWorker(
	q *lvl2TriangleQuery,
	requestChannel, responseChannel chan lvl2TriangleRequest,
	wg *sync.WaitGroup) {
	normalizedVector := func(b0, b1 s2.Point) r3.Vector {
		// Let's care about memory consumption and not precompute
		// the norms for each portal pair.
		return b1.Cross(b0.Vector).Normalize()
	}

	portalsLeftOfLine := []homogeneousPureNode{}
	for req := range requestChannel {
		req.third = req.third[:0]
		p0, p1 := req.p0, req.p1
		portalsLeftOfLine = portalsLeftOfLine[:0]
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
		for k, node := range portalsLeftOfLine {
			// Emit each triangle only once to make sure we have consistent data,
			// even in the face of duplicate or colinear portals.
			if node.index <= p0.Index || node.index <= p1.Index {
				continue
			}
			numPortalsInTriangle := 0
			for j := 0; j < k; j++ {
				if portalsLeftOfLine[j].start <= node.start && portalsLeftOfLine[j].end <= node.end {
					numPortalsInTriangle++
					if numPortalsInTriangle > 1 {
						break
					}
				}
			}
			if numPortalsInTriangle == 1 {
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
	p0, p1    portalIndex
	triangles []triangle
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
			for _, third1 := range triangles[revEdgeIndex] {
				// Emit each triangle only once to make sure we have consistent
				// data, even in the face of duplicate or colinear portals.
				// So emit triangle only if triangle1.third is the largest of
				// its vertices.
				if third1 <= p0 || third1 <= third0 ||
					!s2.Sign(portals[p0].LatLng, portals[third0].LatLng, portals[third1].LatLng) {
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

func findAllLvl2Triangles(portals []portalData, params homogeneousParams) ([][]portalIndex, []edge) {
	resultCache := sync.Pool{
		New: func() interface{} {
			return []portalIndex{}
		},
	}

	requestChannel := make(chan lvl2TriangleRequest, params.numWorkers)
	responseChannel := make(chan lvl2TriangleRequest, params.numWorkers)
	var wg sync.WaitGroup
	wg.Add(params.numWorkers)
	q := newLvl2TriangleQuery(portals)
	for i := 0; i < params.numWorkers; i++ {
		go lvl2TriangleWorker(q, requestChannel, responseChannel, &wg)
	}
	go func() {
		for i, p0 := range portals {
			for _, p1 := range portals[i+1:] {
				requestChannel <- lvl2TriangleRequest{
					p0:    p0,
					p1:    p1,
					third: resultCache.Get().([]portalIndex),
				}
				requestChannel <- lvl2TriangleRequest{
					p0:    p1,
					p1:    p0,
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
	lvl2Triangles := make([][]portalIndex, len(portals)*len(portals))
	lvl2Edges := []edge{}

	numPairs := len(portals) * (len(portals) - 1)
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}

	params.progressFunc(0, numPairs)
	numProcessedPairs := 0
	numProcessedPairsModN := 0

	numPortals := uint32(len(portals))
	numLvl2Triangles := 0
	for resp := range responseChannel {
		if len(resp.third) > 0 {
			edge0Index := uint32(resp.p0.Index)*numPortals + uint32(resp.p1.Index)
			if len(lvl2Triangles[edge0Index]) == 0 {
				lvl2Edges = append(lvl2Edges, edge{resp.p0.Index, resp.p1.Index})
			}
			lvl2Triangles[edge0Index] = append(lvl2Triangles[edge0Index], resp.third...)
			for _, third := range resp.third {
				edge1Index := uint32(resp.p1.Index)*numPortals + uint32(third)
				if len(lvl2Triangles[edge1Index]) == 0 {
					lvl2Edges = append(lvl2Edges, edge{resp.p1.Index, third})
				}
				lvl2Triangles[edge1Index] = append(lvl2Triangles[edge1Index], resp.p0.Index)
				edge2Index := uint32(third)*numPortals + uint32(resp.p0.Index)
				if len(lvl2Triangles[edge2Index]) == 0 {
					lvl2Edges = append(lvl2Edges, edge{third, resp.p0.Index})
				}
				lvl2Triangles[edge2Index] = append(lvl2Triangles[edge2Index], resp.p1.Index)
			}
			numLvl2Triangles += len(resp.third)
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

	fmt.Println()
	fmt.Println("Num portals", numPortals)
	fmt.Println("Num lvl 2 triangles", numLvl2Triangles, len(lvl2Edges))

	return lvl2Triangles, lvl2Edges
}

func deepestPureHomogeneous(portals []portalData, params homogeneousParams) ([]portalIndex, int) {
	prevTriangles, prevEdges := findAllLvl2Triangles(portals, params)

	resultCache := sync.Pool{
		New: func() interface{} {
			return make([]triangle, 0, len(portals))
		},
	}

	bestDepth := 2
	for depth := 3; depth < params.maxDepth; depth++ {
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
				requestChannel <- mergeTrianglesRequest{
					p0:        commonEdge.p1,
					p1:        commonEdge.p0,
					triangles: resultCache.Get().([]triangle),
				}
			}
			close(requestChannel)
		}()
		go func() {
			wg.Wait()
			close(responseChannel)
		}()

		numEdges := len(prevEdges) * 2
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
		fmt.Println("\nNum lvl", depth, "triangles", newTriangles, len(lvlNEdges))
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
		for _, p2 := range edgeTriangles {
			if !hasAllIndicesInTheTriple(params.fixedCornerIndices, p0, p1, int(p2)) {
				continue
			}
			score := params.topLevelScorer.scoreTriangle(
				portals[p0], portals[p1], portals[p2])
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

	var triangleVertices func(p0, p1, p2 int, depth int) []portalIndex
	triangleVertices = func(p0, p1, p2 int, depth int) []portalIndex {
		center := findHomogeneousCenterPortal(p0, p1, p2, portals)
		if center < 0 {
			panic(center)
		}
		result := []portalIndex{portalIndex(center)}
		if depth == 2 {
			return result
		}
		result = append(result, triangleVertices(center, p1, p2, depth-1)...)
		result = append(result, triangleVertices(p0, center, p2, depth-1)...)
		result = append(result, triangleVertices(p0, p1, center, depth-1)...)
		return result

	}
	return append([]portalIndex{portalIndex(bestP0), portalIndex(bestP1), portalIndex(bestP2)},
		triangleVertices(bestP0, bestP1, bestP2, bestDepth)...), bestDepth
}

// Assuming p0, p1, p2 are corners of a pure homogeneous field, find its center portal.
// Return -1 if no suitable center portal found.
func findHomogeneousCenterPortal(p0, p1, p2 int, portals []portalData) int {
	q := newTriangleQuery(portals[p0].LatLng, portals[p1].LatLng, portals[p2].LatLng)
	portalsInTriangle := []portalData{}
	for i := 0; i < len(portals); i++ {
		if i == p0 || i == p1 || i == p2 {
			continue
		}
		if q.ContainsPoint(portals[i].LatLng) {
			portalsInTriangle = append(portalsInTriangle, portals[i])
		}
	}
	for candidate := 0; candidate < len(portalsInTriangle); candidate++ {
		q0 := newTriangleQuery(portals[p0].LatLng, portals[p1].LatLng, portalsInTriangle[candidate].LatLng)
		q1 := newTriangleQuery(portals[p1].LatLng, portals[p2].LatLng, portalsInTriangle[candidate].LatLng)
		q2 := newTriangleQuery(portals[p2].LatLng, portals[p0].LatLng, portalsInTriangle[candidate].LatLng)
		c0, c1, c2 := 0, 0, 0
		for i, p := range portalsInTriangle {
			if i == candidate {
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
			return int(portalsInTriangle[candidate].Index)
		}
	}
	return -1
}
