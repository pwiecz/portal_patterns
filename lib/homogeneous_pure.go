package lib

import "fmt"
import "math"
import "runtime"
import "sync"
import "github.com/golang/geo/r3"
import "github.com/golang/geo/s1"
import "github.com/golang/geo/s2"

type triangle struct {
	third  portalIndex
	center portalIndex
}
type edge struct {
	p0, p1 portalIndex
}

type perfNode struct {
	index      portalIndex
	start, end float64
	distance   s1.ChordAngle
}

// specialize a simple sorting function, to limit overhead of a std sort package.
func sortPerfByDistance(nodes []perfNode) {
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
	sortPerfByDistance(nodes[:left])
	sortPerfByDistance(nodes[left+1:])
}

type lvlNTriangleQuery struct {
	portals                      []portalData
	expectedNumPortalsInTriangle int
}

func newLvlNTriangleQuery(portals []portalData, level int) *lvlNTriangleQuery {
	expectedNumPortalsInTriangle := 0
	for i := 1; i < level; i++ {
		expectedNumPortalsInTriangle = expectedNumPortalsInTriangle*3 + 1
	}
	return &lvlNTriangleQuery{
		portals:                      portals,
		expectedNumPortalsInTriangle: expectedNumPortalsInTriangle,
	}
}

type lvlNTriangleRequest struct {
	p0, p1 portalData
	third  []portalIndex
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
	portalsInTriangle := []portalIndex{}
	for req := range requestChannel {
		req.triangles = req.triangles[:0]
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
			portalsLeftOfLine = append(portalsLeftOfLine, perfNode{
				p2.Index, a0, a1, dist})
		}
		sortPerfByDistance(portalsLeftOfLine)
		for k, node := range portalsLeftOfLine {
			// Emit each triangle only once to make sure we have consistent data,
			// even in the face of duplicate or colinear portals.
			// So emit triangle only if p0 is the smallest of its vertices.
			if node.index <= p0.Index {
				continue
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

type triangleAndEdge struct {
	triangle triangle
	edge     edge
}
type mergeTrianglesRequest struct {
	p0, p1            portalIndex
	trianglesAndEdges []triangleAndEdge
}

func mergeTrianglesWorker(
	portals []portalData,
	triangles [][]triangle,
	requestChannel, responseChannel chan mergeTrianglesRequest,
	wg *sync.WaitGroup) {
	numPortals := uint(len(portals))
	for req := range requestChannel {
		req.trianglesAndEdges = req.trianglesAndEdges[:0]
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
				thirdEdgeIndex := uint(triangle0.third)*numPortals + uint(triangle1.third)
				for _, triangle2 := range triangles[thirdEdgeIndex] {
					if triangle2.third == p0 {
						req.trianglesAndEdges = append(req.trianglesAndEdges,
							triangleAndEdge{
								triangle{
									third:  triangle1.third,
									center: p0,
								},
								edge{p1, triangle0.third},
							},
							triangleAndEdge{
								triangle{
									third:  triangle0.third,
									center: p0,
								},
								edge{triangle1.third, p1},
							},
							triangleAndEdge{
								triangle{
									third: p1, center: p0,
								},
								edge{triangle0.third, triangle1.third},
							})
					}
				}
			}
		}
		responseChannel <- req
	}
	wg.Done()
}

func findAllLvlNTriangles(portals []portalData, params homogeneousParams, level int) ([][]portalIndex, []edge) {
	resultCache := sync.Pool{
		New: func() interface{} {
			return []triangle{}
		},
	}

	requestChannel := make(chan lvlNTriangleRequest, params.numWorkers)
	responseChannel := make(chan lvlNTriangleRequest, params.numWorkers)
	var wg sync.WaitGroup
	wg.Add(params.numWorkers)
	q := newLvlNTriangleQuery(portals, level)
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
		resultCache.Put(resp.triangles[:0])
		numProcessedPairs++
		if numProcessedPairs%everyNth == 0 {
			params.progressFunc(numProcessedPairs, numPairs)
		}
	}
	params.progressFunc(numPairs, numPairs)

	fmt.Println()
	fmt.Println("Num portals", numPortals)
	fmt.Println("Num lvl", level, "triangles", numLvlNTriangles, len(lvlNEdges))

	return lvlNTriangles, lvlNEdges
}

func deepestPureHomogeneous(portals []portalData, params homogeneousParams) ([]portalIndex, int) {
	initialLevel := 4
	prevTriangles, prevEdges := findAllLvlNTriangles(portals, params, initialLevel)

	resultCache := sync.Pool{
		New: func() interface{} {p
			return make([]triangleAndEdge, 0, len(portals))
		},
	}

	bestDepth := initialLevel
	for depth := initialLevel + 1; depth < params.maxDepth; depth++ {
		requestChannel := make(chan mergeTrianglesRequest, params.numWorkers)
		responseChannel := make(chan mergeTrianglesRequest, params.numWorkers)
		var wg sync.WaitGroup
		wg.Add(params.numWorkers)
		for i := 0; i < params.numWorkers; i++ {
			go mergeTrianglesWorker(portals, lvlTriangles[depth-3], requestChannel, responseChannel, &wg)
		}

		newTriangles := 0

		go func() {
			for _, commonEdge := range lvlEdges {
				requestChannel <- mergeTrianglesRequest{
					p0:                commonEdge.p0,
					p1:                commonEdge.p1,
					trianglesAndEdges: resultCache.Get().([]triangleAndEdge),
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

		lvlNTriangles := make([][]triangle, len(portals)*len(portals))
		lvlNEdges := []edge{}
		numPortals := uint(len(portals))

		for resp := range responseChannel {
			for _, triangleAndEdge := range resp.trianglesAndEdges {
				newTriangles++
				edgeIndex := uint(triangleAndEdge.edge.p0)*numPortals + uint(triangleAndEdge.edge.p1)
				lvlNTriangles[edgeIndex] = append(lvlNTriangles[edgeIndex], triangleAndEdge.triangle)
				if len(lvlNTriangles[edgeIndex]) == 1 {
					lvlNEdges = append(lvlNEdges, triangleAndEdge.edge)
				}
			}
			resultCache.Put(resp.trianglesAndEdges)

			numProcessedEdges++
			if numProcessedEdges%everyNth == 0 {
				params.progressFunc(numProcessedEdges, numEdges)
			}
		}
		params.progressFunc(numEdges, numEdges)

		if len(lvlNEdges) == 0 {
			break
		}
		fmt.Println("\nNum lvl", depth, "triangles", newTriangles, len(lvlNEdges))
		lvlTriangles = append(lvlTriangles, lvlNTriangles)
		lvlEdges = lvlNEdges
	}

	var bestTriangle triangle
	var bestP0, bestP1 int
	bestDepth := len(lvlTriangles) + 1
	foundSolution := false
	bestTriangleScore := float32(-math.MaxFloat32)
	for edge, edgeTriangles := range lvlTriangles[len(lvlTriangles)-1] {
		if len(edgeTriangles) == 0 {
			continue
		}
		p0 := edge / len(portals)
		p1 := edge % len(portals)
		for _, t := range edgeTriangles {
			if !hasAllIndicesInTheTriple(params.fixedCornerIndices, p0, p1, int(t.third)) {
				continue
			}
			score := params.topLevelScorer.scoreTriangle(
				portals[p0], portals[p1], portals[t.third])
			if score > bestTriangleScore {
				bestTriangleScore = score
				bestTriangle = t
				bestP0, bestP1 = p0, p1
				foundSolution = true
			}
		}
	}

	if !foundSolution {
		return []portalIndex{}, 0
	}

	var triangleVertices func(p0, p1 int, t triangle, depth int, lvlTriangles [][][]triangle) []portalIndex
	triangleVertices = func(p0, p1 int, t triangle, depth int, lvlTriangles [][][]triangle) []portalIndex {
		result := []portalIndex{t.center}
		if depth == 2 {
			return result
		}
		revEdge0Index := int(t.center)*len(portals) + p1
		found1 := false
		for _, triangle := range lvlTriangles[depth-3][revEdge0Index] {
			if triangle.third == t.third {
				result = append(result, triangleVertices(int(t.center), p1, triangle, depth-1, lvlTriangles)...)
				found1 = true
				break
			}
		}
		if !found1 {
			panic(fmt.Errorf("Could not find triangle %d,%d,%d at depth %d", t.center, p1, t.third, depth))
		}
		edge0Index := p0*len(portals) + int(t.center)
		found0 := false
		for _, triangle := range lvlTriangles[depth-3][edge0Index] {
			if triangle.third == t.third {
				result = append(result, triangleVertices(p0, int(t.center), triangle, depth-1, lvlTriangles)...)
				found0 = true
				break
			}
		}
		if !found0 {
			panic(fmt.Errorf("Could not find triangle %d,%d,%d at depth %d", p0, t.center, t.third, depth))
		}
		edge2Index := p0*len(portals) + p1
		found2 := false
		for _, triangle := range lvlTriangles[depth-3][edge2Index] {
			if triangle.third == t.center {
				result = append(result, triangleVertices(p0, p1, triangle, depth-1, lvlTriangles)...)
				found2 = true
				break
			}
		}
		if !found2 {
			panic(fmt.Errorf("Could not find triangle %d,%d,%d at depth %d", p0, p1, t.center, depth))
		}
		return result

	}
	return append([]portalIndex{portalIndex(bestP0), portalIndex(bestP1), bestTriangle.third},
		triangleVertices(bestP0, bestP1, bestTriangle, bestDepth, lvlTriangles)...), bestDepth
}

func areValidPureHomogeneousPortals(p0, p1, p2 portalIndex, inside []portalIndex, portals []portalData) bool {
	if len(inside) == 1 {
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
