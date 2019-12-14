package lib

import "sort"
import "github.com/golang/geo/r2"

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

type bestHerringboneQuery struct {
	portals []portalData
	// Array of normalized direction vectors between all the pairs of portals
	norms   []r2.Point
	nodes   []node
	weights []float32
}

func newBestHerringboneQuery(portals []portalData) *bestHerringboneQuery {
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
	return &bestHerringboneQuery{
		portals: portals,
		norms:   norms,
		nodes:   make([]node, 0, len(portals)),
		weights: make([]float32, len(portals)),
	}
}

type byDistance []node

func (d byDistance) Len() int           { return len(d) }
func (d byDistance) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d byDistance) Less(i, j int) bool { return d[i].distance < d[j].distance }

func (q *bestHerringboneQuery) normalizedVector(b0, b1 portalData) r2.Point {
	return q.norms[uint(b0.Index)*uint(len(q.portals))+uint(b1.Index)]
}

func (q *bestHerringboneQuery) findBestHerringbone(b0, b1 portalData, result []portalIndex) []portalIndex {
	q.nodes = q.nodes[:0]
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
		q.nodes = append(q.nodes, node{portal.Index, a0, a1, dist, 0, invalidPortalIndex})
	}
	sort.Sort(byDistance(q.nodes))
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
					scaledDistance := float32(Distance(q.portals[node.index].LatLng, q.portals[q.nodes[j].index].LatLng) * radiansToMeters)
					bestWeight = q.weights[q.nodes[j].index] + scaledDistance
				} else if q.nodes[j].length+1 == bestLength {
					scaledDistance := float32(Distance(q.portals[node.index].LatLng, q.portals[q.nodes[j].index].LatLng) * radiansToMeters)
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
				Distance(q.portals[node.index].LatLng, b0.LatLng),
				Distance(q.portals[node.index].LatLng, b1.LatLng)) * radiansToMeters)
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
