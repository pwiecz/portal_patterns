package lib

import (
	"sort"

	"github.com/golang/geo/r3"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

// LargestHerringbone - Find largest possible multilayer of portals to be made
func LargestHerringbone(portals []Portal, fixedBaseIndices []int, numWorkers int, progressFunc func(int, int)) (Portal, Portal, []Portal) {
	if numWorkers == 1 {
		return LargestHerringboneST(portals, fixedBaseIndices, progressFunc)
	}
	return LargestHerringboneMT(portals, fixedBaseIndices, numWorkers, progressFunc)
}

type herringboneNode struct {
	start    float64
	end      float64
	distance s1.ChordAngle
	index    portalIndex
	length   uint16
	next     portalIndex
}

type bestHerringboneQuery struct {
	portals []portalData
	nodes   []herringboneNode
	weights []float32
	// Array of normalized direction vectors between all the pairs of portals
	norms []r3.Vector
}

func newBestHerringboneQuery(portals []portalData) *bestHerringboneQuery {
	norms := make([]r3.Vector, len(portals)*len(portals))
	for i, p0 := range portals {
		for j, p1 := range portals {
			if i == j {
				continue
			}
			norms[i*len(portals)+j] = p1.LatLng.Cross(p0.LatLng.Vector).Normalize()

		}
	}
	return &bestHerringboneQuery{
		portals: portals,
		nodes:   make([]herringboneNode, 0, len(portals)),
		weights: make([]float32, len(portals)),
		norms:   norms,
	}
}

type herringboneNodesByDistance []herringboneNode

func (d herringboneNodesByDistance) Len() int           { return len(d) }
func (d herringboneNodesByDistance) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d herringboneNodesByDistance) Less(i, j int) bool { return d[i].distance < d[j].distance }

func (q *bestHerringboneQuery) normalizedVector(b0, b1 portalData) r3.Vector {
	return q.norms[uint(b0.Index)*uint(len(q.portals))+uint(b1.Index)]
}
func (q *bestHerringboneQuery) findBestHerringbone(b0, b1 portalData, result []portalIndex) []portalIndex {
	q.nodes = q.nodes[:0]
	b01, b10 := q.normalizedVector(b0, b1), q.normalizedVector(b1, b0)
	distQuery := newDistanceQuery(b0.LatLng, b1.LatLng)
	for _, portal := range q.portals {
		if portal == b0 || portal == b1 {
			continue
		}
		if !s2.Sign(portal.LatLng, b0.LatLng, b1.LatLng) {
			continue
		}
		a0 := b01.Dot(q.normalizedVector(b1, portal)) // acos of angle b0,b1,portal
		a1 := b10.Dot(q.normalizedVector(b0, portal)) // acos of angle b1,b0,portal
		dist := distQuery.ChordAngle(portal.LatLng)
		q.nodes = append(q.nodes, herringboneNode{
			index:    portal.Index,
			start:    a0,
			end:      a1,
			distance: dist,
			length:   0,
			next:     invalidPortalIndex,
		})
	}
	sort.Sort(herringboneNodesByDistance(q.nodes))
	for i := 0; i < len(q.weights); i++ {
		q.weights[i] = 0
	}
	for i, node := range q.nodes {
		var bestLength uint16 = 1
		bestNext := invalidPortalIndex
		var bestWeight float32
		for j := 0; j < i; j++ {
			if q.nodes[j].start < node.start && q.nodes[j].end < node.end {
				if q.nodes[j].length >= bestLength {
					bestLength = q.nodes[j].length + 1
					bestNext = portalIndex(j)
					scaledDistance := float32(distance(q.portals[node.index], q.portals[q.nodes[j].index]) * RadiansToMeters)
					bestWeight = q.weights[q.nodes[j].index] + scaledDistance
				} else if q.nodes[j].length+1 == bestLength {
					scaledDistance := float32(distance(q.portals[node.index], q.portals[q.nodes[j].index]) * RadiansToMeters)
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
				distance(q.portals[node.index], b1)) * RadiansToMeters)
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
	if len(fixedBaseIndices) == 1 {
		numPairs = len(portals) - 1
	} else if len(fixedBaseIndices) == 2 {
		numPairs = 1
	}
	everyNth := numPairs / 1000
	if everyNth < 1 {
		everyNth = 1
	}
	numProcessedPairs := 0
	numProcessedPairsModN := 0
	progressFunc(0, numPairs)
	q := newBestHerringboneQuery(portalsData)
	for i, b0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			if !hasAllIndicesInThePair(fixedBaseIndices, i, j) {
				continue
			}
			b1 := portalsData[j]
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

func HerringbonePolyline(b0, b1 Portal, result []Portal) []Portal {
	portalList := []Portal{b0, b1}
	atIndex := 0
	for _, portal := range result {
		portalList = append(portalList, portal, portalList[atIndex])
		atIndex = 1 - atIndex
	}
	return portalList
}
func HerringboneDrawToolsString(b0, b1 Portal, result []Portal) string {
	return "[\n" + PolylineFromPortalList(HerringbonePolyline(b0, b1, result)) + "\n]"
}
