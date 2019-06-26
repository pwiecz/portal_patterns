package main

import "fmt"

import "math"
import "sort"
import "sync"
import "github.com/golang/geo/s1"
import "github.com/golang/geo/s2"
import "github.com/golang/geo/r3"

type node struct {
	index      portalIndex
	start, end s1.Angle
	distance   s1.ChordAngle
	length     uint16
	next       portalIndex
}

func angle(a, b s2.Point, v r3.Vector) s1.Angle {
	return a.PointCross(b).Angle(v)
}

type bestHerringboneQuery struct {
	portals []portalData
}

func newBestHerringboneQuery(portals []portalData) *bestHerringboneQuery {
	return &bestHerringboneQuery{
		portals: portals,
	}
}

type herringboneRequest struct {
	p0, p1 portalData
	result []portalIndex
}

func (q *bestHerringboneQuery) findBestHerringbone(b0, b1 portalData, nodes []node, weights []float32, result []portalIndex) []portalIndex {
	nodes = nodes[:0]
	v0, v1 := b1.LatLng.PointCross(b0.LatLng).Vector, b0.LatLng.PointCross(b1.LatLng).Vector
	distQuery := newDistanceQuery(b0.LatLng, b1.LatLng)
	for _, portal := range q.portals {
		if portal == b0 || portal == b1 {
			continue
		}
		if !s2.Sign(portal.LatLng, b0.LatLng, b1.LatLng) {
			continue
		}
		a0, a1 := angle(portal.LatLng, b0.LatLng, v0), angle(portal.LatLng, b1.LatLng, v1)
		dist := distQuery.ChordAngle(portal.LatLng)
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
					scaledDistance := float32(distance(q.portals[node.index], q.portals[nodes[j].index]) * radiansToMeters)
					bestWeight = weights[nodes[j].index] + scaledDistance
				} else if nodes[j].length+1 == bestLength {
					scaledDistance := float32(distance(q.portals[node.index], q.portals[nodes[j].index]) * radiansToMeters)
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
				distance(q.portals[node.index], b0),
				distance(q.portals[node.index], b1)) * radiansToMeters)
		}
	}

	start := invalidPortalIndex
	var length uint16
	weight := float32(-math.MaxFloat32)
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
	q *bestHerringboneQuery,
	requestChannel, responseChannel chan herringboneRequest,
	doneChannel chan struct{}) {
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
	doneChannel <- struct{}{}
}

// LargestHerringbone - Find largest possible multilayer of portals to be made
func LargestHerringbone(portals []Portal, numWorkers int) (Portal, Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{Index: portalIndex(i), LatLng: portal.LatLng})
	}

	resultCache := sync.Pool{
		New: func() interface{} {
			return make([]portalIndex, 0, len(portals))
		},
	}

	requestChannel := make(chan herringboneRequest, numWorkers)
	responseChannel := make(chan herringboneRequest, numWorkers)
	doneChannel := make(chan struct{}, numWorkers)
	q := newBestHerringboneQuery(portalsData)
	for i := 0; i < numWorkers; i++ {
		go bestHerringboneWorker(q, requestChannel, responseChannel, doneChannel)
	}
	go func() {
		for i, b0 := range portalsData {
			for j := i + 1; j < len(portalsData); j++ {
				b1 := portalsData[j]
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

	numPairs := len(portals) * (len(portals) - 1)
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	printProgressBar(0, numPairs)
	numProcessedPairs := 0

	var largestHerringbone []portalIndex
	var bestB0, bestB1 portalIndex
	numWorkersDone := 0
	for numWorkersDone < numWorkers {
		select {
		case resp := <-responseChannel:
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
				printProgressBar(numProcessedPairs, numPairs)
			}
		case <-doneChannel:
			numWorkersDone++
		}
	}
	printProgressBar(numPairs, numPairs)
	fmt.Println("")
	close(responseChannel)
	close(doneChannel)
	result := make([]Portal, 0, len(largestHerringbone))
	for _, portalIx := range largestHerringbone {
		result = append(result, portals[portalIx])
	}
	return portals[bestB0], portals[bestB1], result
}
