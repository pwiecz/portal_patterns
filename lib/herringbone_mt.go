package lib

import "sort"
import "sync"
import "github.com/golang/geo/r2"

type bestHerringboneMtQuery struct {
	portals []portalData
	// Array of normalized direction vectors between all the pairs of portals
	norms []r2.Point
}

func newBestHerringboneMtQuery(portals []portalData) *bestHerringboneMtQuery {
	norms := make([]r2.Point, len(portals)*len(portals))
	for i, p0 := range portals {
		for j, p1 := range portals {
			if i == j {
				continue
			}
			dp := p1.LatLng.Sub(p0.LatLng)
			dpLen := dp.Norm()
			dp.X /= dpLen
			dp.Y /= dpLen
			norms[i*len(portals)+j] = dp

		}
	}
	return &bestHerringboneMtQuery{
		portals: portals,
		norms:   norms,
	}
}

type herringboneRequest struct {
	p0, p1 portalData
	result []portalIndex
}

func (q *bestHerringboneMtQuery) normalizedVector(b0, b1 portalData) r2.Point {
	return q.norms[uint(b0.Index)*uint(len(q.portals))+uint(b1.Index)]
}

func (q *bestHerringboneMtQuery) findBestHerringbone(b0, b1 portalData, nodes []node, weights []float32, result []portalIndex) []portalIndex {
	nodes = nodes[:0]
	distQuery := NewDistanceQuery(b0.LatLng, b1.LatLng)
	b01, b10 := q.normalizedVector(b0, b1), q.normalizedVector(b1, b0)
	for _, portal := range q.portals {
		if portal == b0 || portal == b1 {
			continue
		}
		if Sign(portal.LatLng, b0.LatLng, b1.LatLng) <= 0 {
			continue
		}
		a0 := b01.Dot(q.normalizedVector(b1, portal)) // acos of angle b0,b1,portal
		a1 := b10.Dot(q.normalizedVector(b0, portal)) // acos of angle b1,b0,portal
		dist := distQuery.DistanceSq(portal.LatLng)
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
					scaledDistance := float32(Distance(q.portals[node.index].LatLng, q.portals[nodes[j].index].LatLng) * radiansToMeters)
					bestWeight = weights[nodes[j].index] + scaledDistance
				} else if nodes[j].length+1 == bestLength {
					scaledDistance := float32(Distance(q.portals[node.index].LatLng, q.portals[nodes[j].index].LatLng) * radiansToMeters)
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
				Distance(q.portals[node.index].LatLng, b0.LatLng),
				Distance(q.portals[node.index].LatLng, b1.LatLng)) * radiansToMeters)
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
