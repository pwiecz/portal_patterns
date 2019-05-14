package main

import "fmt"

import "math"
import "sort"
import "github.com/golang/geo/s1"
import "github.com/golang/geo/s2"
import "github.com/golang/geo/r3"

type node struct {
	index      uint16
	start, end s1.Angle
	distance   s1.Angle
	length     uint16
	next       uint16
}

type distanceQuery struct {
	aCrossB s2.Point
	c2      float64
}

func newDistanceQuery(a, b s2.Point) distanceQuery {
	aCrossB := a.PointCross(b)
	return distanceQuery{aCrossB, aCrossB.Norm2()}
}

func (d *distanceQuery) Distance(p s2.Point) s1.Angle {
	pDotC := p.Dot(d.aCrossB.Vector)
	pDotC2 := pDotC * pDotC
	cx := d.aCrossB.Cross(p.Vector)
	qr := 1 - math.Sqrt(cx.Norm2()/d.c2)
	return s1.ChordAngle((pDotC2 / d.c2) + (qr * qr)).Angle()

}

func angle(a, b s2.Point, v r3.Vector) s1.Angle {
	return a.PointCross(b).Angle(v)
}

func findBestHerringbone(b0, b1 portalData, portals []portalData, nodes []node, result []uint16) []uint16 {
	nodes = nodes[:0]
	v0, v1 := b1.LatLng.PointCross(b0.LatLng).Vector, b0.LatLng.PointCross(b1.LatLng).Vector
	distQuery := newDistanceQuery(b0.LatLng, b1.LatLng)
	for _, portal := range portals {
		if portal == b0 || portal == b1 {
			continue
		}
		if !s2.Sign(portal.LatLng, b0.LatLng, b1.LatLng) {
			continue
		}
		a0, a1 := angle(portal.LatLng, b0.LatLng, v0), angle(portal.LatLng, b1.LatLng, v1)
		dist := distQuery.Distance(portal.LatLng)
		nodes = append(nodes, node{portal.Index, a0, a1, dist, 0, invalidPortalIndex})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].distance < nodes[j].distance
	})
	for i, node := range nodes {
		var bestLength uint16
		bestNext := invalidPortalIndex
		for j := 0; j < i; j++ {
			if nodes[j].start < node.start && nodes[j].end < node.end {
				if nodes[j].length >= bestLength {
					bestLength = nodes[j].length + 1
					bestNext = uint16(j)
				}
			}
		}
		nodes[i].length = bestLength
		nodes[i].next = bestNext
	}

	start := invalidPortalIndex
	var length uint16
	for i, node := range nodes {
		if node.length > length {
			length = node.length
			start = uint16(i)
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

// LargestHerringbone - Find largest possible multilayer of portals to be made
func LargestHerringbone(portals []Portal) (Portal, Portal, []Portal) {
	if len(portals) < 3 {
		panic("Too short portal list")
	}
	portalsData := make([]portalData, 0, len(portals))
	for i, portal := range portals {
		portalsData = append(portalsData, portalData{Index: uint16(i), LatLng: portal.LatLng})
	}

	index := make([]bestSolution, len(portals))
	var largestHerringbone []uint16
	var bestB0, bestB1 portalData
	nodesCache := make([]node, 0, len(portals))
	resultCache := make([]uint16, 0, len(portals))

	numPairs := len(portals) * (len(portals) - 1) / 2
	everyNth := numPairs / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	numProcessedPairs := 0
	printProgressBar(0, numPairs)
	for i, b0 := range portalsData {
		for j := i + 1; j < len(portalsData); j++ {
			b1 := portalsData[j]
			for k := 0; k < len(index); k++ {
				index[k].Length = invalidLength
			}
			bestCCW := findBestHerringbone(b0, b1, portalsData, nodesCache, resultCache)
			if len(bestCCW) > len(largestHerringbone) {
				largestHerringbone = append(largestHerringbone[:0], bestCCW...)
				bestB0 = b0
				bestB1 = b1
			}
			bestCW := findBestHerringbone(b1, b0, portalsData, nodesCache, resultCache)
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
