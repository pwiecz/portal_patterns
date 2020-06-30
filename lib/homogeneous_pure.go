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

	left, right := 0, len(nodes)-1

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

type perfByDistance []perfNode

func (d perfByDistance) Len() int           { return len(d) }
func (d perfByDistance) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d perfByDistance) Less(i, j int) bool { return d[i].distance < d[j].distance }

type lvl2TriangleQuery struct {
	portals []portalData
	norms   []r3.Vector
}

func newLvl2TriangleQuery(portals []portalData) *lvl2TriangleQuery {
	return &lvl2TriangleQuery{
		portals: portals,
	}
}

type lvl2TriangleRequest struct {
	p0, p1    portalData
	triangles []triangle
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

	portalsLeftOfLine := []perfNode{}
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
			portalsInTriangle := uint16(0)
			centerIndex := invalidPortalIndex
			for j := 0; j < k; j++ {
				if portalsLeftOfLine[j].start <= node.start && portalsLeftOfLine[j].end <= node.end {
					portalsInTriangle++
					centerIndex = portalsLeftOfLine[j].index
					if portalsInTriangle > 1 {
						break
					}
				}
			}
			// Emit each triangle only once to make sure we have consistent data,
			// even in the face of duplicate or colinear portals.
			if portalsInTriangle == 1 && node.index > p0.Index && node.index > p1.Index {
				req.triangles = append(req.triangles, triangle{third: node.index, center: centerIndex})
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
	for req := range requestChannel {
		req.trianglesAndEdges = req.trianglesAndEdges[:0]
		p0, p1 := req.p0, req.p1
		edgeIndex := int(p0)*len(portals) + int(p1)
		revEdgeIndex := int(p1)*len(portals) + int(p0)
		for _, triangle0 := range triangles[edgeIndex] {
			for _, triangle1 := range triangles[revEdgeIndex] {
				// Emit each triangle only once to make sure we have consistent
				// data, even in the face of duplicate or colinear portals.
				// So emit triangle only if triangle1.third is the largest of
				// its vertices.
				if triangle1.third <= p0 || triangle1.third <= triangle0.third ||
					!s2.Sign(portals[p0].LatLng,
						portals[triangle0.third].LatLng,
						portals[triangle1.third].LatLng) {
					continue
				}
				thirdEdgeIndex := int(triangle0.third)*len(portals) + int(triangle1.third)
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

func findAllLvl2Triangles(portals []portalData, params homogeneousParams) ([][]triangle, []edge) {
	resultCache := sync.Pool{
		New: func() interface{} {
			return []triangle{}
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
					p0:        p0,
					p1:        p1,
					triangles: resultCache.Get().([]triangle),
				}
				requestChannel <- lvl2TriangleRequest{
					p0:        p1,
					p1:        p0,
					triangles: resultCache.Get().([]triangle),
				}
			}
		}
		close(requestChannel)
	}()
	go func() {
		wg.Wait()
		close(responseChannel)
	}()
	lvl2Triangles := make([][]triangle, len(portals)*len(portals))
	lvl2Edges := []edge{}

	numPairs := len(portals) * (len(portals) - 1)
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}

	params.progressFunc(0, numPairs)
	numProcessedPairs := 0

	numLvl2Triangles := 0
	for resp := range responseChannel {
		if len(resp.triangles) > 0 {
			edge0Index := int(resp.p0.Index)*len(portals) + int(resp.p1.Index)
			if len(lvl2Triangles[edge0Index]) == 0 {
				lvl2Edges = append(lvl2Edges, edge{resp.p0.Index, resp.p1.Index})
			}
			lvl2Triangles[edge0Index] = append(lvl2Triangles[edge0Index], resp.triangles...)
			for _, t := range resp.triangles {
				edge1Index := int(resp.p1.Index)*len(portals) + int(t.third)
				if len(lvl2Triangles[edge1Index]) == 0 {
					lvl2Edges = append(lvl2Edges, edge{resp.p1.Index, t.third})
				}
				lvl2Triangles[edge1Index] = append(lvl2Triangles[edge1Index],
					triangle{third: resp.p0.Index, center: t.center})
				edge2Index := int(t.third)*len(portals) + int(resp.p0.Index)
				if len(lvl2Triangles[edge2Index]) == 0 {
					lvl2Edges = append(lvl2Edges, edge{t.third, resp.p0.Index})
				}
				lvl2Triangles[edge2Index] = append(lvl2Triangles[edge2Index],
					triangle{third: resp.p1.Index, center: t.center})
			}
			numLvl2Triangles += len(resp.triangles)
		}
		resultCache.Put(resp.triangles[:0])
		numProcessedPairs++
		if numProcessedPairs%everyNth == 0 {
			params.progressFunc(numProcessedPairs, numPairs)
		}
	}
	params.progressFunc(numPairs, numPairs)

	fmt.Println()
	fmt.Println("Num portals", len(portals))
	fmt.Println("Num lvl 2 triangles", numLvl2Triangles, len(lvl2Edges))

	return lvl2Triangles, lvl2Edges
}

func deepestPureHomogeneous(portals []portalData, params homogeneousParams) ([]portalIndex, int) {
	if params.numWorkers <= 0 {
		params.numWorkers = runtime.GOMAXPROCS(0)
	}

	lvl2Triangles, lvlEdges := findAllLvl2Triangles(portals, params)

	lvlTriangles := [][][]triangle{lvl2Triangles}

	resultCache := sync.Pool{
		New: func() interface{} {
			return make([]triangleAndEdge, 0, len(portals))
		},
	}

	for depth := 3; depth < params.maxDepth; depth++ {
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
				requestChannel <- mergeTrianglesRequest{
					p0:                commonEdge.p1,
					p1:                commonEdge.p0,
					trianglesAndEdges: resultCache.Get().([]triangleAndEdge),
				}
			}
			close(requestChannel)
		}()
		go func() {
			wg.Wait()
			close(responseChannel)
		}()

		numEdges := len(lvlEdges) * 2
		everyNth := numEdges / 1000
		if everyNth < 50 {
			everyNth = 2
		}

		params.progressFunc(0, numEdges)
		numProcessedEdges := 0

		lvlNTriangles := make([][]triangle, len(portals)*len(portals))
		lvlNEdges := []edge{}

		for resp := range responseChannel {
			for _, triangleAndEdge := range resp.trianglesAndEdges {
				newTriangles++
				edgeIndex := int(triangleAndEdge.edge.p0)*len(portals) + int(triangleAndEdge.edge.p1)
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
