package lib

import "sort"
import "github.com/golang/geo/s1"
import "github.com/golang/geo/s2"

// LargestHerringbone - Find largest possible multilayer of portals to be made
func LargestHerringbone(portals []Portal, fixedBaseIndices []int, numWorkers int, progressFunc func(int, int)) (Portal, Portal, []Portal) {
	if numWorkers == 1 {
		return LargestHerringboneST(portals, fixedBaseIndices, progressFunc)
	}
	return LargestHerringboneMT(portals, fixedBaseIndices, numWorkers, progressFunc)
}

type node struct {
	index      portalIndex
	start, end s1.Angle
	distance   s1.ChordAngle
	length     uint16
	next       portalIndex
}

type bestHerringboneQuery struct {
	portals []portalData
	nodes   []node
	weights []float32
}

func newBestHerringboneQuery(portals []portalData) *bestHerringboneQuery {
	return &bestHerringboneQuery{
		portals: portals,
		nodes:   make([]node, 0, len(portals)),
		weights: make([]float32, len(portals)),
	}
}

func (q *bestHerringboneQuery) findBestHerringbone(b0, b1 portalData, result []portalIndex) []portalIndex {
	q.nodes = q.nodes[:0]
	aq0, aq1 := NewAngleQuery(b0.LatLng, b1.LatLng), NewAngleQuery(b1.LatLng, b0.LatLng)
	distQuery := newDistanceQuery(b0.LatLng, b1.LatLng)
	for _, portal := range q.portals {
		if portal == b0 || portal == b1 {
			continue
		}
		if !s2.Sign(portal.LatLng, b0.LatLng, b1.LatLng) {
			continue
		}
		a0, a1 := aq0.Angle(portal.LatLng), aq1.Angle(portal.LatLng)
		dist := distQuery.ChordAngle(portal.LatLng)
		q.nodes = append(q.nodes, node{portal.Index, a0, a1, dist, 0, invalidPortalIndex})
	}
	sort.Slice(q.nodes, func(i, j int) bool {
		return q.nodes[i].distance < q.nodes[j].distance
	})
	for i := 0; i < len(q.weights); i++ {
		q.weights[i] = 0
	}
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
	var weight float32
	for i, node := range q.nodes {
		if node.length > length || (node.length == length && q.weights[node.index] < weight) {
			length = node.length
			start = portalIndex(i)
			weight = q.weights[node.index]
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

// LargestHerringboneST - Find largest possible multilayer of portals to be made, using a single thread
func LargestHerringboneST(portals []Portal, fixedBaseIndices []int, progressFunc func(int, int)) (Portal, Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := portalsToPortalData(portals)

	var largestHerringbone []portalIndex
	var bestB0, bestB1 portalData
	resultCache := make([]portalIndex, 0, len(portals))

	numPairs := len(portals) * (len(portals) - 1) / 2
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	numProcessedPairs := 0
	numProcessedPairsModN := 0
	progressFunc(0, numPairs)
	q := newBestHerringboneQuery(portalsData)
	for i, b0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			b1 := portalsData[j]
			if !hasAllIndicesInThePair(fixedBaseIndices, i, j) {
				continue
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
			numProcessedPairsModN++
			if numProcessedPairsModN == everyNth {
				numProcessedPairsModN = 0
				progressFunc(numProcessedPairs, numPairs)
			}
		}
	}
	progressFunc(numPairs, numPairs)
	result := make([]Portal, 0, len(largestHerringbone))
	for _, portalIx := range largestHerringbone {
		result = append(result, portals[portalIx])
	}
	return portals[bestB0.Index], portals[bestB1.Index], result
}
