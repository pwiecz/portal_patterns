package lib

import "math"
import "sort"
import "github.com/golang/geo/r2"
import "github.com/pwiecz/portal_patterns/lib/r2geo"

// LargestHerringbone - Find largest possible multilayer of portals to be made
func LargestHerringbone(portals []Portal, fixedBaseIndices []int, numWorkers int, progressFunc func(int, int)) (Portal, Portal, []Portal) {
	if numWorkers == 1 {
		return LargestHerringboneST(portals, fixedBaseIndices, progressFunc)
	}
	return LargestHerringboneMT(portals, fixedBaseIndices, numWorkers, progressFunc)
}

type node struct {
	index      portalIndex
	start, end float64
	distance   float64
	length     uint16
	next       portalIndex
}

// Internal angle at b in triangle abc. Value must be in range [-Pi, Pi].
func angle(a, b, c r2.Point) float64 {
	ab := a.Sub(b)
	bc := c.Sub(b)
	angle := math.Acos(ab.Dot(bc) / (ab.Norm() * bc.Norm()))
	return angle
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
	distQuery := r2geo.NewDistanceQuery(b0.LatLng, b1.LatLng)
	for _, portal := range q.portals {
		if portal == b0 || portal == b1 {
			continue
		}
		if r2geo.Sign(portal.LatLng, b0.LatLng, b1.LatLng) <= 0 {
			continue
		}
		a0, a1 := angle(portal.LatLng, b0.LatLng, b1.LatLng), angle(portal.LatLng, b1.LatLng, b0.LatLng)
		dist := distQuery.DistanceSq(portal.LatLng)
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
					scaledDistance := float32(r2geo.Distance(q.portals[node.index].LatLng, q.portals[q.nodes[j].index].LatLng) * radiansToMeters)
					bestWeight = q.weights[q.nodes[j].index] + scaledDistance
				} else if q.nodes[j].length+1 == bestLength {
					scaledDistance := float32(r2geo.Distance(q.portals[node.index].LatLng, q.portals[q.nodes[j].index].LatLng) * radiansToMeters)
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
				r2geo.Distance(q.portals[node.index].LatLng, b0.LatLng),
				r2geo.Distance(q.portals[node.index].LatLng, b1.LatLng)) * radiansToMeters)
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
	numProcessedPairsModN := 0
	progressFunc(0, numPairs)
	q := newBestHerringboneQuery(portalsData)
	for i, b0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			b1 := portalsData[j]
			if !hasAllIndicesInThePair(fixedBaseIndices, i, j) {
				continue
			}
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
