package lib

import (
	"fmt"
	"sync"
)

type doubleHerringboneRequest struct {
	resultCCW []portalIndex
	resultCW  []portalIndex
	p0        portalData
	p1        portalData
}

func bestDoubleHerringboneWorker(
	q *bestHerringboneMtQuery,
	requestChannel, responseChannel chan doubleHerringboneRequest,
	doneChannel chan struct{}) {
	nodes := make([]herringboneNode, 0, len(q.portals))
	weights := make([]float32, len(q.portals))
	for req := range requestChannel {
		nodes = nodes[:0]
		for i := 0; i < len(weights); i++ {
			weights[i] = 0
		}
		req.resultCCW = q.findBestHerringbone(req.p0, req.p1, nodes, weights, req.resultCCW)
		req.resultCW = q.findBestHerringbone(req.p1, req.p0, nodes, weights, req.resultCW)
		responseChannel <- req
	}
	doneChannel <- struct{}{}
}

// LargestDoubleHerringboneMT - Find largest possible multilayer of portals to be made, parallel version
func LargestDoubleHerringboneMT(portals []Portal, fixedBaseIndices []int, numWorkers int, progressFunc func(int, int)) (Portal, Portal, []Portal, []Portal) {
	if numWorkers < 1 {
		panic(fmt.Errorf("too few workers: %d", numWorkers))
	}
	if len(portals) < 3 {
		panic(fmt.Errorf("too short portal list: %d", len(portals)))
	}
	portalsData := portalsToPortalData(portals)

	var largestCCW, largestCW []portalIndex
	var bestB0, bestB1 portalIndex
	resultCache := sync.Pool{
		New: func() interface{} {
			return make([]portalIndex, 0, len(portals))
		},
	}

	numPairs := len(portals) * (len(portals) - 1) / 2
	everyNth := numPairs / 1000
	if everyNth < 1 {
		everyNth = 1
	}
	numProcessedPairs := 0
	requestChannel := make(chan doubleHerringboneRequest, numWorkers)
	responseChannel := make(chan doubleHerringboneRequest, numWorkers)
	doneChannel := make(chan struct{}, numWorkers)
	q := newBestHerringboneMtQuery(portalsData)
	for i := 0; i < numWorkers; i++ {
		go bestDoubleHerringboneWorker(q, requestChannel, responseChannel, doneChannel)
	}
	go func() {
		for i, b0 := range portalsData {
			for j := i + 1; j < len(portalsData); j++ {
				b1 := portalsData[j]
				if !hasAllIndicesInThePair(fixedBaseIndices, i, j) {
					continue
				}
				requestChannel <- doubleHerringboneRequest{
					p0:        b0,
					p1:        b1,
					resultCCW: resultCache.Get().([]portalIndex),
					resultCW:  resultCache.Get().([]portalIndex),
				}
			}
		}
		close(requestChannel)
	}()
	progressFunc(0, numPairs)
	numWorkersDone := 0
	for numWorkersDone < numWorkers {
		select {
		case resp := <-responseChannel:
			if len(resp.resultCCW)+len(resp.resultCW) > len(largestCCW)+len(largestCW) {
				if len(largestCCW)+len(largestCW) > 0 {
					resultCache.Put(largestCCW)
					resultCache.Put(largestCW)
				}
				largestCCW = resp.resultCCW
				largestCW = resp.resultCW
				bestB0, bestB1 = resp.p0.Index, resp.p1.Index
			} else {
				resultCache.Put(resp.resultCCW)
				resultCache.Put(resp.resultCW)
			}
			numProcessedPairs++
			if numProcessedPairs%everyNth == 0 {
				progressFunc(numProcessedPairs, numPairs)
			}
		case <-doneChannel:
			numWorkersDone++
		}
	}
	progressFunc(numPairs, numPairs)
	close(responseChannel)
	close(doneChannel)
	resultCCW := make([]Portal, 0, len(largestCCW))
	for _, portalIx := range largestCCW {
		resultCCW = append(resultCCW, portals[portalIx])
	}
	resultCW := make([]Portal, 0, len(largestCW))
	for _, portalIx := range largestCW {
		resultCW = append(resultCW, portals[portalIx])
	}

	return portals[bestB0], portals[bestB1], resultCCW, resultCW
}
