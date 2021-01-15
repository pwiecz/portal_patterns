package lib

import (
	"fmt"
	"sync"
)

type longestDroneFlightMtQuery struct {
	neighbours     [][]droneFlightNeighbour
	portalDistance func(portalIndex, portalIndex) float64
}

type droneFlightResponse struct {
	path, keysNeeded []portalIndex
	distance         float64
}

func longestDroneFlightWorker(
	neighbours [][]droneFlightNeighbour,
	portalDistance func(portalIndex, portalIndex) float64,
	targetPortal portalIndex,
	requestChannel chan portalIndex, responseChannel chan droneFlightResponse,
	wg *sync.WaitGroup) {
	q := newLongestDroneFlightQuery(neighbours, portalDistance)
	for start := range requestChannel {
		path, keysNeeded, distance := q.longestFlightFrom(start, targetPortal)
		responseChannel <- droneFlightResponse{
			path:       path,
			keysNeeded: keysNeeded,
			distance:   distance,
		}
	}
	wg.Done()
}

func longestDroneFlightMT(portals []Portal, params droneFlightParams) ([]Portal, []Portal) {
	if params.numWorkers < 1 {
		panic(fmt.Errorf("Too few workers: %d", params.numWorkers))
	}
	if len(portals) < 2 {
		panic(fmt.Errorf("Too short portal list: %d", len(portals)))
	}
	portalsData := portalsToPortalData(portals)

	// If we have specified endIndex and not startIndex it's much faster to find best
	// route from the endIndex using reversed neighbours list, and later reverse the
	// result route.
	reverseRoute := false
	if params.startPortalIndex < 0 && params.endPortalIndex >= 0 {
		reverseRoute = true
		params.startPortalIndex, params.endPortalIndex = params.endPortalIndex, params.startPortalIndex
	}
	neighbours := prepareDroneGraph(portalsData, params.useLongJumps, reverseRoute)
	portalDistanceInRadians := func(i, j portalIndex) float64 {
		return distanceSq(portalsData[i], portalsData[j])
	}
	targetPortal := invalidPortalIndex
	if params.endPortalIndex >= 0 {
		targetPortal = portalIndex(params.endPortalIndex)
	}

	requestChannel := make(chan portalIndex, params.numWorkers)
	responseChannel := make(chan droneFlightResponse, params.numWorkers)

	var wg sync.WaitGroup
	wg.Add(params.numWorkers)

	for i := 0; i < params.numWorkers; i++ {
		go longestDroneFlightWorker(neighbours, portalDistanceInRadians, targetPortal,
			requestChannel, responseChannel, &wg)
	}
	go func() {
		for _, p := range portalsData {
			if params.startPortalIndex >= 0 && p.Index != portalIndex(params.startPortalIndex) {
				continue
			}
			requestChannel <- p.Index
		}
		close(requestChannel)
	}()
	go func() {
		wg.Wait()
		close(responseChannel)
	}()
	numIndexEntries := len(portals)
	everyNth := numIndexEntries / 1000
	if everyNth < 50 {
		everyNth = 2
	}
	indexEntriesFilled := 0
	indexEntriesFilledModN := 0
	params.progressFunc(0, numIndexEntries)

	bestDistance := -1.
	bestPath := []portalIndex{}
	bestKeysNeeded := []portalIndex{}

	for resp := range responseChannel {
		if resp.distance > bestDistance || (resp.distance == bestDistance && (len(resp.keysNeeded) < len(bestKeysNeeded) || (len(resp.keysNeeded) == len(bestKeysNeeded) && len(resp.path) < len(bestPath)))) {
			bestDistance = resp.distance
			bestKeysNeeded = resp.keysNeeded
			bestPath = resp.path
		}

		indexEntriesFilled++
		indexEntriesFilledModN++
		if indexEntriesFilledModN == everyNth {
			indexEntriesFilledModN = 0
			params.progressFunc(indexEntriesFilled, numIndexEntries)
		}
	}
	params.progressFunc(numIndexEntries, numIndexEntries)

	if reverseRoute {
		reverse(bestPath)
	}
	bestPortalPath := []Portal{}
	for i := len(bestPath) - 1; i >= 0; i-- {
		bestPortalPath = append(bestPortalPath, portals[bestPath[i]])
	}
	bestPortalKeysNeeded := []Portal{}
	for _, index := range bestKeysNeeded {
		bestPortalKeysNeeded = append(bestPortalKeysNeeded, portals[index])
	}
	return bestPortalPath, bestPortalKeysNeeded
}
