package lib

import "sort"
import "sync"
//import "github.com/golang/geo/s2"
import "github.com/pwiecz/portal_patterns/lib/r2geo"

type bestHerringboneMtQuery struct {
	portals []portalData
}

func newBestHerringboneMtQuery(portals []portalData) *bestHerringboneMtQuery {
	return &bestHerringboneMtQuery{
		portals: portals,
	}
}

type herringboneRequest struct {
	p0, p1 portalData
	result []portalIndex
}

func (q *bestHerringboneMtQuery) findBestHerringbone(b0, b1 portalData, nodes []node, weights []float32, result []portalIndex) []portalIndex {
	nodes = nodes[:0]
	//v0, v1 := b1.LatLng.PointCross(b0.LatLng).Vector, b0.LatLng.PointCross(b1.LatLng).Vector
	distQuery := r2geo.NewDistanceQuery(b0.LatLng, b1.LatLng)
	for _, portal := range q.portals {
		if portal == b0 || portal == b1 {
			continue
		}
		if r2geo.Sign(portal.LatLng, b0.LatLng, b1.LatLng) <= 0 {
			continue
		}
		//a0, a1 := angle(portal.LatLng, b0.LatLng, v0), angle(portal.LatLng, b1.LatLng, v1)
		a0, a1 := angle(portal.LatLng, b0.LatLng, b1.LatLng), angle(portal.LatLng, b1.LatLng, b0.LatLng)
//		dist := distQuery.ChordAngle(portal.LatLng)
		dist := distQuery.Distance(portal.LatLng)
		nodes = append(nodes, node{portal.Index, a0, a1, dist, 0, invalidPortalIndex})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].distance < nodes[j].distance
	})
	for i := 0; i < len(weights); i++ {
		weights[i] = 0
	}
	for i, node := range nodes {
		var bestLength uint16
		bestNext := invalidPortalIndex
		var bestWeight float32
		for j := 0; j < i; j++ {
			if nodes[j].start < node.start && nodes[j].end < node.end {
				if nodes[j].length >= bestLength {
					bestLength = nodes[j].length + 1
					bestNext = portalIndex(j)
					scaledDistance := float32(r2geo.Distance(q.portals[node.index].LatLng, q.portals[nodes[j].index].LatLng) * radiansToMeters)
					bestWeight = weights[nodes[j].index] + scaledDistance
				} else if nodes[j].length+1 == bestLength {
					scaledDistance := float32(r2geo.Distance(q.portals[node.index].LatLng, q.portals[nodes[j].index].LatLng) * radiansToMeters)
					if weights[node.index]+scaledDistance < bestWeight {
						bestLength = nodes[j].length + 1
						bestNext = portalIndex(j)
						bestWeight = weights[nodes[j].index] + scaledDistance
					}
				}
			}
		}
		nodes[i].length = bestLength
		nodes[i].next = bestNext
		if bestLength > 0 {
			weights[node.index] = bestWeight
		} else {
			weights[node.index] = float32(float64Min(
				r2geo.Distance(q.portals[node.index].LatLng, b0.LatLng),
				r2geo.Distance(q.portals[node.index].LatLng, b1.LatLng)) * radiansToMeters)
		}
	}

	start := invalidPortalIndex
	var length uint16
	var weight float32
	for i, node := range nodes {
		if node.length > length || (node.length == length && weights[node.index] < weight) {
			length = node.length
			start = portalIndex(i)
			weight = weights[node.index]
		}
	}
	result = result[:0]
	if start == invalidPortalIndex {
		return result
	}
	for start != invalidPortalIndex {
		result = append(result, nodes[start].index)
		start = nodes[start].next
	}
	return result
}

func bestHerringboneWorker(
	q *bestHerringboneMtQuery,
	requestChannel, responseChannel chan herringboneRequest,
	wg *sync.WaitGroup) {
	nodes := make([]node, 0, len(q.portals))
	weights := make([]float32, len(q.portals))
	for req := range requestChannel {
		nodes = nodes[:0]
		for i := 0; i < len(weights); i++ {
			weights[i] = 0
		}
		req.result = q.findBestHerringbone(req.p0, req.p1, nodes, weights, req.result)
		responseChannel <- req
	}
	wg.Done()
}

// LargestHerringboneMT - Find largest possible multilayer of portals to be made, parallel version
func LargestHerringboneMT(portals []Portal, fixedBaseIndices []int, numWorkers int, progressFunc func(int, int)) (Portal, Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

	resultCache := sync.Pool{
		New: func() interface{} {
			return make([]portalIndex, 0, len(portals))
		},
	}

	requestChannel := make(chan herringboneRequest, numWorkers)
	responseChannel := make(chan herringboneRequest, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	q := newBestHerringboneMtQuery(portalsData)
	for i := 0; i < numWorkers; i++ {
		go bestHerringboneWorker(q, requestChannel, responseChannel, &wg)
	}
	go func() {
		for i, b0 := range portalsData {
			for j := i + 1; j < len(portalsData); j++ {
				b1 := portalsData[j]
				if !hasAllIndicesInThePair(fixedBaseIndices, i, j) {
					continue
				}
				requestChannel <- herringboneRequest{
					p0:     b0,
					p1:     b1,
					result: resultCache.Get().([]portalIndex),
				}
				requestChannel <- herringboneRequest{
					p0:     b1,
					p1:     b0,
					result: resultCache.Get().([]portalIndex),
				}
			}
		}
		close(requestChannel)
	}()
	go func() {
		wg.Wait()
		close(responseChannel)

	}()
	numPairs := len(portals) * (len(portals) - 1)
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	progressFunc(0, numPairs)
	numProcessedPairs := 0

	var largestHerringbone []portalIndex
	var bestB0, bestB1 portalIndex
	for resp := range responseChannel {
		if len(resp.result) > len(largestHerringbone) {
			if len(largestHerringbone) > 0 {
				resultCache.Put(largestHerringbone)
			}
			largestHerringbone = resp.result
			bestB0, bestB1 = resp.p0.Index, resp.p1.Index
		} else {
			resultCache.Put(resp.result)
		}
		numProcessedPairs++
		if numProcessedPairs%everyNth == 0 {
			progressFunc(numProcessedPairs, numPairs)
		}
	}
	progressFunc(numPairs, numPairs)
	result := make([]Portal, 0, len(largestHerringbone))
	for _, portalIx := range largestHerringbone {
		result = append(result, portals[portalIx])
	}
	return portals[bestB0], portals[bestB1], result
}
