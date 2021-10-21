package lib

import (
	"fmt"
	"sync"

	"github.com/golang/geo/r3"
)

type bestHerringboneMtQuery struct {
	portals []portalData
	// Array of normalized direction vectors between all the pairs of portals
	norms []r3.Vector
}

func newBestHerringboneMtQuery(portals []portalData) *bestHerringboneMtQuery {
	norms := make([]r3.Vector, len(portals)*len(portals))
	for i, p0 := range portals {
		for j, p1 := range portals {
			if i == j {
				continue
			}
			norms[i*len(portals)+j] = p1.LatLng.Cross(p0.LatLng.Vector).Normalize()

		}
	}
	return &bestHerringboneMtQuery{
		portals: portals,
		norms:   norms,
	}
}

type herringboneRequest struct {
	result []portalIndex
	p0     portalData
	p1     portalData
}

func (q *bestHerringboneMtQuery) findBestHerringbone(b0, b1 portalData, nodes []herringboneNode, weights []float32, result []portalIndex) []portalIndex {
	hq := bestHerringboneQuery{
		portals: q.portals,
		nodes:   nodes,
		weights: weights,
		norms:   q.norms,
	}
	return hq.findBestHerringbone(b0, b1, result)
}

func bestHerringboneWorker(
	q *bestHerringboneMtQuery,
	requestChannel, responseChannel chan herringboneRequest,
	wg *sync.WaitGroup) {
	nodes := make([]herringboneNode, 0, len(q.portals))
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
	if numWorkers < 1 {
		panic(fmt.Errorf("too few workers: %d", numWorkers))
	}
	if len(portals) < 3 {
		panic(fmt.Errorf("too short portal list: %d", len(portals)))
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
