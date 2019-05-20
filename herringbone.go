package main

import "fmt"

import "sort"
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

type bestHerringBoneQuery struct {
	portals []portalData
	nodes   []node
	weights []float32
}

func newBestHerringBoneQuery(portals []portalData) *bestHerringBoneQuery {
	return &bestHerringBoneQuery{
		portals: portals,
		nodes:   make([]node, 0, len(portals)),
	}
}

func (q *bestHerringBoneQuery) findBestHerringbone(b0, b1 portalData, result []portalIndex) []portalIndex {
	q.nodes = q.nodes[:0]
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
		q.nodes = append(q.nodes, node{portal.Index, a0, a1, dist, 0, invalidPortalIndex})
	}
	sort.Slice(q.nodes, func(i, j int) bool {
		return q.nodes[i].distance < q.nodes[j].distance
	})
	q.weights = make([]float32, len(q.portals), len(q.portals))
	for i, node := range q.nodes {
		var bestLength uint16
		bestNext := invalidPortalIndex
		var bestWeight float32
		for j := 0; j < i; j++ {
			if q.nodes[j].start < node.start && q.nodes[j].end < node.end {
				if q.nodes[j].length >= bestLength {
					bestLength = q.nodes[j].length + 1
					bestNext = portalIndex(j)
					scaledDistance := float32(distance(q.portals[node.index], q.portals[q.nodes[j].index]) * radiansToMeters)
					bestWeight = q.weights[q.nodes[j].index] + scaledDistance
				} else if q.nodes[j].length+1 == bestLength {
					scaledDistance := float32(distance(q.portals[node.index], q.portals[q.nodes[j].index]) * radiansToMeters)
					if q.weights[node.index]+scaledDistance < bestWeight {
						bestLength = q.nodes[j].length + 1
						bestNext = portalIndex(j)
						bestWeight = q.weights[q.nodes[j].index] + scaledDistance
					}
				}
			}
		}
		q.nodes[i].length = bestLength
		q.nodes[i].next = bestNext
		if bestLength > 0 {
			q.weights[node.index] = bestWeight
		} else {
			q.weights[node.index] = float32(float64Min(
				distance(q.portals[node.index], b0),
				distance(q.portals[node.index], b1)) * radiansToMeters)
		}
	}

	start := invalidPortalIndex
	var length uint16
	for i, node := range q.nodes {
		if node.length > length {
			length = node.length
			start = portalIndex(i)
		}
	}
	result = result[:0]
	if start == invalidPortalIndex {
		return result
	}
	for start != invalidPortalIndex {
		result = append(result, q.nodes[start].index)
		start = q.nodes[start].next
	}
	return result
}

// LargestHerringbone - Find largest possible multilayer of portals to be made
func LargestHerringbone(portals []Portal) (Portal, Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{Index: portalIndex(i), LatLng: portal.LatLng})
	}

	index := make([]bestSolution, len(portals))
	var largestHerringbone []portalIndex
	var bestB0, bestB1 portalData
	resultCache := make([]portalIndex, 0, len(portals))

	numPairs := len(portals) * (len(portals) - 1) / 2
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	numProcessedPairs := 0
	printProgressBar(0, numPairs)
	q := newBestHerringBoneQuery(portalsData)
	for i, b0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			b1 := portalsData[j]
			for k := 0; k < len(index); k++ {
				index[k].Length = invalidLength
			}
			bestCCW := q.findBestHerringbone(b0, b1, resultCache)
			if len(bestCCW) > len(largestHerringbone) {
				largestHerringbone = append(largestHerringbone[:0], bestCCW...)
				bestB0 = b0
				bestB1 = b1
			}
			bestCW := q.findBestHerringbone(b1, b0, resultCache)
			if len(bestCW) > len(largestHerringbone) {
				largestHerringbone = append(largestHerringbone[:0], bestCW...)
				bestB0 = b1
				bestB1 = b0
			}
			numProcessedPairs++
			if numProcessedPairs%everyNth == 0 {
				printProgressBar(numProcessedPairs, numPairs)
			}
		}
	}
	printProgressBar(numPairs, numPairs)
	fmt.Println("")
	result := make([]Portal, 0, len(largestHerringbone))
	for _, portalIx := range largestHerringbone {
		result = append(result, portals[portalIx])
	}
	return portals[bestB0.Index], portals[bestB1.Index], result
}
